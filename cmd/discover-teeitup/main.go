package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Discovery tool for TeeItUp courses in a target region.
//
// Usage: go run cmd/discover-teeitup/main.go <state> <course name> [course name] ...
//   or: go run cmd/discover-teeitup/main.go <state> -f <file>
//
// The file should have one course name per line (blank lines and # comments ignored).
// Format: "Course Name | City" (city is optional but used for validation).
//
// Example:
//   go run cmd/discover-teeitup/main.go AZ -f discovery/courses/phoenix.txt
//
// Discovery approach:
//   1. For each course, generates multiple alias candidates (suffix swaps, core name, etc.)
//   2. Probes Kenna /facilities endpoint with x-be-alias header for each candidate
//   3. On hit: validates state, fuzzy-matches facility name, checks siblings
//   4. Validates with tee time checks on 3 dates (Wed, Sat, Sat+7)
//   5. Deduplicates by facility ID to avoid double-counting
//
// Results are saved to discovery/results/teeitup-{state}-{timestamp}.json

var apiBase = "https://phx-api-be-east-1b.kenna.io"

type Facility struct {
	ID       int       `json:"id"`
	CourseID string    `json:"courseId"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Locality string    `json:"locality"`
	Region   string    `json:"region"`
	Country  string    `json:"country"`
	Location []float64 `json:"location"`
	TimeZone string    `json:"timeZone"`
}

type TeeTimeResponse struct {
	Teetimes []struct {
		Teetime string `json:"teetime"`
	} `json:"teetimes"`
}

type Result struct {
	Input        string    `json:"input"`
	City         string    `json:"city,omitempty"`
	Alias        string    `json:"alias"`
	AliasSource  string    `json:"aliasSource"` // which pattern matched
	Status       string    `json:"status"`      // "confirmed", "listed_only", "miss", "error"
	Facility     *Facility `json:"facility,omitempty"`
	Error        string    `json:"error,omitempty"`
	DatesChecked []string  `json:"datesChecked,omitempty"`
	TeeTimes     []int     `json:"teeTimes,omitempty"`
}

type CourseInput struct {
	Name string
	City string
}

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// coreName strips common golf suffixes/prefixes to get the core name.
func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Golf Links", " Golf Center", " Country Club", " Golf & Country Club",
		" Golf and Country Club", " GC", " CC",
	} {
		if strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix)) {
			s = s[:len(s)-len(suffix)]
			break
		}
	}
	for _, prefix := range []string{"The ", "Golf Club of ", "Golf Club at "} {
		if strings.HasPrefix(s, prefix) {
			s = s[len(prefix):]
			break
		}
	}
	return strings.TrimSpace(s)
}

// generateAliases produces multiple alias candidates for a course name.
// Based on observed real-world alias patterns:
//   - "stonecreek-golf-club" (exact slug)
//   - "raven-golf-club-phoenix" (full slug + city for disambiguation)
//   - "continental-country-club" (different suffix than expected)
//   - "ballwin-gc-public-booking-engine" (unusual suffix patterns)
//   - "city-of-phoenix-golf-courses" (group aliases — can't be generated, found via sibling matching)
func generateAliases(name, city string) []struct{ alias, source string } {
	var candidates []struct{ alias, source string }
	seen := map[string]bool{}

	add := func(alias, source string) {
		if alias == "" || seen[alias] {
			return
		}
		seen[alias] = true
		candidates = append(candidates, struct{ alias, source string }{alias, source})
	}

	exact := slugify(name)
	core := slugify(coreName(name))

	// 1. Exact name slug (highest priority)
	add(exact, "exact")

	// 2. Without "the-" prefix if present, or add it if missing
	if strings.HasPrefix(exact, "the-") {
		add(strings.TrimPrefix(exact, "the-"), "no-the")
	} else {
		add("the-"+exact, "add-the")
	}

	// 3. Suffix swaps — the alias suffix often doesn't match the course's actual name
	//    e.g. "Continental Golf Course" might be "continental-country-club" on TeeItUp
	suffixes := []string{
		"golf-course", "golf-club", "country-club", "golf-resort",
		"golf-complex", "golf-links", "golf-center", "gc", "golf",
	}
	for _, oldSuffix := range suffixes {
		if strings.HasSuffix(exact, "-"+oldSuffix) {
			base := strings.TrimSuffix(exact, "-"+oldSuffix)
			for _, newSuffix := range suffixes {
				if newSuffix != oldSuffix {
					add(base+"-"+newSuffix, "swap-"+newSuffix)
				}
			}
			// Base alone (no suffix)
			add(base, "base-only")
			break
		}
	}

	// 4. Core name alone (strips prefixes like "The", "Golf Club of")
	add(core, "core")

	// 5. Core + common suffixes — always try these since the core name
	//    with a different suffix is a common alias pattern
	//    e.g. "Golf Club of Estrella" -> try "estrella-golf-course", "estrella-golf-club"
	for _, suffix := range []string{
		"golf-course", "golf-club", "country-club", "golf-resort", "golf",
	} {
		add(core+"-"+suffix, "core+"+suffix)
	}

	// 6. City-qualified variants — used for disambiguation
	//    e.g. "raven-golf-club-phoenix", "continental-golf-course-scottsdale"
	if city != "" {
		citySlug := slugify(city)

		// Strip city from beginning of name (e.g. "Scottsdale Silverado" → "silverado")
		cityLower := strings.ToLower(city)
		coreLower := strings.ToLower(coreName(name))
		if strings.HasPrefix(coreLower, cityLower+" ") {
			stripped := slugify(coreLower[len(cityLower)+1:])
			add(stripped, "strip-city")
			for _, suffix := range []string{"golf-club", "golf-course", "country-club", "golf"} {
				add(stripped+"-"+suffix, "strip-city+"+suffix)
			}
		}

		// Full slug + city (the raven-golf-club-phoenix pattern)
		add(exact+"-"+citySlug, "exact+city")

		// Core + city
		add(core+"-"+citySlug, "core+city")

		// Core + suffix + city
		add(core+"-golf-course-"+citySlug, "core-gc+city")
		add(core+"-golf-club-"+citySlug, "core-club+city")

		// City + core
		add(citySlug+"-"+core, "city+core")
		add(citySlug+"-"+core+"-golf-course", "city+core-gc")
		add(citySlug+"-"+core+"-golf-club", "city+core-club")
	}

	// 7. Public booking engine pattern (seen: ballwin-gc-public-booking-engine)
	add(core+"-public-booking-engine", "core+pbe")
	add(exact+"-public-booking-engine", "exact+pbe")
	add(core+"-gc-public-booking-engine", "core-gc+pbe")

	return candidates
}

// normalize for name comparison.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, suffix := range []string{
		"golf course", "golf club", "golf resort", "golf complex",
		"golf links", "golf center", "country club", " gc", " cc",
	} {
		s = strings.TrimSuffix(s, " "+suffix)
	}
	for _, prefix := range []string{"the ", "golf club of ", "golf club at "} {
		s = strings.TrimPrefix(s, prefix)
	}
	return strings.TrimSpace(s)
}

// fuzzyMatch checks if an input course name matches a Kenna facility name.
// After normalizing (stripping golf suffixes/prefixes), it checks:
//   1. Exact normalized match
//   2. One contains the other (min 5 chars to avoid "mesa" matching "mesa verde")
//   3. Significant word overlap — all significant words (2+) from the shorter name
//      appear in the longer (handles "Raven" matching "Raven at South Mountain")
func fuzzyMatch(inputName, facilityName string) bool {
	return fuzzyMatchWithCity(inputName, facilityName, "")
}

func fuzzyMatchWithCity(inputName, facilityName, city string) bool {
	a := normalize(inputName)
	b := normalize(facilityName)
	if a == "" || b == "" {
		return false
	}
	// Strip city name from input if present (e.g. "Raven Golf Club Phoenix" -> "raven" after normalize strips "golf club")
	// The city often gets appended to course names for disambiguation but isn't part of the Kenna facility name
	if city != "" {
		cityLower := strings.ToLower(strings.TrimSpace(city))
		a = strings.TrimSpace(strings.ReplaceAll(a, cityLower, ""))
	}
	if a == "" {
		return false
	}
	if a == b {
		return true
	}
	// Containment check with min length
	shorter, longer := a, b
	if len(a) > len(b) {
		shorter, longer = b, a
	}
	if len(shorter) >= 5 && strings.Contains(longer, shorter) {
		return true
	}
	// Word overlap: check if all significant words from the shorter name
	// appear in the longer name. Requires 2+ significant words to avoid
	// false positives like "mesa" matching "mesa verde".
	skip := map[string]bool{"at": true, "of": true, "the": true, "in": true, "and": true, "a": true}
	shorterWords := strings.Fields(shorter)
	longerWords := strings.Fields(longer)
	longerSet := map[string]bool{}
	for _, w := range longerWords {
		longerSet[w] = true
	}
	matchCount := 0
	totalCount := 0
	for _, w := range shorterWords {
		if skip[w] {
			continue
		}
		totalCount++
		if longerSet[w] {
			matchCount++
		}
	}
	if totalCount >= 2 && matchCount == totalCount {
		return true
	}
	return false
}

func probeDates() []string {
	now := time.Now()
	var dates []string

	d := now
	for d.Weekday() != time.Wednesday {
		d = d.AddDate(0, 0, 1)
	}
	dates = append(dates, d.Format("2006-01-02"))

	d = now
	for d.Weekday() != time.Saturday {
		d = d.AddDate(0, 0, 1)
	}
	dates = append(dates, d.Format("2006-01-02"))
	dates = append(dates, d.AddDate(0, 0, 7).Format("2006-01-02"))

	return dates
}

func probeFacility(alias string) ([]Facility, int, error) {
	req, err := http.NewRequest("GET", apiBase+"/facilities", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("x-be-alias", alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+alias+".book.teeitup.com")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, nil
	}

	var facilities []Facility
	if err := json.Unmarshal(body, &facilities); err != nil {
		return nil, resp.StatusCode, nil
	}
	return facilities, resp.StatusCode, nil
}

func probeTeeTimes(alias string, facilityID int, date string) (int, error) {
	url := fmt.Sprintf("%s/v2/tee-times?date=%s&facilityIds=%d&dateMax=%s", apiBase, date, facilityID, date)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-be-alias", alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+alias+".book.teeitup.com")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		return 0, nil
	}

	var data []TeeTimeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, nil
	}

	total := 0
	for _, d := range data {
		total += len(d.Teetimes)
	}
	return total, nil
}

func readInputFromFile(path string) ([]CourseInput, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var inputs []CourseInput
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		name := strings.TrimSpace(parts[0])
		city := ""
		if len(parts) > 1 {
			city = strings.TrimSpace(parts[1])
		}
		if name != "" {
			inputs = append(inputs, CourseInput{Name: name, City: city})
		}
	}
	return inputs, scanner.Err()
}

// validateAndRecord checks state, probes tee times, records the result.
func validateAndRecord(
	f Facility, alias, aliasSource, inputName, inputCity, state string,
	dates []string,
	results *[]Result, discoveredByFID map[int]bool, discoveredByName map[string]bool,
	confirmedCount, listedOnlyCount *int,
) {
	if discoveredByFID[f.ID] || discoveredByName[strings.ToLower(inputName)] {
		return
	}
	if !strings.EqualFold(f.Region, state) {
		return
	}

	if inputCity != "" && !strings.EqualFold(strings.TrimSpace(f.Locality), strings.TrimSpace(inputCity)) {
		log("    ⚠️  City mismatch: expected %q, got %q — continuing anyway", inputCity, f.Locality)
	}

	log("  FACILITY FOUND — FID:%d  %s (%s, %s)  via alias %q [%s]", f.ID, f.Name, f.Locality, f.Region, alias, aliasSource)

	var datesChecked []string
	var teeTimes []int
	totalTimes := 0

	for _, date := range dates {
		count, err := probeTeeTimes(alias, f.ID, date)
		if err != nil {
			log("    ERROR checking %s: %v", date, err)
			count = 0
		}
		datesChecked = append(datesChecked, date)
		teeTimes = append(teeTimes, count)
		totalTimes += count
		log("    %s: %d tee times", date, count)
		time.Sleep(200 * time.Millisecond)
	}

	status := "listed_only"
	if totalTimes > 0 {
		status = "confirmed"
		log("  ✅ CONFIRMED — %d total tee times", totalTimes)
		*confirmedCount++
	} else {
		log("  ⚠️  LISTED ONLY — 0 tee times")
		*listedOnlyCount++
	}

	fCopy := f
	*results = append(*results, Result{
		Input:        inputName,
		City:         inputCity,
		Alias:        alias,
		AliasSource:  aliasSource,
		Status:       status,
		Facility:     &fCopy,
		DatesChecked: datesChecked,
		TeeTimes:     teeTimes,
	})
	discoveredByFID[f.ID] = true
	discoveredByName[strings.ToLower(inputName)] = true
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-teeitup/main.go <state> <course name> [course name] ...\n")
		fmt.Fprintf(os.Stderr, "   or: go run cmd/discover-teeitup/main.go <state> -f <file>\n")
		os.Exit(1)
	}

	state := strings.ToUpper(os.Args[1])
	var inputs []CourseInput
	if os.Args[2] == "-f" {
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Missing file path after -f\n")
			os.Exit(1)
		}
		var err error
		inputs, err = readInputFromFile(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	} else {
		for _, name := range os.Args[2:] {
			inputs = append(inputs, CourseInput{Name: name})
		}
	}

	startTime := time.Now()
	dates := probeDates()

	log("=== TeeItUp Discovery ===")
	log("Target state: %s", state)
	log("Courses to probe: %d", len(inputs))
	log("API base: %s", apiBase)
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	var results []Result
	var confirmedCount, listedOnlyCount, missCount, errorCount int

	discoveredByFID := map[int]bool{}
	discoveredByName := map[string]bool{}
	aliasCache := map[string][]Facility{}
	// Track aliases that returned "Booking Engine Settings not found" so we don't retry
	deadAliases := map[string]bool{}

	for i, input := range inputs {
		if discoveredByName[strings.ToLower(input.Name)] {
			log("[%d/%d] %q — SKIPPED (already discovered)", i+1, len(inputs), input.Name)
			log("")
			continue
		}

		candidates := generateAliases(input.Name, input.City)
		log("[%d/%d] %q (city: %q) — %d alias candidates", i+1, len(inputs), input.Name, input.City, len(candidates))

		found := false
		for _, c := range candidates {
			if deadAliases[c.alias] {
				continue
			}

			// Check cache
			facilities, cached := aliasCache[c.alias]
			if !cached {
				var statusCode int
				var err error
				facilities, statusCode, err = probeFacility(c.alias)
				if err != nil {
					log("  %s (%s): ERROR %v", c.alias, c.source, err)
					time.Sleep(200 * time.Millisecond)
					continue
				}
				if statusCode != 200 || len(facilities) == 0 {
					deadAliases[c.alias] = true
					time.Sleep(100 * time.Millisecond)
					continue
				}
				aliasCache[c.alias] = facilities
				log("  %s (%s): HIT — %d facility(ies)", c.alias, c.source, len(facilities))
				for _, f := range facilities {
					log("    FID:%d  %s  (%s, %s)", f.ID, f.Name, f.Locality, f.Region)
				}
			}

			// Try to match the triggering course
			for _, f := range facilities {
				if fuzzyMatchWithCity(input.Name, f.Name, input.City) {
					validateAndRecord(f, c.alias, c.source, input.Name, input.City, state, dates,
						&results, discoveredByFID, discoveredByName, &confirmedCount, &listedOnlyCount)
					if discoveredByName[strings.ToLower(input.Name)] {
						found = true
					}
				}
			}

			// Multi-facility: check siblings against ALL inputs
			if len(facilities) > 1 {
				for _, f := range facilities {
					if discoveredByFID[f.ID] {
						continue
					}
					for _, other := range inputs {
						if discoveredByName[strings.ToLower(other.Name)] {
							continue
						}
						if fuzzyMatchWithCity(other.Name, f.Name, other.City) {
							log("  ↳ Sibling match: FID:%d %q ↔ input %q", f.ID, f.Name, other.Name)
							validateAndRecord(f, c.alias, c.source+"/sibling", other.Name, other.City, state, dates,
								&results, discoveredByFID, discoveredByName, &confirmedCount, &listedOnlyCount)
						}
					}
					// Log unmatched facilities
					if !discoveredByFID[f.ID] && strings.EqualFold(f.Region, state) {
						log("  ↳ Unmatched: FID:%d %q (%s) — not in input list", f.ID, f.Name, f.Locality)
					}
				}
			}

			if found {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if !found && !discoveredByName[strings.ToLower(input.Name)] {
			log("  MISS — no alias matched")
			results = append(results, Result{Input: input.Name, City: input.City, Alias: slugify(input.Name), Status: "miss"})
			missCount++
		}

		log("")
	}

	elapsed := time.Since(startTime)

	log("========================================")
	log("=== SUMMARY")
	log("========================================")
	log("Total probed:  %d", len(inputs))
	log("Confirmed:     %d  (facility + tee times)", confirmedCount)
	log("Listed only:   %d  (facility but no tee times)", listedOnlyCount)
	log("Misses:        %d", missCount)
	log("Errors:        %d", errorCount)
	log("Elapsed:       %s", elapsed.Round(time.Millisecond))
	log("")

	if confirmedCount > 0 {
		log("=== CONFIRMED ===")
		for _, r := range results {
			if r.Status == "confirmed" {
				total := 0
				for _, t := range r.TeeTimes {
					total += t
				}
				log("  %-40s alias:%-35s [%s] FID:%-6d %s (%s, %s)  [%d times]",
					r.Input, r.Alias, r.AliasSource, r.Facility.ID, r.Facility.Name, r.Facility.Locality, r.Facility.Region, total)
			}
		}
		log("")
	}

	if listedOnlyCount > 0 {
		log("=== LISTED ONLY ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  %-40s alias:%-35s [%s] FID:%-6d %s (%s, %s)",
					r.Input, r.Alias, r.AliasSource, r.Facility.ID, r.Facility.Name, r.Facility.Locality, r.Facility.Region)
			}
		}
		log("")
	}

	if missCount > 0 {
		log("=== MISSES ===")
		for _, r := range results {
			if r.Status == "miss" {
				log("  %-40s (city: %s)", r.Input, r.City)
			}
		}
		log("")
	}

	// Save results
	os.MkdirAll("discovery/results", 0755)
	filename := fmt.Sprintf("discovery/results/teeitup-%s-%s.json",
		strings.ToLower(state), startTime.Format("2006-01-02-150405"))

	output := map[string]any{
		"platform":        "teeitup",
		"state":           state,
		"timestamp":       startTime.Format(time.RFC3339),
		"elapsed":         elapsed.String(),
		"totalInput":      len(inputs),
		"confirmed":       confirmedCount,
		"listedOnly":      listedOnlyCount,
		"misses":          missCount,
		"errors":          errorCount,
		"validationDates": dates,
		"results":         results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log("ERROR saving results: %v", err)
		return
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log("ERROR writing file: %v", err)
		return
	}
	log("Results saved to %s", filename)
}
