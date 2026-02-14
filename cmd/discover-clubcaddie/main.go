package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// ClubCaddie Discovery Tool
//
// Probes courses by guessing their website URL, fetching the HTML, and
// looking for ClubCaddie iframe embeds or links. Extracts server number
// and API key, then validates with a 3-date tee time check.
//
// Usage: go run cmd/discover-clubcaddie/main.go <state> -f <file>
//
// The file should have lines like: Course Name | City
//
// Example:
//   go run cmd/discover-clubcaddie/main.go CO -f discovery/courses/denver.txt

var ccURLRe = regexp.MustCompile(`apimanager-cc(\d+)\.clubcaddie\.com/webapi/view/([a-z0-9]+)`)

type ClubCaddieMatch struct {
	Server int
	APIKey string
	URL    string // the website URL where we found it
}

type Result struct {
	Input        string   `json:"input"`
	City         string   `json:"city"`
	Status       string   `json:"status"` // "confirmed", "listed_only", "miss"
	WebsiteURL   string   `json:"websiteUrl,omitempty"`
	Server       int      `json:"server,omitempty"`
	APIKey       string   `json:"apiKey,omitempty"`
	CourseID     int      `json:"courseId,omitempty"`
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

// joinAlpha strips non-alphanumeric chars and lowercases
func joinAlpha(s string) string {
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
		}
	}
	return b.String()
}

// coreName strips common golf suffixes and "the" prefix
func coreName(name string) string {
	s := strings.ToLower(name)
	for _, suffix := range []string{
		"golf course", "golf club", "golf resort", "golf complex",
		"golf links", "golf center", "country club",
	} {
		s = strings.ReplaceAll(s, suffix, "")
	}
	s = strings.TrimPrefix(s, "the ")
	return strings.TrimSpace(s)
}

// slugify creates a hyphenated slug
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// buildWebsiteURLs generates candidate website URLs from a course name.
// Golf course websites follow unpredictable patterns, so we try many variants.
func buildWebsiteURLs(name, city string) []string {
	seen := map[string]bool{}
	var urls []string
	add := func(domain string) {
		domain = strings.TrimSpace(domain)
		if domain == "" || seen[domain] {
			return
		}
		seen[domain] = true
		urls = append(urls, "https://www."+domain, "https://"+domain)
	}

	full := joinAlpha(name)
	core := joinAlpha(coreName(name))
	hyphenated := slugify(name)
	coreHyphenated := slugify(coreName(name))

	// Joined patterns (no separators): stonecreekgolfclub.com
	add(full + ".com")
	add(core + "golf.com")
	add(core + "golfclub.com")
	add(core + "golfcourse.com")
	add(core + "cc.com") // country club abbreviation
	add(core + ".com")

	// Without "the" prefix
	withoutThe := strings.TrimPrefix(full, "the")
	if withoutThe != full {
		add(withoutThe + ".com")
		add(withoutThe + "golf.com")
	}

	// Hyphenated patterns: stone-creek-golf-club.com
	add(hyphenated + ".com")
	add(coreHyphenated + "-golf.com")
	add(coreHyphenated + "-golf-club.com")
	add(coreHyphenated + "-golf-course.com")

	// "play" prefix: playstonecreek.com
	add("play" + core + ".com")
	add("play" + coreHyphenated + ".com")
	add("golf" + core + ".com")

	// .org and .net (common for municipal courses)
	add(full + ".org")
	add(core + "golf.org")
	add(hyphenated + ".org")
	add(full + ".net")

	return urls
}

// probeWebsite fetches a URL and looks for ClubCaddie embeds.
// If the homepage doesn't have an embed, checks common booking subpages.
// Only skips subpages if the base domain can't be reached at all.
func probeWebsite(client *http.Client, siteURL string) *ClubCaddieMatch {
	subpages := []string{"", "/book", "/tee-times", "/booking", "/book-a-tee-time", "/book-now", "/teetimes"}

	homepageReachable := false
	for _, subpage := range subpages {
		targetURL := siteURL + subpage
		match, reachable := probeOnePage(client, targetURL)
		if match != nil {
			return match
		}
		if subpage == "" {
			homepageReachable = reachable
			if !reachable {
				return nil // can't reach domain at all, skip subpages
			}
		}
	}
	_ = homepageReachable
	return nil
}

func probeOnePage(client *http.Client, targetURL string) (*ClubCaddieMatch, bool) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false // connection error
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != 200 {
		return nil, true // reachable but no content
	}

	m := ccURLRe.FindStringSubmatch(string(body))
	if m == nil {
		return nil, true // reachable but no embed
	}

	server := 0
	fmt.Sscanf(m[1], "%d", &server)
	return &ClubCaddieMatch{
		Server: server,
		APIKey: m[2],
		URL:    targetURL,
	}, true
}

