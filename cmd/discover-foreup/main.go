package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ForeUP Discovery Tool
//
// Two modes:
//
//   --index <start> <end>    Scan course_ids and build/append to the master index.
//                            Results saved to discovery/foreup-index.json.
//                            Automatically resumes from where you left off.
//
//   --match <state> -f <file>  Match a course list against the saved index.
//
// Examples:
//   go run cmd/discover-foreup/main.go --index 1 30000
//   go run cmd/discover-foreup/main.go --match AZ -f discovery/courses/phoenix.txt

const indexPath = "discovery/foreup-index.json"

var bookingPageBase = "https://foreupsoftware.com/index.php/booking/index"

var courseRe = regexp.MustCompile(`COURSE\s*=\s*(\{.*?\})\s*;`)
var defaultFilterRe = regexp.MustCompile(`DEFAULT_FILTER\s*=\s*(\{.*?\})\s*;`)
var schedulesRe = regexp.MustCompile(`SCHEDULES\s*=\s*(\[.*?\])\s*;`)

type IndexEntry struct {
	CourseID     int    `json:"courseId"`
	ScheduleID   int    `json:"scheduleId"`
	BookingClass string `json:"bookingClass"`
	Name         string `json:"name"`
	City         string `json:"city"`
	State        string `json:"state"`
	Address      string `json:"address,omitempty"`
	Postal       string `json:"postal,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Website      string `json:"website,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
	Holes        int    `json:"holes,omitempty"`
}

type IndexFile struct {
	Platform   string       `json:"platform"`
	Updated    string       `json:"updated"`
	ScannedTo  int          `json:"scannedTo"`
	TotalFound int          `json:"totalFound"`
	Courses    []IndexEntry `json:"courses"`
}

type RawCourse struct {
	CourseID string `json:"course_id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"`
	Postal   string `json:"postal"`
	Country  string `json:"country"`
	Website  string `json:"website"`
	Phone    string `json:"phone"`
	Timezone string `json:"timezone"`
}

type RawFilter struct {
	ScheduleID   int `json:"schedule_id"`
	BookingClass any `json:"booking_class"`
}

type RawSchedule struct {
	TeesheetID string `json:"teesheet_id"`
	Title      string `json:"title"`
	Holes      string `json:"holes"`
}

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

func bookingClassStr(bc any) string {
	switch v := bc.(type) {
	case float64:
		return strconv.Itoa(int(v))
	case string:
		return v
	default:
		return ""
	}
}

func probeCourseID(client *http.Client, courseID int) (*IndexEntry, error) {
	url := fmt.Sprintf("%s/%d", bookingPageBase, courseID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

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

	m := courseRe.FindStringSubmatch(html)
	if m == nil {
		return nil, fmt.Errorf("no COURSE found")
	}
	var course RawCourse
	if err := json.Unmarshal([]byte(m[1]), &course); err != nil {
		return nil, fmt.Errorf("parse COURSE: %w", err)
	}

	var df RawFilter
	m = defaultFilterRe.FindStringSubmatch(html)
	if m != nil {
		json.Unmarshal([]byte(m[1]), &df)
	}

	holes := 0
	m = schedulesRe.FindStringSubmatch(html)
	if m != nil {
		var schedules []RawSchedule
		if json.Unmarshal([]byte(m[1]), &schedules) == nil && len(schedules) > 0 {
			holes, _ = strconv.Atoi(schedules[0].Holes)
		}
	}

	return &IndexEntry{
		CourseID:     courseID,
		ScheduleID:   df.ScheduleID,
		BookingClass: bookingClassStr(df.BookingClass),
		Name:         course.Name,
		City:         course.City,
		State:        course.State,
		Address:      course.Address,
		Postal:       course.Postal,
		Phone:        course.Phone,
		Website:      course.Website,
		Timezone:     course.Timezone,
		Holes:        holes,
	}, nil
}

func loadIndex() *IndexFile {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return &IndexFile{Platform: "foreup"}
	}
	var idx IndexFile
	if json.Unmarshal(data, &idx) != nil {
		return &IndexFile{Platform: "foreup"}
	}
	return &idx
}

func saveIndex(idx *IndexFile) {
	idx.Updated = time.Now().Format(time.RFC3339)
	idx.TotalFound = len(idx.Courses)
	os.MkdirAll("discovery", 0755)
	data, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(indexPath, data, 0644)
}

// --- Index mode ---

