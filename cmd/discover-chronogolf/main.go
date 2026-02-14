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
// Probes courses by constructing URL slugs from name + state + city and
// fetching the club page. The embedded __NEXT_DATA__ contains club_id,
// course UUIDs, affiliation_type_id — everything needed for tee time queries.
//
// Usage: go run cmd/discover-chronogolf/main.go <state> -f <file>
//
// The file should have lines like: Course Name | City
//
// Example:
//   go run cmd/discover-chronogolf/main.go AZ -f discovery/courses/phoenix.txt

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
	Status       string    `json:"status"` // "confirmed", "listed_only", "wrong_state", "miss"
	Club         *ClubData `json:"club,omitempty"`
	DatesChecked []string  `json:"datesChecked,omitempty"`
	TeeTimes     []int     `json:"teeTimes,omitempty"`
}

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

func probeDates() []string {
	now := time.Now()
	var dates []string
	// Next Wednesday
	d := now
	for d.Weekday() != time.Wednesday {
		d = d.AddDate(0, 0, 1)
	}
	dates = append(dates, d.Format("2006-01-02"))
	// Next Saturday
	d = now
	for d.Weekday() != time.Saturday {
		d = d.AddDate(0, 0, 1)
	}
	dates = append(dates, d.Format("2006-01-02"))
	// Saturday after that
	dates = append(dates, d.AddDate(0, 0, 7).Format("2006-01-02"))
	return dates
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

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	// Replace non-alphanumeric with hyphens
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteRune('-')
		}
	}
	// Collapse multiple hyphens
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}

func coreName(name string) string {
	s := strings.ToLower(name)
	// Strip common suffixes
	for _, suffix := range []string{
		"golf course", "golf club", "golf resort", "golf complex",
		"country club", "golf links", "golf center",
	} {
		s = strings.ReplaceAll(s, suffix, "")
	}
	s = strings.TrimPrefix(s, "the ")
	return strings.TrimSpace(s)
}

func buildSlugs(name, stateFull, city string) []string {
	seen := map[string]bool{}
	var slugs []string
	add := func(s string) {
		s = slugify(s)
		if s != "" && !seen[s] {
			seen[s] = true
			slugs = append(slugs, s)
		}
	}

	core := coreName(name)

	// Full name + state + city
	add(name + " " + stateFull + " " + city)
	// Full name only
	add(name)
	// Core name + common suffixes
	add(core + " golf club")
	add(core + " golf course")
	add(core + " resort")
	add(core + " golf")
	// Core name bare
	add(core)

	return slugs
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
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-chronogolf/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run cmd/discover-chronogolf/main.go AZ -f discovery/courses/phoenix.txt\n")
		os.Exit(1)
	}

	state := strings.ToUpper(os.Args[1])
	stateFull, ok := stateNames[state]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown state: %s\n", state)
		os.Exit(1)
	}

	courses, err := readCoursesFromFile(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	startTime := time.Now()

	log("=== Chronogolf Discovery ===")
	log("State: %s (%s)", state, stateFull)
	log("Courses to probe: %d", len(courses))
	log("")

	client := &http.Client{Timeout: 15 * time.Second}

	var results []Result
	confirmed, missed, wrongState, listedOnly := 0, 0, 0, 0
	dates := probeDates()

	for i, c := range courses {
		if c.City == "" {
			log("[%d/%d] %-45s  ⚠️  no city, skipping", i+1, len(courses), c.Name)
			results = append(results, Result{Input: c.Name, City: c.City, Status: "miss"})
			missed++
			continue
		}

		slugs := buildSlugs(c.Name, stateFull, c.City)
		var club *ClubData
		var matchedSlug string

		for _, slug := range slugs {
			club, err = probeSlug(client, slug)
			if err == nil {
				matchedSlug = slug
				break
			}
		}

		if club != nil {
			// Validate state — slug may match a course in another state
			clubState := strings.ToLower(club.Province)
			if clubState != stateFull && clubState != strings.ToLower(state) {
				log("[%d/%d] %-45s  ⚠️  wrong state: %s (slug=%s)", i+1, len(courses), c.Name, club.Province, club.Slug)
				results = append(results, Result{
					Input:  c.Name,
					City:   c.City,
					Slug:   matchedSlug,
					Status: "wrong_state",
					Club:   club,
				})
				wrongState++
				continue
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
						count = 0
					}
					datesChecked = append(datesChecked, date)
					teeTimes = append(teeTimes, count)
					totalTimes += count
					time.Sleep(200 * time.Millisecond)
				}
			}

			courseNames := make([]string, len(club.Courses))
			for j, cc := range club.Courses {
				courseNames[j] = cc.Name
			}

			if totalTimes > 0 {
				confirmed++
				log("[%d/%d] %-45s  ✅ id=%d  times=%v  courses=%v",
					i+1, len(courses), c.Name, club.ID, teeTimes, courseNames)
				results = append(results, Result{
					Input:        c.Name,
					City:         c.City,
					Slug:         club.Slug,
					Status:       "confirmed",
					Club:         club,
					DatesChecked: datesChecked,
					TeeTimes:     teeTimes,
				})
			} else {
				listedOnly++
				log("[%d/%d] %-45s  📋 listed (0 times)  id=%d  slug=%s",
					i+1, len(courses), c.Name, club.ID, club.Slug)
				results = append(results, Result{
					Input:        c.Name,
					City:         c.City,
					Slug:         club.Slug,
					Status:       "listed_only",
					Club:         club,
					DatesChecked: datesChecked,
					TeeTimes:     teeTimes,
				})
			}
		} else {
			missed++
			log("[%d/%d] %-45s  ❌ miss  (tried %d slugs)", i+1, len(courses), c.Name, len(slugs))
			results = append(results, Result{
				Input:  c.Name,
				City:   c.City,
				Slug:   matchedSlug,
				Status: "miss",
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
				log("  %-45s  id=%-6d  slug=%s", r.Club.Name, r.Club.ID, r.Club.Slug)
			}
		}
		log("")
	}

	if missed > 0 {
		log("=== Missed ===")
		for _, r := range results {
			if r.Status == "miss" {
				log("  %-45s  (tried: %s)", r.Input, r.Slug)
			}
		}
		log("")
	}

	// Save results
	os.MkdirAll("discovery/results", 0755)
	ts := time.Now().Format("2006-01-02-150405")
	outPath := fmt.Sprintf("discovery/results/chronogolf-%s-%s.json", strings.ToLower(state), ts)

	output := map[string]any{
		"platform":  "chronogolf",
		"state":     state,
		"timestamp": time.Now().Format(time.RFC3339),
		"elapsed":   elapsed.String(),
		"total":     len(courses),
		"confirmed":  confirmed,
		"listedOnly": listedOnly,
		"wrongState": wrongState,
		"missed":     missed,
		"results":   results,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, data, 0644)
	log("Results saved to %s", outPath)
}
