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

// CourseCo Discovery Tool
//
// Probes courses by guessing CourseCo subdomains. CourseCo uses:
//   subdomain: {slug}.totaleintegrated.net
//   API: courseco-gateway.totaleintegrated.net/Booking/Teetimes?CourseID={SLUG_UPPER}
//   Origin header must match the subdomain.
//
// Usage: go run cmd/discover-courseco/main.go <state> -f <file>

type TeeTimeData struct {
	Title         string  `json:"Title"`
	PerPlayerCost float64 `json:"PerPlayerCost"`
	AvailableSlot string  `json:"AvailableSlot"`
	Time          string  `json:"Time"`
}

type CourseCoResponse struct {
	TeeTimeData []TeeTimeData `json:"TeeTimeData"`
}

type Result struct {
	Input        string   `json:"input"`
	City         string   `json:"city"`
	Status       string   `json:"status"` // "confirmed", "listed_only", "miss"
	Subdomain    string   `json:"subdomain,omitempty"`
	CourseID     string   `json:"courseId,omitempty"`
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
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// coreName strips common golf suffixes and prefixes
func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Golf Links", " Golf Center", " Country Club", " GC", " CC",
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

// buildSubdomains generates candidate CourseCo subdomains.
// From "Ken McDonald Golf Course" → kenmcdonald, kenmcdonaldgolfcourse, kenmcdonaldgolf, etc.
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

	add(core)                // kenmcdonald
	add(full)                // kenmcdonaldgolfcourse
	add(core + "golf")       // kenmcdonaldgolf
	add(core + "golfclub")   // kenmcdonaldgolfclub
	add(core + "golfcourse") // kenmcdonaldgolfcourse (may dup)
	add(core + "cc")         // kenmcdonaldcc

	// Hyphenated: ken-mcdonald
	re := regexp.MustCompile(`[^a-z0-9]+`)
	hyphenated := re.ReplaceAllString(strings.ToLower(name), "-")
	hyphenated = strings.Trim(hyphenated, "-")
	add(strings.ReplaceAll(hyphenated, "-", "")) // may dup full

	coreHyphen := re.ReplaceAllString(coreName(name), "-")
	coreHyphen = strings.Trim(coreHyphen, "-")
	if strings.Contains(coreHyphen, "-") {
		add(strings.ReplaceAll(coreHyphen, "-", "")) // may dup core
	}

	// Without "the"
	withoutThe := strings.TrimPrefix(full, "the")
	if withoutThe != full {
		add(withoutThe)
		add(withoutThe + "golf")
	}

	return slugs
}

// probeSubdomain checks if a CourseCo subdomain has tee times on a given date.
func probeSubdomain(client *http.Client, subdomain, date string) (int, error) {
	courseID := strings.ToUpper(subdomain)
	url := fmt.Sprintf(
		"https://courseco-gateway.totaleintegrated.net/Booking/Teetimes?IsInitTeeTimeRequest=false&TeeTimeDate=%s&CourseID=%s&StartTime=05:00&EndTime=21:00&NumOfPlayers=-1&Holes=18&IsNineHole=0&StartPrice=0&EndPrice=&CartIncluded=false&SpecialsOnly=0&IsClosest=0&PlayerIDs=&DateFilterChange=false&DateFilterChangeNoSearch=false&SearchByGroups=true&IsPrepaidOnly=0&QueryStringFilters=null",
		date, courseID,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Origin", "https://"+subdomain+".totaleintegrated.net")
	req.Header.Set("Referer", "https://"+subdomain+".totaleintegrated.net/")

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

	var data CourseCoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("invalid JSON")
	}

	return len(data.TeeTimeData), nil
}

// subdomainExists checks if {subdomain}.totaleintegrated.net is a real CourseCo site.
func subdomainExists(client *http.Client, subdomain string) bool {
	url := fmt.Sprintf("https://%s.totaleintegrated.net", subdomain)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	return resp.StatusCode == 200
}

// findSubdomain tries subdomains against CourseCo.
// First checks if the subdomain site exists, then probes the gateway for tee times.
func findSubdomain(client *http.Client, subdomains []string, date string, deadCache map[string]bool) (string, int) {
	for _, sub := range subdomains {
		if deadCache[sub] {
			continue
		}
		if !subdomainExists(client, sub) {
			deadCache[sub] = true
			continue
		}
		count, err := probeSubdomain(client, sub, date)
		if err != nil {
			deadCache[sub] = true
			continue
		}
		return sub, count
	}
	return "", 0
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
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-courseco/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run cmd/discover-courseco/main.go AZ -f discovery/courses/phoenix.txt\n")
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

	log("=== CourseCo Discovery ===")
	log("State: %s", state)
	log("Courses to probe: %d", len(courses))
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	client := &http.Client{Timeout: 10 * time.Second}

	var results []Result
	confirmed, missed, listedOnly := 0, 0, 0
	deadSubdomains := map[string]bool{}

	for i, c := range courses {
		subdomains := buildSubdomains(c.Name)

		// Phase 1: Find a valid subdomain using the first probe date
		foundSub, initialCount := findSubdomain(client, subdomains, dates[0], deadSubdomains)

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

		log("[%d/%d] %-45s  🔍 found: %s.totaleintegrated.net", i+1, len(courses), c.Name, foundSub)

		// Phase 2: 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := initialCount

		datesChecked = append(datesChecked, dates[0])
		teeTimes = append(teeTimes, initialCount)

		for _, date := range dates[1:] {
			count, err := probeSubdomain(client, foundSub, date)
			if err != nil {
				count = 0
			}
			datesChecked = append(datesChecked, date)
			teeTimes = append(teeTimes, count)
			totalTimes += count
			time.Sleep(300 * time.Millisecond)
		}

		courseID := strings.ToUpper(foundSub)
		if totalTimes > 0 {
			confirmed++
			log("[%d/%d] %-45s  ✅ confirmed  %s  courseId=%s  times=%v",
				i+1, len(courses), c.Name, foundSub, courseID, teeTimes)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "confirmed",
				Subdomain: foundSub, CourseID: courseID,
				DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		} else {
			listedOnly++
			log("[%d/%d] %-45s  📋 listed (0 times)  %s  courseId=%s",
				i+1, len(courses), c.Name, foundSub, courseID)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "listed_only",
				Subdomain: foundSub, CourseID: courseID,
				DatesChecked: datesChecked, TeeTimes: teeTimes,
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
				log("  ✅ %-45s  %s.totaleintegrated.net  courseId=%s  times=%v",
					r.Input, r.Subdomain, r.CourseID, r.TeeTimes)
			}
		}
		log("")
	}

	if listedOnly > 0 {
		log("=== Listed Only ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  📋 %-45s  %s.totaleintegrated.net  courseId=%s", r.Input, r.Subdomain, r.CourseID)
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
	outPath := fmt.Sprintf("discovery/results/courseco-%s-%s.json", strings.ToLower(state), ts)

	output := map[string]any{
		"platform":   "courseco",
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
