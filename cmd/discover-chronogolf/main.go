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

// Chronogolf Discovery Tool
//
// Probes courses by constructing URL slugs and fetching the club page at
// https://www.chronogolf.com/club/{slug}. The embedded __NEXT_DATA__ contains
// club_id, course UUIDs, affiliation_type_id — everything needed for tee time queries.
//
// Usage: go run cmd/discover-chronogolf/main.go <state> -f <file>
//    or: go run cmd/discover-chronogolf/main.go <state> "Course Name"
//
// The file should have lines like: Course Name | City
//
// Slug patterns tried (in order):
//   1. name-state-city       (e.g. "papago-golf-course-arizona-phoenix")
//   2. name                  (e.g. "papago-golf-course")
//   3. Suffix swaps          (e.g. "papago-golf-club", "papago-country-club")
//   4. Core + suffixes       (e.g. "papago-golf", "papago")
//   5. City-first variants   (e.g. "phoenix-papago-golf-course")

var stateNames = map[string]string{
	"AL": "alabama", "AK": "alaska", "AZ": "arizona", "AR": "arkansas",
	"CA": "california", "CO": "colorado", "CT": "connecticut", "DE": "delaware",
	"FL": "florida", "GA": "georgia", "HI": "hawaii", "ID": "idaho",
	"IL": "illinois", "IN": "indiana", "IA": "iowa", "KS": "kansas",
	"KY": "kentucky", "LA": "louisiana", "ME": "maine", "MD": "maryland",
	"MA": "massachusetts", "MI": "michigan", "MN": "minnesota", "MS": "mississippi",
	"MO": "missouri", "MT": "montana", "NE": "nebraska", "NV": "nevada",
	"NH": "new-hampshire", "NJ": "new-jersey", "NM": "new-mexico", "NY": "new-york",
	"NC": "north-carolina", "ND": "north-dakota", "OH": "ohio", "OK": "oklahoma",
	"OR": "oregon", "PA": "pennsylvania", "RI": "rhode-island", "SC": "south-carolina",
	"SD": "south-dakota", "TN": "tennessee", "TX": "texas", "UT": "utah",
	"VT": "vermont", "VA": "virginia", "WA": "washington", "WV": "west-virginia",
	"WI": "wisconsin", "WY": "wyoming",
}

