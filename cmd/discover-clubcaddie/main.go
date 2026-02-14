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
// Usage:
//   Build master index (one-time scan):
//     go run cmd/discover-clubcaddie/main.go --index
//
//   Match against course list:
//     go run cmd/discover-clubcaddie/main.go --match AZ -f discovery/courses/phoenix.txt

type IndexEntry struct {
	Server    int    `json:"server"`
	APIKey    string `json:"apiKey"`
	CourseID  string `json:"courseId,omitempty"`
	Name      string `json:"name"`
	RawName   string `json:"rawName"`
	LogoURL   string `json:"logoUrl,omitempty"`
}

type MatchResult struct {
	Input        string      `json:"input"`
	City         string      `json:"city"`
	Status       string      `json:"status"`
	Match        *IndexEntry `json:"match,omitempty"`
	DatesChecked []string    `json:"datesChecked,omitempty"`
	TeeTimes     []int       `json:"teeTimes,omitempty"`
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

// probeKey checks if an API key is valid on a given server.
// Returns the course name extracted from the logo URL, or empty if invalid.
func probeKey(client *http.Client, server int, apiKey string) (string, string, error) {
	// Hit the slots page with a date — logo URL reveals the course name
	date := url.QueryEscape(time.Now().AddDate(0, 0, 3).Format("01/02/2006"))
	slotsURL := fmt.Sprintf("https://apimanager-cc%d.clubcaddie.com/webapi/view/%s/slots?date=%s&player=1&ratetype=any",
		server, apiKey, date)

	req, err := http.NewRequest("GET", slotsURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	html := string(body)
	if len(html) < 500 {
		return "", "", fmt.Errorf("empty page")
	}

	// Extract course name from logo URL: /Uploads/CourseLogo/0_{Name}_{timestamp}.png
	logoRe := regexp.MustCompile(`Uploads/CourseLogo/\d+_([^_]+(?:_[^_]+)*)_\d{8}_\d+\.(?:png|jpg|gif)`)
	logoMatch := logoRe.FindStringSubmatch(html)
	if len(logoMatch) > 1 {
		raw := logoMatch[1]
		// Convert CamelCase/underscored to readable name
		name := strings.ReplaceAll(raw, "_", " ")
		return name, raw, nil
	}

	// Fallback: look for <title> tag
	titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	titleMatch := titleRe.FindStringSubmatch(html)
	if len(titleMatch) > 1 && titleMatch[1] != "" {
		return strings.TrimSpace(titleMatch[1]), "", nil
	}

	// Fallback: check if page has tee time content (slot forms)
	if strings.Contains(html, "name=\"slot\"") {
		return "(unknown - has tee times)", "", nil
	}

	return "", "", fmt.Errorf("no course name found")
}

func fetchTeeTimeCount(client *http.Client, server int, apiKey, courseID, date string) (int, error) {
	// First GET slots page to establish session
	t, _ := time.Parse("2006-01-02", date)
	formDate := url.QueryEscape(t.Format("01/02/2006"))
	slotsURL := fmt.Sprintf("https://apimanager-cc%d.clubcaddie.com/webapi/view/%s/slots?date=%s&player=4&ratetype=any",
		server, apiKey, formDate)

	pageResp, err := client.Get(slotsURL)
	if err != nil {
		return 0, err
	}
	pageBody, _ := io.ReadAll(pageResp.Body)
	pageResp.Body.Close()

	// Extract interaction ID
	interactionRe := regexp.MustCompile(`Interaction=([a-zA-Z0-9]+)`)
	interactionMatch := interactionRe.FindStringSubmatch(string(pageBody))
	interaction := ""
	if len(interactionMatch) > 1 {
		interaction = interactionMatch[1]
	}

	// Extract CourseId from page if we don't have one
	if courseID == "" {
		courseIDRe := regexp.MustCompile(`CourseId["\s:=]+["]*(\d+)`)
		courseIDMatch := courseIDRe.FindStringSubmatch(string(pageBody))
		if len(courseIDMatch) > 1 {
			courseID = courseIDMatch[1]
		}
	}
	if courseID == "" {
		return 0, fmt.Errorf("no courseId found")
	}

	// POST to get tee times
	formData := url.Values{
		"date": {t.Format("01/02/2006")}, "player": {"4"}, "holes": {"any"},
		"fromtime": {"0"}, "totime": {"23"}, "minprice": {"0"}, "maxprice": {"9999"},
		"ratetype": {"any"}, "HoleGroup": {"all"}, "CourseId": {courseID},
		"apikey": {apiKey}, "Interaction": {interaction},
	}

	baseURL := fmt.Sprintf("https://apimanager-cc%d.clubcaddie.com", server)
	req, err := http.NewRequest("POST", baseURL+"/webapi/TeeTimes", strings.NewReader(formData.Encode()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for _, cookie := range pageResp.Cookies() {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Count slot inputs
	slotRe := regexp.MustCompile(`name="slot"`)
	return len(slotRe.FindAll(body, -1)), nil
}

func runIndex() {
	log("ClubCaddie Index Builder")
	log("Scanning API key prefixes aa-zz across servers")

	// Check for existing index to resume
	var index []IndexEntry
	indexPath := "discovery/clubcaddie-index.json"
	if data, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(data, &index)
		log("Loaded %d existing entries from index", len(index))
	}

	// Track what we've already scanned
	scanned := map[string]bool{}
	for _, e := range index {
		scanned[fmt.Sprintf("%d:%s", e.Server, e.APIKey)] = true
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	// Scan servers 1-50, keys aa-zz
	servers := []int{}
	for i := 1; i <= 50; i++ {
		servers = append(servers, i)
	}

	total := 0
	found := len(index)

	for _, srv := range servers {
		for c1 := 'a'; c1 <= 'z'; c1++ {
			for c2 := 'a'; c2 <= 'z'; c2++ {
				prefix := string(c1) + string(c2)
				apiKey := prefix + "fdabab"
				key := fmt.Sprintf("%d:%s", srv, apiKey)

				if scanned[key] {
					continue
				}

				total++
				name, rawName, err := probeKey(client, srv, apiKey)
				if err == nil && name != "" {
					found++
					entry := IndexEntry{
						Server:  srv,
						APIKey:  apiKey,
						Name:    name,
						RawName: rawName,
					}
					index = append(index, entry)
					log("  ✅ cc%d/%s  →  %s", srv, apiKey, name)

					// Save checkpoint every 10 finds
					if found%10 == 0 {
						saveIndex(indexPath, index)
					}
				}

				// Rate limit
				if total%100 == 0 {
					log("  progress: %d probed, %d found (server cc%d, prefix %s)", total, found, srv, prefix)
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	saveIndex(indexPath, index)
	log("")
	log("=== Index Complete ===")
	log("Total probed: %d", total)
	log("Total found: %d", found)
	log("Saved to %s", indexPath)
}

func saveIndex(path string, index []IndexEntry) {
	os.MkdirAll("discovery", 0755)
	data, _ := json.MarshalIndent(index, "", "  ")
	os.WriteFile(path, data, 0644)
}

type Course struct {
	Name string
	City string
}

func loadCourses(path string) ([]Course, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var courses []Course
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
		courses = append(courses, Course{Name: name, City: city})
	}
	return courses, nil
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, ".", "")
	// Strip common suffixes for matching
	for _, suf := range []string{"golf course", "golf club", "golf resort", "country club", "golf"} {
		s = strings.ReplaceAll(s, suf, "")
	}
	s = strings.TrimPrefix(s, "the ")
	// Collapse whitespace
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

func runMatch(state string, coursesFile string) {
	log("ClubCaddie Match — state=%s", state)

	// Load index
	indexPath := "discovery/clubcaddie-index.json"
	data, err := os.ReadFile(indexPath)
	if err != nil {
		log("ERROR: no index found at %s — run --index first", indexPath)
		os.Exit(1)
	}
	var index []IndexEntry
	json.Unmarshal(data, &index)
	log("Loaded %d index entries", len(index))

	// Load courses
	courses, err := loadCourses(coursesFile)
	if err != nil {
		log("ERROR: %v", err)
		os.Exit(1)
	}
	log("Loaded %d courses from %s", len(courses), coursesFile)

	// Build normalized index lookup
	type normEntry struct {
		norm  string
		entry IndexEntry
	}
	var normIndex []normEntry
	for _, e := range index {
		normIndex = append(normIndex, normEntry{norm: normalize(e.Name), entry: e})
	}

	client := &http.Client{Timeout: 15 * time.Second}
	dates := probeDates()
	var results []MatchResult
	confirmed, missed, listedOnly := 0, 0, 0
	startTime := time.Now()

	for i, c := range courses {
		normName := normalize(c.Name)
		var match *IndexEntry

		// Try exact normalized match
		for _, ne := range normIndex {
			if ne.norm == normName {
				match = &ne.entry
				break
			}
		}

		// Try contains match (index name contains course core, or vice versa)
		if match == nil {
			for _, ne := range normIndex {
				if strings.Contains(ne.norm, normName) || strings.Contains(normName, ne.norm) {
					match = &ne.entry
					break
				}
			}
		}

		if match == nil {
			missed++
			log("[%d/%d] %-45s  ❌ miss", i+1, len(courses), c.Name)
			results = append(results, MatchResult{Input: c.Name, City: c.City, Status: "miss"})
			continue
		}

		// 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := 0

		for _, date := range dates {
			count, err := fetchTeeTimeCount(client, match.Server, match.APIKey, match.CourseID, date)
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
			log("[%d/%d] %-45s  ✅ cc%d/%s  times=%v", i+1, len(courses), c.Name, match.Server, match.APIKey, teeTimes)
			results = append(results, MatchResult{
				Input: c.Name, City: c.City, Status: "confirmed",
				Match: match, DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		} else {
			listedOnly++
			log("[%d/%d] %-45s  📋 listed (0 times)  cc%d/%s", i+1, len(courses), c.Name, match.Server, match.APIKey)
			results = append(results, MatchResult{
				Input: c.Name, City: c.City, Status: "listed_only",
				Match: match, DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		}
	}

	elapsed := time.Since(startTime)
	log("")
	log("=== Results ===")
	log("Total: %d", len(courses))
	log("Confirmed: %d", confirmed)
	log("Listed Only: %d", listedOnly)
	log("Missed: %d", missed)
	log("Elapsed: %s", elapsed.Round(time.Second))

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
	outData, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, outData, 0644)
	log("Results saved to %s", outPath)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  discover-clubcaddie --index                        Build master index")
		fmt.Println("  discover-clubcaddie --match <STATE> -f <file>      Match courses against index")
		os.Exit(1)
	}

	if args[0] == "--index" {
		runIndex()
		return
	}

	if args[0] == "--match" && len(args) >= 4 && args[2] == "-f" {
		runMatch(args[1], args[3])
		return
	}

	fmt.Println("Invalid arguments. Use --index or --match <STATE> -f <file>")
	os.Exit(1)
}
