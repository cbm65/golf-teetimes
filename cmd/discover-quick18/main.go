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

// Quick18 Discovery Tool
//
// Probes courses by guessing Quick18 subdomains from the course name.
// Quick18 uses {subdomain}.quick18.com/teetimes/searchmatrix?teedate={YYYYMMDD}
//
// Example: Papago Golf Course → papago.quick18.com
//
// Usage: go run cmd/discover-quick18/main.go <state> -f <file>

var teeTimeRe = regexp.MustCompile(`mtrxTeeTimes">\s*(\d+:\d+)<div class="be_tee_time_ampm">(AM|PM)`)
var priceRe = regexp.MustCompile(`mtrxPrice">\$([0-9.]+)</div>`)

// stateNames maps abbreviation to full name for page-text matching
var stateFullNames = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas",
	"CA": "California", "CO": "Colorado", "CT": "Connecticut", "DE": "Delaware",
	"FL": "Florida", "GA": "Georgia", "HI": "Hawaii", "ID": "Idaho",
	"IL": "Illinois", "IN": "Indiana", "IA": "Iowa", "KS": "Kansas",
	"KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi",
	"MO": "Missouri", "MT": "Montana", "NE": "Nebraska", "NV": "Nevada",
	"NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico", "NY": "New York",
	"NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio", "OK": "Oklahoma",
	"OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah",
	"VT": "Vermont", "VA": "Virginia", "WA": "Washington", "WV": "West Virginia",
	"WI": "Wisconsin", "WY": "Wyoming",
}

// validateStateInHTML checks if the page HTML contains evidence of the target state.
// Looks for ", AZ" or ", Arizona" patterns near address/zip context.
// Returns true if state matches OR if no state info found (benefit of the doubt).
func validateStateInHTML(html, targetState string) (bool, string) {
	upper := strings.ToUpper(html)

	// Look for ", AZ 85" or ", AZ\n" style patterns (comma + state abbrev near zip)
	statePattern := regexp.MustCompile(`,\s*([A-Z]{2})\s*\d{5}`)
	matches := statePattern.FindAllStringSubmatch(upper, -1)
	for _, m := range matches {
		if m[1] == targetState {
			return true, m[1]
		}
		// Found a different state — reject
		if _, ok := stateFullNames[m[1]]; ok {
			return false, m[1]
		}
	}

	// Also try ", AZ " or ", AZ<" without zip
	statePattern2 := regexp.MustCompile(`,\s*([A-Z]{2})\s*[<\s]`)
	matches = statePattern2.FindAllStringSubmatch(upper, -1)
	for _, m := range matches {
		if m[1] == targetState {
			return true, m[1]
		}
		if _, ok := stateFullNames[m[1]]; ok {
			return false, m[1]
		}
	}

	// No state info found — allow (benefit of the doubt)
	return true, ""
}

type Result struct {
	Input        string   `json:"input"`
	City         string   `json:"city"`
	Status       string   `json:"status"` // "confirmed", "listed_only", "wrong_state", "miss"
	Subdomain    string   `json:"subdomain,omitempty"`
	FoundState   string   `json:"foundState,omitempty"`
	DatesChecked []string `json:"datesChecked,omitempty"`
	TeeTimes     []int    `json:"teeTimes,omitempty"`
	HasPrice     bool     `json:"hasPrice,omitempty"`
	SlugsTried   []string `json:"slugsTried,omitempty"`
}

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
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

// joinAlpha strips non-alphanumeric chars and lowercases
func joinAlpha(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// coreName strips common golf suffixes and "the"/"Golf Club of" prefixes
func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Golf Links", " Golf Center", " Country Club",
		" Golf & Country Club", " Golf and Country Club", " GC", " CC",
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
	return strings.TrimSpace(strings.ToLower(s))
}

