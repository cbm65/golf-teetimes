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

// GolfWithAccess (Troon) Discovery Tool
//
// Probes courses by guessing slugs on golfwithaccess.com. The page HTML
// contains React Server Components data with courseId UUID, city, and state.
// Then validates with the tee-times API.
//
// Usage: go run cmd/discover-golfwithaccess/main.go <state> -f <file>

var (
	// Extract courseId UUID from RSC data: courses:...[id:"UUID"
	courseIdRe = regexp.MustCompile(`courses:\$R\[\d+\]=\[\$R\[\d+\]=\{id:"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
	// Extract city and state from address block
	cityRe  = regexp.MustCompile(`city:"([^"]+)"`)
	stateRe = regexp.MustCompile(`state:"([^"]+)"`)
	// Extract course name
	courseNameRe = regexp.MustCompile(`courses:\$R\[\d+\]=\[\$R\[\d+\]=\{id:"[^"]+",name:"([^"]+)"`)
)

// stateAbbreviations maps full names to abbreviations
var stateAbbreviations = map[string]string{
	"Alabama": "AL", "Alaska": "AK", "Arizona": "AZ", "Arkansas": "AR",
	"California": "CA", "Colorado": "CO", "Connecticut": "CT", "Delaware": "DE",
	"Florida": "FL", "Georgia": "GA", "Hawaii": "HI", "Idaho": "ID",
	"Illinois": "IL", "Indiana": "IN", "Iowa": "IA", "Kansas": "KS",
	"Kentucky": "KY", "Louisiana": "LA", "Maine": "ME", "Maryland": "MD",
	"Massachusetts": "MA", "Michigan": "MI", "Minnesota": "MN", "Mississippi": "MS",
	"Missouri": "MO", "Montana": "MT", "Nebraska": "NE", "Nevada": "NV",
	"New Hampshire": "NH", "New Jersey": "NJ", "New Mexico": "NM", "New York": "NY",
	"North Carolina": "NC", "North Dakota": "ND", "Ohio": "OH", "Oklahoma": "OK",
	"Oregon": "OR", "Pennsylvania": "PA", "Rhode Island": "RI", "South Carolina": "SC",
	"South Dakota": "SD", "Tennessee": "TN", "Texas": "TX", "Utah": "UT",
	"Vermont": "VT", "Virginia": "VA", "Washington": "WA", "West Virginia": "WV",
	"Wisconsin": "WI", "Wyoming": "WY",
}

type TeeTimeResponse struct {
	TeeTimes []struct {
		DayTime struct {
			Hour   int `json:"hour"`
			Minute int `json:"minute"`
		} `json:"dayTime"`
	} `json:"teeTimes"`
}