func runIndex(startID, endID int) {
	idx := loadIndex()

	// Resume: skip IDs already scanned
	if idx.ScannedTo >= startID {
		log("Index exists with %d courses, scanned to ID %d", len(idx.Courses), idx.ScannedTo)
		if idx.ScannedTo >= endID {
			log("Already scanned this range. Nothing to do.")
			return
		}
		startID = idx.ScannedTo + 1
		log("Resuming from course_id %d", startID)
	}

	totalIDs := endID - startID + 1
	startTime := time.Now()

	log("=== ForeUP Index Build ===")
	log("Scanning course_ids %d - %d (%d IDs)", startID, endID, totalIDs)
	log("Output: %s", indexPath)
	log("")

	client := &http.Client{Timeout: 15 * time.Second}
	newCount := 0

	for id := startID; id <= endID; id++ {
		entry, err := probeCourseID(client, id)
		if err == nil {
			idx.Courses = append(idx.Courses, *entry)
			newCount++
			log("  %-5d  %-45s  %s, %s  (sched=%d, class=%q)",
				id, entry.Name, entry.City, entry.State, entry.ScheduleID, entry.BookingClass)
			time.Sleep(50 * time.Millisecond)
		}

		idx.ScannedTo = id

		// Save checkpoint & progress every 500
		if (id-startID+1)%500 == 0 {
			saveIndex(idx)
			elapsed := time.Since(startTime)
			rate := float64(id-startID+1) / elapsed.Seconds()
			eta := time.Duration(float64(endID-id) / rate * float64(time.Second))
			log("  --- checkpoint: %d/%d scanned, %d new courses, %.0f/sec, ETA %s ---",
				id-startID+1, totalIDs, newCount, rate, eta.Round(time.Second))
		}
	}

	saveIndex(idx)

	elapsed := time.Since(startTime)
	log("")
	log("=== Index Complete ===")
	log("Scanned %d IDs in %s", totalIDs, elapsed.Round(time.Second))
	log("New courses found: %d", newCount)
	log("Total in index: %d", len(idx.Courses))
	log("Saved to %s", indexPath)
}

// --- Match mode ---

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "golf course", "")
	s = strings.ReplaceAll(s, "golf club", "")
	s = strings.ReplaceAll(s, "golf resort", "")
	s = strings.ReplaceAll(s, "country club", "")
	s = strings.ReplaceAll(s, "golf complex", "")
	s = strings.ReplaceAll(s, "golf", "")
	s = strings.ReplaceAll(s, "the ", "")
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "'", "")
	return strings.Join(strings.Fields(s), " ")
}

func readNamesFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "|"); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			names = append(names, line)
		}
	}
	return names, scanner.Err()
}