var nextDataRe = regexp.MustCompile(`<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

type ClubData struct {
	UUID                     string       `json:"uuid"`
	ID                       int          `json:"id"`
	Name                     string       `json:"name"`
	Slug                     string       `json:"slug"`
	City                     string       `json:"city"`
	Province                 string       `json:"province"`
	Country                  string       `json:"country"`
	Address                  string       `json:"address"`
	Phone                    string       `json:"phone"`
	Website                  string       `json:"website"`
	Postcode                 string       `json:"postcode"`
	DefaultAffiliationTypeId int          `json:"defaultAffiliationTypeId"`
	Location                 LatLon       `json:"location"`
	Courses                  []CourseData `json:"courses"`
}

type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type CourseData struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Holes int    `json:"holes"`
	UUID  string `json:"uuid,omitempty"`
}

type Result struct {
	Input        string    `json:"input"`
	City         string    `json:"city"`
	Slug         string    `json:"slug"`
	SlugSource   string    `json:"slugSource"`
	Status       string    `json:"status"` // "confirmed", "listed_only", "wrong_state", "wrong_city", "miss"
	Club         *ClubData `json:"club,omitempty"`
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
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteRune('-')
		}
	}
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}

func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Country Club", " Golf Links", " Golf Center", " Golf & Country Club",
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

func buildSlugs(name, stateFull, city string) []struct{ slug, source string } {
	seen := map[string]bool{}
	var slugs []struct{ slug, source string }
	add := func(raw, source string) {
		s := slugify(raw)
		if s != "" && !seen[s] {
			seen[s] = true
			slugs = append(slugs, struct{ slug, source string }{s, source})
		}
	}

	core := coreName(name)

	// 1. Full name + state + city (Chronogolf's primary slug pattern)
	if city != "" {
		add(name+" "+stateFull+" "+city, "name+state+city")
	}

	// 2. Full name + state (no city)
	add(name+" "+stateFull, "name+state")

	// 3. Full name only
	add(name, "name")

	// 4. Without "The " prefix
	if strings.HasPrefix(name, "The ") {
		add(strings.TrimPrefix(name, "The "), "no-the")
		if city != "" {
			add(strings.TrimPrefix(name, "The ")+" "+stateFull+" "+city, "no-the+state+city")
		}
	}

	// 5. Suffix swaps
	suffixes := []string{"golf-course", "golf-club", "country-club", "golf-resort", "golf-links", "golf-center"}
	exactSlug := slugify(name)
	for _, oldSuffix := range suffixes {
		if strings.HasSuffix(exactSlug, "-"+oldSuffix) {
			base := strings.TrimSuffix(exactSlug, "-"+oldSuffix)
			for _, newSuffix := range suffixes {
				if newSuffix != oldSuffix {
					add(base+"-"+newSuffix, "swap-"+newSuffix)
				}
			}
			break
		}
	}

	// 6. Core name + common suffixes
	add(core+" golf club", "core+club")
	add(core+" golf course", "core+course")
	add(core+" golf resort", "core+resort")
	add(core+" golf", "core+golf")

	// 7. Core name bare
	add(core, "core")

	// 8. Core + state + city
	if city != "" {
		add(core+" "+stateFull+" "+city, "core+state+city")
	}

	// 9. Strip "and" from name (e.g. "Littleton Golf and Tennis Club" → "littleton-golf-tennis-club")
	noAnd := strings.ReplaceAll(name, " and ", " ")
	if noAnd != name {
		if city != "" {
			add(noAnd+" "+stateFull+" "+city, "no-and+state+city")
		}
		add(noAnd+" "+stateFull, "no-and+state")
		add(noAnd, "no-and")
	}

	return slugs
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

func probeSlug(client *http.Client, slug string) (*ClubData, error) {
	url := fmt.Sprintf("https://www.chronogolf.com/club/%s", slug)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("no __NEXT_DATA__ found")
	}

	var nextData struct {
		Props struct {
			PageProps struct {
				Club ClubData `json:"club"`
			} `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &nextData); err != nil {
		return nil, fmt.Errorf("parse __NEXT_DATA__: %w", err)
	}

	club := &nextData.Props.PageProps.Club
	if club.ID == 0 {
		return nil, fmt.Errorf("empty club data")
	}

	return club, nil
}