// buildSubdomains generates candidate Quick18 subdomains from a course name.
// Quick18 subdomains are unpredictable — some use just core name, some full name,
// some hyphenated. We try many patterns to maximize hit rate.
func buildSubdomains(name string) []string {
	seen := map[string]bool{}
	var slugs []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		slugs = append(slugs, s)
	}

	core := joinAlpha(coreName(name))
	full := joinAlpha(name)

	// Core name: papago
	add(core)
	// Full joined: papagogolfcourse
	add(full)
	// Core + golf: papagogolf
	add(core + "golf")
	// Core + golfclub: papagogolfclub
	add(core + "golfclub")
	// Core + golfcourse: papagogolfcourse (may dup full, seen dedup handles)
	add(core + "golfcourse")
	// Core + cc: papagocc
	add(core + "cc")

	// Hyphenated full: papago-golf-course
	words := strings.Fields(strings.ToLower(name))
	cleaned := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ReplaceAll(w, "'", "")
		w = strings.ReplaceAll(w, "\u2019", "")
		w = strings.ReplaceAll(w, ".", "")
		if w != "" {
			cleaned = append(cleaned, w)
		}
	}
	add(strings.Join(cleaned, "-"))

	// Hyphenated core: papago
	coreWords := strings.Fields(coreName(name))
	if len(coreWords) > 1 {
		add(strings.Join(coreWords, "-"))
	}

	// Without "the": phoenician instead of thephoenician
	withoutThe := strings.TrimPrefix(full, "the")
	if withoutThe != full {
		add(withoutThe)
		add(withoutThe + "golf")
	}
	coreWithoutThe := strings.TrimPrefix(core, "the")
	if coreWithoutThe != core {
		add(coreWithoutThe)
	}

	return slugs
}

// probeSubdomain checks if a Quick18 subdomain exists and has tee time content
func probeSubdomain(client *http.Client, subdomain, date string) (int, bool) {
	dateClean := strings.ReplaceAll(date, "-", "")
	url := fmt.Sprintf("https://%s.quick18.com/teetimes/searchmatrix?teedate=%s", subdomain, dateClean)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.ReadAll(resp.Body)
		return 0, false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false
	}

	html := string(body)

	// Count tee times
	times := teeTimeRe.FindAllString(html, -1)
	count := len(times)

	// Check for prices
	prices := priceRe.FindAllString(html, -1)
	hasPrice := len(prices) > 0

	return count, hasPrice
}

// findSubdomain tries subdomains to find one that returns a valid Quick18 page.
// Returns (subdomain, teeTimeCount, hasPrice, pageHTML).
func findSubdomain(client *http.Client, subdomains []string, date string, deadCache map[string]bool) (string, int, bool, string) {
	for _, sub := range subdomains {
		dateClean := strings.ReplaceAll(date, "-", "")
		url := fmt.Sprintf("https://%s.quick18.com/teetimes/searchmatrix?teedate=%s", sub, dateClean)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

		resp, err := client.Do(req)
		if err != nil {
			deadCache[sub] = true
			continue
		}

		if resp.StatusCode != 200 {
			io.ReadAll(resp.Body)
			resp.Body.Close()
			deadCache[sub] = true
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		html := string(body)

		// A valid Quick18 page has the search matrix content
		if !strings.Contains(html, "mtrxTeeTimes") && !strings.Contains(html, "searchmatrix") && len(html) < 2000 {
			deadCache[sub] = true
			continue
		}

		// Found a valid page — count tee times
		times := teeTimeRe.FindAllString(html, -1)
		prices := priceRe.FindAllString(html, -1)
		return sub, len(times), len(prices) > 0, html
	}
	return "", 0, false, ""
}

type CourseEntry struct {
	Name string
	City string
}

func readCoursesFromFile(path string) ([]CourseEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var courses []CourseEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		name := strings.TrimSpace(parts[0])
		city := ""
		if len(parts) > 1 {
			city = strings.TrimSpace(parts[1])
		}
		if name != "" {
			courses = append(courses, CourseEntry{Name: name, City: city})
		}
	}
	return courses, scanner.Err()
}