func runMatch(state string, targets []string) {
	idx := loadIndex()
	if len(idx.Courses) == 0 {
		fmt.Fprintf(os.Stderr, "No index found. Run --index first.\n")
		os.Exit(1)
	}

	log("=== ForeUP Match ===")
	log("Index: %d courses (scanned to ID %d)", len(idx.Courses), idx.ScannedTo)
	log("State filter: %s", state)
	log("Target courses: %d", len(targets))
	log("")

	// Filter index to state
	var stateCourses []IndexEntry
	for _, c := range idx.Courses {
		if strings.EqualFold(c.State, state) {
			stateCourses = append(stateCourses, c)
		}
	}
	log("Courses in %s: %d", state, len(stateCourses))
	log("")

	matched := make(map[string]*IndexEntry)
	claimedCourseIDs := make(map[int]bool)
	for i, c := range stateCourses {
		cn := normalize(c.Name)
		for _, t := range targets {
			tn := normalize(t)
			if cn == tn || strings.Contains(cn, tn) || strings.Contains(tn, cn) {
				if _, already := matched[t]; !already && !claimedCourseIDs[c.CourseID] {
					matched[t] = &stateCourses[i]
					claimedCourseIDs[c.CourseID] = true
				}
			}
		}
	}

	// Validate matches with 3-date tee time probe
	dates := probeDates()
	type ValidatedMatch struct {
		*IndexEntry
		BookingClassID string `json:"bookingClassId"`
		DatesChecked   []string `json:"datesChecked"`
		TeeTimes       []int    `json:"teeTimes"`
		HasPrice       bool     `json:"hasPrice"`
		HasThirdParty  bool     `json:"hasThirdParty"`
		Status         string   `json:"status"`
	}

	confirmed := make(map[string]*ValidatedMatch)
	listedOnly := make(map[string]*ValidatedMatch)

	if len(matched) > 0 {
		log("=== Validating %d matches with booking page + tee time probe ===", len(matched))
		client := &http.Client{Timeout: 10 * time.Second}

		for _, t := range targets {
			c, ok := matched[t]
			if !ok {
				continue
			}

			vm := &ValidatedMatch{IndexEntry: c, DatesChecked: dates}

			// Step 1: Fetch booking page and find active booking class
			classID, hasThirdParty := fetchActiveBookingClass(client, c.CourseID, c.ScheduleID)
			if classID == "" {
				vm.Status = "listed_only"
				listedOnly[t] = vm
				log("  📋 %-45s → %-45s  (no active booking class on page)", t, c.Name)
				continue
			}
			vm.BookingClassID = classID
			vm.HasThirdParty = hasThirdParty

			// If a "third party" booking class exists, ForeUp is just the teesheet
			// backend — another platform (TeeItUp, GolfNow, etc.) handles consumer booking
			if hasThirdParty {
				vm.Status = "third_party_backend"
				listedOnly[t] = vm
				log("  🔀 %-45s → %-45s  class=%s  (third-party booking class detected — skipping)", t, c.Name, classID)
				continue
			}

			// Step 2: Probe tee times with the real booking class
			totalTimes := 0
			anyPrice := false
			for _, date := range dates {
				result := fetchTeeTimeProbe(client, c, classID, date)
				vm.TeeTimes = append(vm.TeeTimes, result.Count)
				totalTimes += result.Count
				if result.HasPrice {
					anyPrice = true
				}
				time.Sleep(300 * time.Millisecond)
			}

			vm.HasPrice = anyPrice
			if totalTimes > 0 && anyPrice {
				vm.Status = "confirmed"
				confirmed[t] = vm
				log("  ✅ %-45s → %-45s  class=%s  times=%v", t, c.Name, classID, vm.TeeTimes)
			} else if totalTimes > 0 {
				vm.Status = "listed_only"
				listedOnly[t] = vm
				log("  📋 %-45s → %-45s  class=%s  times=%v  price=NO", t, c.Name, classID, vm.TeeTimes)
			} else {
				vm.Status = "listed_only"
				listedOnly[t] = vm
				log("  📋 %-45s → %-45s  class=%s  times=%v (no tee times)", t, c.Name, classID, vm.TeeTimes)
			}
		}
		log("")
	}

	if len(confirmed) > 0 {
		log("=== Confirmed (%d) ===", len(confirmed))
		for _, t := range targets {
			if c, ok := confirmed[t]; ok {
				log("  ✅ %-45s  course_id=%-6d  sched=%d  class=%s",
					t, c.CourseID, c.ScheduleID, c.BookingClassID)
			}
		}
		log("")
	}

	if len(listedOnly) > 0 {
		// Separate third_party_backend from actual listed_only
		var thirdPartyCount int
		for _, t := range targets {
			if c, ok := listedOnly[t]; ok && c.Status == "third_party_backend" {
				thirdPartyCount++
			}
		}
		if thirdPartyCount > 0 {
			log("=== Third-Party Backend — ForeUp is teesheet only (%d) ===", thirdPartyCount)
			for _, t := range targets {
				if c, ok := listedOnly[t]; ok && c.Status == "third_party_backend" {
					log("  🔀 %-45s  course_id=%-6d  sched=%d",
						t, c.CourseID, c.ScheduleID)
				}
			}
			log("")
		}
		if len(listedOnly)-thirdPartyCount > 0 {
			log("=== Listed Only — no live tee times (%d) ===", len(listedOnly)-thirdPartyCount)
			for _, t := range targets {
				if c, ok := listedOnly[t]; ok && c.Status != "third_party_backend" {
					log("  📋 %-45s  course_id=%-6d  sched=%d",
						t, c.CourseID, c.ScheduleID)
				}
			}
			log("")
		}
	}

	var unmatched []string
	for _, t := range targets {
		if _, ok := matched[t]; !ok {
			unmatched = append(unmatched, t)
		}
	}

	if len(unmatched) > 0 {
		log("=== Unmatched (%d) ===", len(unmatched))
		for _, t := range unmatched {
			log("  ❌ %s", t)
		}
		log("")
	}

	// Show other courses in this state not in target list
	unmatchedIndex := make(map[int]bool)
	for _, c := range matched {
		unmatchedIndex[c.CourseID] = true
	}
	var other []IndexEntry
	for _, c := range stateCourses {
		if !unmatchedIndex[c.CourseID] {
			other = append(other, c)
		}
	}
	if len(other) > 0 {
		log("=== Other %s Courses on ForeUP (%d) ===", state, len(other))
		for _, c := range other {
			log("  %-45s  course_id=%-6d  sched=%-6d  %s", c.Name, c.CourseID, c.ScheduleID, c.City)
		}
		log("")
	}

	// Save match results
	os.MkdirAll("discovery/results", 0755)
	ts := time.Now().Format("2006-01-02-150405")
	outPath := fmt.Sprintf("discovery/results/foreup-%s-%s.json", strings.ToLower(state), ts)

	// Merge confirmed and listedOnly for output
	allValidated := make(map[string]*ValidatedMatch)
	for k, v := range confirmed {
		allValidated[k] = v
	}
	for k, v := range listedOnly {
		allValidated[k] = v
	}

	output := map[string]any{
		"platform":   "foreup",
		"state":      state,
		"timestamp":  time.Now().Format(time.RFC3339),
		"totalInput": len(targets),
		"confirmed":  len(confirmed),
		"listedOnly": len(listedOnly),
		"unmatched":  unmatched,
		"results":    allValidated,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, data, 0644)
	log("Results saved to %s", outPath)
}