type Result struct {
	Input        string   `json:"input"`
	City         string   `json:"city"`
	Status       string   `json:"status"` // "confirmed", "listed_only", "wrong_state", "miss"
	Slug         string   `json:"slug,omitempty"`
	CourseID     string   `json:"courseId,omitempty"`
	FoundCity    string   `json:"foundCity,omitempty"`
	FoundState   string   `json:"foundState,omitempty"`
	CourseName   string   `json:"courseName,omitempty"`
	DatesChecked []string `json:"datesChecked,omitempty"`
	TeeTimes     []int    `json:"teeTimes,omitempty"`
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

// coreName strips common golf suffixes and prefixes
func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Golf Links", " Golf Center", " Country Club",
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

// slugify converts a name to a URL slug: "Eagle Mountain Golf Club" → "eagle-mountain-golf-club"
func slugify(name string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s := re.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// buildSlugs generates candidate slugs for GolfWithAccess.
// GWA uses clean hyphenated slugs: "eagle-mountain-golf-club"
func buildSlugs(name string) []string {
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

	full := slugify(name)
	core := slugify(coreName(name))

	add(full)                         // eagle-mountain-golf-club
	add(core + "-golf-club")          // eagle-mountain-golf-club (may dup)
	add(core + "-golf-course")        // eagle-mountain-golf-course
	add(core + "-golf-resort")        // eagle-mountain-golf-resort
	add(core + "-country-club")       // eagle-mountain-country-club
	add(core + "-golf")               // eagle-mountain-golf
	add(core)                         // eagle-mountain
	add("the-" + core + "-golf-club") // the-eagle-mountain-golf-club

	// Handle "Golf Club of X" → "x-golf-club"
	if strings.HasPrefix(name, "Golf Club of ") || strings.HasPrefix(name, "Golf Club at ") {
		rest := name[13:]
		add(slugify(rest) + "-golf-club")
		add(slugify(rest))
	}

	return slugs
}

type PageData struct {
	CourseID  string
	Name      string
	City      string
	StateAbbr string
}

// probePage fetches a GolfWithAccess course page and extracts courseId, city, state.
func probePage(client *http.Client, slug string) (*PageData, error) {
	url := "https://golfwithaccess.com/course/" + slug + "/reserve-tee-time"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

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

	html := string(body)

	// Extract courseId UUID
	courseMatch := courseIdRe.FindStringSubmatch(html)
	if courseMatch == nil {
		return nil, fmt.Errorf("no courseId found")
	}

	// Extract course name
	nameStr := ""
	nameMatch := courseNameRe.FindStringSubmatch(html)
	if nameMatch != nil {
		nameStr = nameMatch[1]
	}

	// Extract city
	cityStr := ""
	cityMatch := cityRe.FindStringSubmatch(html)
	if cityMatch != nil {
		cityStr = cityMatch[1]
	}

	// Extract state (full name → abbreviation)
	stateAbbr := ""
	stateMatch := stateRe.FindStringSubmatch(html)
	if stateMatch != nil {
		if abbr, ok := stateAbbreviations[stateMatch[1]]; ok {
			stateAbbr = abbr
		} else {
			stateAbbr = stateMatch[1]
		}
	}

	return &PageData{
		CourseID:  courseMatch[1],
		Name:      nameStr,
		City:      cityStr,
		StateAbbr: stateAbbr,
	}, nil
}

// probeTeeTimes hits the tee-times API and returns the count.
func probeTeeTimes(client *http.Client, courseID, date string) (int, error) {
	url := fmt.Sprintf(
		"https://golfwithaccess.com/api/v1/tee-times?courseIds=%s&players=1&startAt=00:00:00&endAt=23:59:59&day=%s",
		courseID, date,
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

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

	var data TeeTimeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	return len(data.TeeTimes), nil
}

// findSlug tries slug candidates and returns the first valid page data.
func findSlug(client *http.Client, slugs []string, deadCache map[string]bool) (string, *PageData) {
	for _, slug := range slugs {
		if deadCache[slug] {
			continue
		}
		data, err := probePage(client, slug)
		if err != nil {
			deadCache[slug] = true
			continue
		}
		return slug, data
	}
	return "", nil
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
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-golfwithaccess/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run cmd/discover-golfwithaccess/main.go AZ -f discovery/courses/phoenix.txt\n")
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

	log("=== GolfWithAccess (Troon) Discovery ===")
	log("State: %s", state)
	log("Courses to probe: %d", len(courses))
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	client := &http.Client{Timeout: 15 * time.Second}

	var results []Result
	confirmed, missed, listedOnly, wrongState := 0, 0, 0, 0
	deadSlugs := map[string]bool{}
	seenCourseIDs := map[string]bool{}

	for i, c := range courses {
		slugs := buildSlugs(c.Name)

		foundSlug, pageData := findSlug(client, slugs, deadSlugs)

		if foundSlug == "" {
			missed++
			log("[%d/%d] %-45s  ❌ miss  (tried: %s)", i+1, len(courses), c.Name, strings.Join(slugs, ", "))
			results = append(results, Result{
				Input:      c.Name,
				City:       c.City,
				Status:     "miss",
				SlugsTried: slugs,
			})
			continue
		}

		log("[%d/%d] %-45s  🔍 found: %s  courseId=%s  %s, %s",
			i+1, len(courses), c.Name, foundSlug, pageData.CourseID[:8]+"...", pageData.City, pageData.StateAbbr)

		// State validation
		if pageData.StateAbbr != "" && pageData.StateAbbr != state {
			wrongState++
			log("[%d/%d] %-45s  ⚠️  WRONG STATE — found %s, wanted %s",
				i+1, len(courses), c.Name, pageData.StateAbbr, state)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "wrong_state",
				Slug: foundSlug, CourseID: pageData.CourseID,
				FoundCity: pageData.City, FoundState: pageData.StateAbbr,
			})
			continue
		}

		// Dedup by courseId
		if seenCourseIDs[pageData.CourseID] {
			log("[%d/%d] %-45s  ⚠️  DUPLICATE courseId (already found)", i+1, len(courses), c.Name)
			missed++
			results = append(results, Result{Input: c.Name, City: c.City, Status: "miss"})
			continue
		}
		seenCourseIDs[pageData.CourseID] = true

		// City mismatch warning (not rejection)
		if c.City != "" && pageData.City != "" && !strings.EqualFold(c.City, pageData.City) {
			log("[%d/%d] %-45s  ℹ️  city mismatch: input=%s, found=%s",
				i+1, len(courses), c.Name, c.City, pageData.City)
		}

		// Phase 2: 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := 0

		for _, date := range dates {
			count, err := probeTeeTimes(client, pageData.CourseID, date)
			if err != nil {
				count = 0
			}
			datesChecked = append(datesChecked, date)
			teeTimes = append(teeTimes, count)
			totalTimes += count
			time.Sleep(300 * time.Millisecond)
		}

		if totalTimes > 0 {
			confirmed++
			log("[%d/%d] %-45s  ✅ confirmed  %s  courseId=%s  times=%v",
				i+1, len(courses), c.Name, foundSlug, pageData.CourseID, teeTimes)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "confirmed",
				Slug: foundSlug, CourseID: pageData.CourseID,
				CourseName: pageData.Name,
				FoundCity: pageData.City, FoundState: pageData.StateAbbr,
				DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		} else {
			listedOnly++
			log("[%d/%d] %-45s  📋 listed (0 times)  %s  courseId=%s",
				i+1, len(courses), c.Name, foundSlug, pageData.CourseID)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "listed_only",
				Slug: foundSlug, CourseID: pageData.CourseID,
				CourseName: pageData.Name,
				FoundCity: pageData.City, FoundState: pageData.StateAbbr,
				DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		}

		time.Sleep(500 * time.Millisecond)
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
				log("  ✅ %-45s  slug=%s  courseId=%s  times=%v",
					r.Input, r.Slug, r.CourseID, r.TeeTimes)
			}
		}
		log("")
	}

	if listedOnly > 0 {
		log("=== Listed Only ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  📋 %-45s  slug=%s  courseId=%s", r.Input, r.Slug, r.CourseID)
			}
		}
		log("")
	}

	if wrongState > 0 {
		log("=== Wrong State ===")
		for _, r := range results {
			if r.Status == "wrong_state" {
				log("  ⚠️  %-45s  slug=%s → %s (wanted %s)", r.Input, r.Slug, r.FoundState, state)
			}
		}
		log("")
	}

	// Save results
	os.MkdirAll("discovery/results", 0755)
	ts := time.Now().Format("2006-01-02-150405")
	outPath := fmt.Sprintf("discovery/results/golfwithaccess-%s-%s.json", strings.ToLower(state), ts)

	output := map[string]any{
		"platform":   "golfwithaccess",
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