// fetchTeeTimeCount probes for actual tee times on a given date.
// Returns (count, courseId, error).
func fetchTeeTimeCount(client *http.Client, server int, apiKey, date string) (int, int, error) {
	t, _ := time.Parse("2006-01-02", date)
	formDate := url.QueryEscape(t.Format("01/02/2006"))

	// Step 1: GET slots page to establish session and extract courseId + interaction
	slotsURL := fmt.Sprintf("https://apimanager-cc%d.clubcaddie.com/webapi/view/%s/slots?date=%s&player=4&ratetype=any",
		server, apiKey, formDate)

	pageResp, err := client.Get(slotsURL)
	if err != nil {
		return 0, 0, err
	}
	pageBody, _ := io.ReadAll(pageResp.Body)
	pageResp.Body.Close()

	html := string(pageBody)

	// Extract interaction ID
	interactionRe := regexp.MustCompile(`Interaction=([a-zA-Z0-9]+)`)
	interactionMatch := interactionRe.FindStringSubmatch(html)
	interaction := ""
	if len(interactionMatch) > 1 {
		interaction = interactionMatch[1]
	}

	// Extract CourseId
	courseIDRe := regexp.MustCompile(`CourseId["\s:=]+["]*(\d+)`)
	courseIDMatch := courseIDRe.FindStringSubmatch(html)
	courseID := ""
	if len(courseIDMatch) > 1 {
		courseID = courseIDMatch[1]
	}
	if courseID == "" {
		return 0, 0, fmt.Errorf("no courseId found")
	}

	var courseIDInt int
	fmt.Sscanf(courseID, "%d", &courseIDInt)

	// Step 2: POST to get tee times
	formData := url.Values{
		"date": {t.Format("01/02/2006")}, "player": {"4"}, "holes": {"any"},
		"fromtime": {"0"}, "totime": {"23"}, "minprice": {"0"}, "maxprice": {"9999"},
		"ratetype": {"any"}, "HoleGroup": {"all"}, "CourseId": {courseID},
		"apikey": {apiKey}, "Interaction": {interaction},
	}

	baseURL := fmt.Sprintf("https://apimanager-cc%d.clubcaddie.com", server)
	req, err := http.NewRequest("POST", baseURL+"/webapi/TeeTimes", strings.NewReader(formData.Encode()))
	if err != nil {
		return 0, courseIDInt, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for _, cookie := range pageResp.Cookies() {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, courseIDInt, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	// Count slot inputs
	slotRe := regexp.MustCompile(`name="slot"`)
	return len(slotRe.FindAll(respBody, -1)), courseIDInt, nil
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
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-clubcaddie/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run cmd/discover-clubcaddie/main.go CO -f discovery/courses/denver.txt\n")
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

	log("=== ClubCaddie Discovery ===")
	log("State: %s", state)
	log("Courses to probe: %d", len(courses))
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var results []Result
	confirmed, missed, listedOnly := 0, 0, 0

	for i, c := range courses {
		candidateURLs := buildWebsiteURLs(c.Name, c.City)

		// Deduplicate domains for logging
		slugsForLog := []string{}
		seen := map[string]bool{}
		for _, u := range candidateURLs {
			domain := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "www.")
			if !seen[domain] {
				seen[domain] = true
				slugsForLog = append(slugsForLog, domain)
			}
		}

		var match *ClubCaddieMatch
		for _, siteURL := range candidateURLs {
			match = probeWebsite(client, siteURL)
			if match != nil {
				break
			}
		}

		if match == nil {
			missed++
			log("[%d/%d] %-45s  ❌ miss  (tried %d URLs)", i+1, len(courses), c.Name, len(candidateURLs))
			results = append(results, Result{
				Input:      c.Name,
				City:       c.City,
				Status:     "miss",
				SlugsTried: slugsForLog,
			})
			continue
		}

		log("[%d/%d] %-45s  🔍 found cc%d/%s via %s", i+1, len(courses), c.Name, match.Server, match.APIKey, match.URL)

		// 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := 0
		var courseID int

		for _, date := range dates {
			count, cid, err := fetchTeeTimeCount(client, match.Server, match.APIKey, date)
			if err != nil {
				count = 0
			}
			if cid != 0 {
				courseID = cid
			}
			datesChecked = append(datesChecked, date)
			teeTimes = append(teeTimes, count)
			totalTimes += count
			time.Sleep(300 * time.Millisecond)
		}

		if totalTimes > 0 {
			confirmed++
			log("[%d/%d] %-45s  ✅ confirmed  cc%d/%s  courseId=%d  times=%v",
				i+1, len(courses), c.Name, match.Server, match.APIKey, courseID, teeTimes)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "confirmed",
				WebsiteURL: match.URL, Server: match.Server, APIKey: match.APIKey,
				CourseID: courseID, DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		} else {
			listedOnly++
			log("[%d/%d] %-45s  📋 listed (0 times)  cc%d/%s  courseId=%d",
				i+1, len(courses), c.Name, match.Server, match.APIKey, courseID)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "listed_only",
				WebsiteURL: match.URL, Server: match.Server, APIKey: match.APIKey,
				CourseID: courseID, DatesChecked: datesChecked, TeeTimes: teeTimes,
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
	log("Missed: %d", missed)
	log("Elapsed: %s", elapsed.Round(time.Second))
	log("")

	if confirmed > 0 {
		log("=== Confirmed ===")
		for _, r := range results {
			if r.Status == "confirmed" {
				log("  ✅ %-45s  cc%d/%s  courseId=%d  times=%v",
					r.Input, r.Server, r.APIKey, r.CourseID, r.TeeTimes)
			}
		}
		log("")
	}

	if listedOnly > 0 {
		log("=== Listed Only ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  📋 %-45s  cc%d/%s  courseId=%d",
					r.Input, r.Server, r.APIKey, r.CourseID)
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
	outPath := fmt.Sprintf("discovery/results/clubcaddie-%s-%s.json", strings.ToLower(state), ts)

	output := map[string]any{
		"platform":   "clubcaddie",
		"state":      state,
		"timestamp":  time.Now().Format(time.RFC3339),
		"elapsed":    elapsed.String(),
		"total":      len(courses),
		"confirmed":  confirmed,
		"listedOnly": listedOnly,
		"missed":     missed,
		"results":    results,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, data, 0644)
	log("Results saved to %s", outPath)
}