// --- Main ---

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--index":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: --index <startId> <endId>\n")
			os.Exit(1)
		}
		startID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid startId: %s\n", os.Args[2])
			os.Exit(1)
		}
		endID, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid endId: %s\n", os.Args[3])
			os.Exit(1)
		}
		runIndex(startID, endID)

	case "--match":
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "Usage: --match <state> -f <file>\n")
			os.Exit(1)
		}
		state := strings.ToUpper(os.Args[2])
		if os.Args[3] != "-f" {
			fmt.Fprintf(os.Stderr, "Expected -f <file>\n")
			os.Exit(1)
		}
		targets, err := readNamesFromFile(os.Args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		runMatch(state, targets)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "ForeUP Discovery Tool\n\n")
	fmt.Fprintf(os.Stderr, "  --index <start> <end>           Build master index (saves to %s)\n", indexPath)
	fmt.Fprintf(os.Stderr, "  --match <state> -f <file>       Match course list against index\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  go run cmd/discover-foreup/main.go --index 1 30000\n")
	fmt.Fprintf(os.Stderr, "  go run cmd/discover-foreup/main.go --match AZ -f discovery/courses/phoenix.txt\n")
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

var bookingClassesRe = regexp.MustCompile(`"booking_classes"\s*:\s*\[`)

type BookingClass struct {
	ID           string `json:"booking_class_id"`
	Active       string `json:"active"`
	Hidden       string `json:"hidden"`
	Name         string `json:"name"`
	RequireLogin string `json:"online_booking_protected"`
}

func fetchActiveBookingClass(client *http.Client, courseID int, scheduleID int) (string, bool) {
	url := fmt.Sprintf("https://foreupsoftware.com/index.php/booking/%d/%d", courseID, scheduleID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}

	html := string(body)
	loc := bookingClassesRe.FindStringIndex(html)
	if loc == nil {
		return "", false
	}

	// Find the matching ] for the array
	start := loc[0] + len(`"booking_classes":`)
	depth := 0
	end := start
	for i := start; i < len(html); i++ {
		if html[i] == '[' {
			depth++
		} else if html[i] == ']' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	var classes []BookingClass
	if json.Unmarshal([]byte(html[start:end]), &classes) != nil {
		return "", false
	}

	// Check if any class indicates a third-party booking platform owns this course
	hasThirdParty := false
	for _, c := range classes {
		lower := strings.ToLower(c.Name)
		if strings.Contains(lower, "third party") || strings.Contains(lower, "3rd party") {
			hasThirdParty = true
			break
		}
	}

	// Find first active, non-hidden, non-login-required class; prefer "public" named
	var fallback string
	for _, c := range classes {
		if c.Active != "1" || c.Hidden == "1" {
			continue
		}
		if fallback == "" {
			fallback = c.ID
		}
		lower := strings.ToLower(c.Name)
		if strings.Contains(lower, "public") || strings.Contains(lower, "online") || strings.Contains(lower, "standard") {
			return c.ID, hasThirdParty
		}
	}
	return fallback, hasThirdParty
}

type probeResult struct {
	Count    int
	HasPrice bool
}

func fetchTeeTimeProbe(client *http.Client, entry *IndexEntry, classID string, date string) probeResult {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return probeResult{}
	}
	foreUpDate := t.Format("01-02-2006")

	url := fmt.Sprintf(
		"https://foreupsoftware.com/index.php/api/booking/times?time=all&date=%s&holes=all&players=0&booking_class=%s&schedule_id=%d&schedule_ids%%5B%%5D=%d&specials_only=0&api_key=no_limits",
		foreUpDate, classID, entry.ScheduleID, entry.ScheduleID,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return probeResult{}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://foreupsoftware.com/index.php/booking/%d", entry.CourseID))

	resp, err := client.Do(req)
	if err != nil {
		return probeResult{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return probeResult{}
	}

	var times []struct {
		GreenFee float64 `json:"green_fee"`
	}
	if json.Unmarshal(body, &times) != nil {
		return probeResult{}
	}

	hasPrice := false
	for _, tt := range times {
		if tt.GreenFee > 0 {
			hasPrice = true
			break
		}
	}
	return probeResult{Count: len(times), HasPrice: hasPrice}
}