func fetchTeeTimeCount(client *http.Client, courseUUIDs []string, date string) (int, error) {
	ids := strings.Join(courseUUIDs, ",")
	url := fmt.Sprintf("https://www.chronogolf.com/marketplace/v2/teetimes?start_date=%s&course_ids=%s&holes=9,18&page=1", date, ids)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.ReadAll(resp.Body)
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var data struct {
		TeeTimes []json.RawMessage `json:"teetimes"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	return len(data.TeeTimes), nil
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

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-chronogolf/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "   or: go run cmd/discover-chronogolf/main.go <state> \"Course Name\"\n")
		os.Exit(1)
	}

	state := strings.ToUpper(os.Args[1])
	stateFull, ok := stateNames[state]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown state: %s\n", state)
		os.Exit(1)
	}

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

	log("=== Chronogolf Discovery ===")
	log("State: %s (%s)", state, stateFull)
	log("Courses to probe: %d", len(inputs))
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	client := &http.Client{Timeout: 15 * time.Second}

	var results []Result
	var confirmedCount, missCount, wrongStateCount, listedOnlyCount int

	// Track discovered clubs by ID to avoid duplicates
	discoveredByClubID := map[int]bool{}
	// Cache slugs that 404'd
	deadSlugs := map[string]bool{}

	for i, input := range inputs {
		slugs := buildSlugs(input.Name, stateFull, input.City)
		log("[%d/%d] %q (city: %q) — %d slug candidates", i+1, len(inputs), input.Name, input.City, len(slugs))

		var club *ClubData
		var matchedSource string

		var wrongStateClub *ClubData
		var wrongStateSlug, wrongStateSource string

		for _, s := range slugs {
			if deadSlugs[s.slug] {
				continue
			}
			var err error
			club, err = probeSlug(client, s.slug)
			if err != nil {
				deadSlugs[s.slug] = true
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log("  %s (%s): HIT — %q (%s, %s)", s.slug, s.source, club.Name, club.City, club.Province)

			// State check inside loop — if wrong state, save it but keep trying
			clubState := strings.ToLower(club.Province)
			if clubState != stateFull && clubState != strings.ToLower(state) {
				log("  WRONG STATE — got %q, wanted %q — continuing", club.Province, stateFull)
				if wrongStateClub == nil {
					wrongStateClub = club
					wrongStateSlug = s.slug
					wrongStateSource = s.source
				}
				club = nil
				time.Sleep(100 * time.Millisecond)
				continue
			}

			matchedSource = s.source
			break
		}

		if club == nil && wrongStateClub != nil {
			log("  WRONG STATE (final) — no correct-state slug found")
			results = append(results, Result{
				Input: input.Name, City: input.City, Slug: wrongStateSlug, SlugSource: wrongStateSource,
				Status: "wrong_state", Club: wrongStateClub,
			})
			wrongStateCount++
			log("")
			continue
		}

		if club == nil {
			log("  MISS — no slug matched")
			results = append(results, Result{Input: input.Name, City: input.City, Status: "miss"})
			missCount++
			log("")
			continue
		}

		// Dedup by club ID
		if discoveredByClubID[club.ID] {
			log("  SKIPPED — club ID %d already discovered", club.ID)
			log("")
			continue
		}

		// City validation — warn but don't reject (Chronogolf city names can vary)
		if input.City != "" && !strings.EqualFold(strings.TrimSpace(club.City), strings.TrimSpace(input.City)) {
			log("  ⚠️  City mismatch: expected %q, got %q — continuing", input.City, club.City)
		}

		// Collect course UUIDs for tee time probe
		var courseUUIDs []string
		for _, cc := range club.Courses {
			if cc.UUID != "" {
				courseUUIDs = append(courseUUIDs, cc.UUID)
			}
		}

		// 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := 0

		if len(courseUUIDs) > 0 {
			for _, date := range dates {
				count, err := fetchTeeTimeCount(client, courseUUIDs, date)
				if err != nil {
					log("    %s: ERROR %v", date, err)
					count = 0
				}
				datesChecked = append(datesChecked, date)
				teeTimes = append(teeTimes, count)
				totalTimes += count
				log("    %s: %d tee times", date, count)
				time.Sleep(200 * time.Millisecond)
			}
		}

		courseNames := make([]string, len(club.Courses))
		for j, cc := range club.Courses {
			courseNames[j] = fmt.Sprintf("%s (%dh)", cc.Name, cc.Holes)
		}

		discoveredByClubID[club.ID] = true

		if totalTimes > 0 {
			log("  ✅ CONFIRMED — %d total tee times  courses=%v", totalTimes, courseNames)
			results = append(results, Result{
				Input: input.Name, City: input.City, Slug: club.Slug, SlugSource: matchedSource,
				Status: "confirmed", Club: club, DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
			confirmedCount++
		} else {
			log("  ⚠️  LISTED ONLY — 0 tee times  courses=%v", courseNames)
			results = append(results, Result{
				Input: input.Name, City: input.City, Slug: club.Slug, SlugSource: matchedSource,
				Status: "listed_only", Club: club, DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
			listedOnlyCount++
		}

		log("")
	}

	elapsed := time.Since(startTime)

	log("========================================")
	log("=== SUMMARY")
	log("========================================")
	log("Total probed:  %d", len(inputs))
	log("Confirmed:     %d", confirmedCount)
	log("Listed only:   %d", listedOnlyCount)
	log("Wrong state:   %d", wrongStateCount)
	log("Misses:        %d", missCount)
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
				log("  %-40s slug:%-45s [%s] id:%-6d %s (%s)  [%d times]",
					r.Input, r.Club.Slug, r.SlugSource, r.Club.ID, r.Club.Name, r.Club.City, total)
			}
		}
		log("")
	}

	if listedOnlyCount > 0 {
		log("=== LISTED ONLY ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  %-40s slug:%-45s [%s] id:%-6d %s (%s)",
					r.Input, r.Club.Slug, r.SlugSource, r.Club.ID, r.Club.Name, r.Club.City)
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
	filename := fmt.Sprintf("discovery/results/chronogolf-%s-%s.json",
		strings.ToLower(state), startTime.Format("2006-01-02-150405"))

	output := map[string]any{
		"platform":        "chronogolf",
		"state":           state,
		"timestamp":       startTime.Format(time.RFC3339),
		"elapsed":         elapsed.String(),
		"totalInput":      len(inputs),
		"confirmed":       confirmedCount,
		"listedOnly":      listedOnlyCount,
		"wrongState":      wrongStateCount,
		"misses":          missCount,
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