func main() {
	if len(os.Args) < 4 || os.Args[2] != "-f" {
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-quick18/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run cmd/discover-quick18/main.go AZ -f discovery/courses/phoenix.txt\n")
		os.Exit(1)
	}

	state := strings.ToUpper(os.Args[1])
	courses, err := readCoursesFromFile(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	startTime := time.Now()
	dates := probeDates()

	log("=== Quick18 Discovery ===")
	log("State: %s", state)
	log("Courses to probe: %d", len(courses))
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	client := &http.Client{Timeout: 10 * time.Second}

	var results []Result
	confirmed, missed, listedOnly, wrongState := 0, 0, 0, 0
	deadSubdomains := map[string]bool{}

	for i, c := range courses {
		subdomains := buildSubdomains(c.Name)

		// Filter out known-dead subdomains
		var live []string
		for _, s := range subdomains {
			if !deadSubdomains[s] {
				live = append(live, s)
			}
		}

		// Phase 1: Find a valid subdomain using the first probe date
		foundSub, initialCount, _, pageHTML := findSubdomain(client, live, dates[0], deadSubdomains)

		if foundSub == "" {
			missed++
			log("[%d/%d] %-45s  ❌ miss  (tried: %s)", i+1, len(courses), c.Name, strings.Join(subdomains, ", "))
			results = append(results, Result{
				Input:      c.Name,
				City:       c.City,
				Status:     "miss",
				SlugsTried: subdomains,
			})
			continue
		}

		log("[%d/%d] %-45s  🔍 found: %s.quick18.com", i+1, len(courses), c.Name, foundSub)

		// State validation: check if page HTML contains evidence of the target state
		stateOK, foundState := validateStateInHTML(pageHTML, state)
		if !stateOK {
			wrongState++
			log("[%d/%d] %-45s  ⚠️  WRONG STATE — found %q, wanted %s", i+1, len(courses), c.Name, foundState, state)
			results = append(results, Result{
				Input:      c.Name,
				City:       c.City,
				Status:     "wrong_state",
				Subdomain:  foundSub,
				FoundState: foundState,
			})
			continue
		}

		// Phase 2: 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := initialCount
		anyPrice := false

		// Already have first date from the probe
		datesChecked = append(datesChecked, dates[0])
		teeTimes = append(teeTimes, initialCount)

		// Probe remaining dates
		for _, date := range dates[1:] {
			count, hasPrice := probeSubdomain(client, foundSub, date)
			datesChecked = append(datesChecked, date)
			teeTimes = append(teeTimes, count)
			totalTimes += count
			if hasPrice {
				anyPrice = true
			}
			time.Sleep(300 * time.Millisecond)
		}

		if totalTimes > 0 {
			confirmed++
			log("[%d/%d] %-45s  ✅ confirmed  %s.quick18.com  times=%v",
				i+1, len(courses), c.Name, foundSub, teeTimes)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "confirmed",
				Subdomain: foundSub, DatesChecked: datesChecked,
				TeeTimes: teeTimes, HasPrice: anyPrice,
			})
		} else {
			listedOnly++
			log("[%d/%d] %-45s  📋 listed (0 times)  %s.quick18.com",
				i+1, len(courses), c.Name, foundSub)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "listed_only",
				Subdomain: foundSub, DatesChecked: datesChecked,
				TeeTimes: teeTimes,
			})
		}

		time.Sleep(200 * time.Millisecond)
	}

	elapsed := time.Since(startTime)

	log("")
	log("=== Results ===")
	log("Total: %d", len(courses))
	log("Confirmed: %d", confirmed)
	log("Listed Only: %d", listedOnly)
	log("Wrong State: %d", wrongState)
	log("Missed: %d", missed)
	log("Elapsed: %s", elapsed.Round(time.Second))
	log("")

	if confirmed > 0 {
		log("=== Confirmed ===")
		for _, r := range results {
			if r.Status == "confirmed" {
				log("  ✅ %-45s  %s.quick18.com  times=%v",
					r.Input, r.Subdomain, r.TeeTimes)
			}
		}
		log("")
	}

	if listedOnly > 0 {
		log("=== Listed Only ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  📋 %-45s  %s.quick18.com", r.Input, r.Subdomain)
			}
		}
		log("")
	}

	if wrongState > 0 {
		log("=== Wrong State ===")
		for _, r := range results {
			if r.Status == "wrong_state" {
				log("  ⚠️  %-45s  %s.quick18.com → state=%s (wanted %s)", r.Input, r.Subdomain, r.FoundState, state)
			}
		}
		log("")
	}

	if missed > 0 {
		log("=== Missed ===")
		for _, r := range results {
			if r.Status == "miss" {
				log("  ❌ %-45s  (tried: %s)", r.Input, strings.Join(r.SlugsTried, ", "))
			}
		}
		log("")
	}

	// Save results
	os.MkdirAll("discovery/results", 0755)
	ts := time.Now().Format("2006-01-02-150405")
	outPath := fmt.Sprintf("discovery/results/quick18-%s-%s.json", strings.ToLower(state), ts)

	output := map[string]any{
		"platform":   "quick18",
		"state":      state,
		"timestamp":  time.Now().Format(time.RFC3339),
		"elapsed":    elapsed.String(),
		"total":      len(courses),
		"confirmed":  confirmed,
		"listedOnly": listedOnly,
		"wrongState": wrongState,
		"missed":     missed,
		"results":    results,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, data, 0644)
	log("Results saved to %s", outPath)
}
