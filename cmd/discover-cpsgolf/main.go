package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// CPS Golf Discovery Tool
//
// Probes courses by guessing CPS Golf subdomains from the course name.
// CPS Golf uses {subdomain}.cps.golf with an Angular SPA and REST API.
//
// Discovery flow:
//   1. Probe {slug}.cps.golf/onlineresweb/Home/Configuration (no auth needed)
//   2. If valid JSON → extract apiKey, siteName
//   3. Call /OnlineCourses with headers → get websiteId, courseIds, timezone
//   4. Call /TeeTimes on 3 dates → confirm vs listed_only
//
// Usage: go run cmd/discover-cpsgolf/main.go <state> -f <file>

// --- Configuration response from CPS Golf ---

type CPSConfig struct {
	ClientID    string `json:"clientId"`
	Production  bool   `json:"production"`
	OnlineAPI   string `json:"onlineApi"`
	SiteName    string `json:"siteName"`
	APIKey      string `json:"apiKey"`
	BuildNumber string `json:"buildNumber"`
	BaseURL     string // computed: https://{slug}.cps.golf
	WebSiteID   string // extracted from GetAllOptions
	SiteID      int    // extracted from GetAllOptions reservationOptions.siteId
}

type CPSCourse struct {
	WebsiteID    string `json:"websiteId"`
	TimezoneID   string `json:"timezoneId"`
	StoreID      int    `json:"storeId"`
	CourseID     int    `json:"courseId"`
	CourseName   string `json:"courseName"`
	SiteID       int    `json:"siteId"`
	Holes        int    `json:"holes"`
	Active       bool   `json:"active"`
}

type CPSTeeTimeResponse struct {
	TransactionID string          `json:"transactionId"`
	IsSuccess     bool            `json:"isSuccess"`
	Content       json.RawMessage `json:"content"`
}

type CPSTeeTimeSlot struct {
	StartTime    string `json:"startTime"`
	Participants int    `json:"participants"`
	CourseID     int    `json:"courseId"`
	Holes        int    `json:"holes"`
}

type CPSGetAllOptionsResponse struct {
	WebSiteID          string `json:"webSiteId"`
	ReservationOptions struct {
		WebSiteID string `json:"webSiteId"`
		SiteID    int    `json:"siteId"`
		PageTitle string `json:"pageTitle"`
	} `json:"reservationOptions"`
	CourseOptions []struct {
		CourseName string `json:"courseName"`
		CourseID   int    `json:"courseId"`
	} `json:"courseOptions"`
}

// --- Result types ---

type Result struct {
	Input        string      `json:"input"`
	City         string      `json:"city"`
	Status       string      `json:"status"` // "confirmed", "listed_only", "miss"
	Slug         string      `json:"slug,omitempty"`
	SlugSource   string      `json:"slugSource,omitempty"`
	Config       *CPSConfig  `json:"config,omitempty"`
	Courses      []CPSCourse `json:"courses,omitempty"`
	DatesChecked []string    `json:"datesChecked,omitempty"`
	TeeTimes     []int       `json:"teeTimes,omitempty"`
	HasPrice     bool        `json:"hasPrice,omitempty"`
	SlugsTried   []string    `json:"slugsTried,omitempty"`
}

// --- Helpers ---

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

func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Golf Links", " Golf Center", " Country Club",
		" Golf & Country Club", " Golf and Country Club",
		" Golf Courses", " Golf Preserve",
		" GC", " CC",
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

// buildSlugs generates candidate CPS Golf subdomains from a course name.
// CPS Golf uses simple lowercase joined slugs (e.g. "indiantree" for Indian Tree Golf Club).
func buildSlugs(name, city string) []struct{ slug, source string } {
	seen := map[string]bool{}
	var slugs []struct{ slug, source string }
	add := func(s, source string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		slugs = append(slugs, struct{ slug, source string }{s, source})
	}

	core := coreName(name)
	coreAlpha := joinAlpha(core)
	fullAlpha := joinAlpha(name)

	// 1. Core name joined: indiantree
	add(coreAlpha, "core")

	// 2. Full name joined: indiantreegolfclub
	add(fullAlpha, "full")

	// 3. Core + golf: indiantreegolf
	add(coreAlpha+"golf", "core+golf")

	// 4. Core + golfclub: indiantreegolfclub
	add(coreAlpha+"golfclub", "core+golfclub")

	// 5. Core + golfcourse: indiantreegolfcourse
	add(coreAlpha+"golfcourse", "core+golfcourse")

	// 6. Core + cc: indiantreecc
	add(coreAlpha+"cc", "core+cc")

	// 7. Full name with "and" stripped
	noAnd := strings.ReplaceAll(name, " and ", " ")
	if noAnd != name {
		add(joinAlpha(noAnd), "full-no-and")
		noAndCore := coreName(noAnd)
		add(joinAlpha(noAndCore), "core-no-and")
	}

	// 8. Without "the" prefix
	withoutThe := strings.TrimPrefix(fullAlpha, "the")
	if withoutThe != fullAlpha {
		add(withoutThe, "no-the")
	}
	coreWithoutThe := strings.TrimPrefix(coreAlpha, "the")
	if coreWithoutThe != coreAlpha {
		add(coreWithoutThe, "core-no-the")
	}

	// 9. Hyphenated core: indian-tree
	coreWords := strings.Fields(core)
	if len(coreWords) > 1 {
		add(strings.Join(coreWords, "-"), "core-hyphenated")
	}

	// 10. Base facility name when " - " present: "Earlywine Golf Club - North" → earlywine
	if idx := strings.Index(name, " - "); idx > 0 {
		base := strings.TrimSpace(name[:idx])
		baseCore := coreName(base)
		baseCoreAlpha := joinAlpha(baseCore)
		add(baseCoreAlpha, "base-core")
		add(joinAlpha(base), "base-full")
		add(baseCoreAlpha+"golf", "base-core+golf")
		add(baseCoreAlpha+"golfclub", "base-core+golfclub")
		add(baseCoreAlpha+"golfcourse", "base-core+golfcourse")
	}

	// 11. Hyphenated full: indian-tree-golf-club
	words := strings.Fields(strings.ToLower(name))

	// 12. City-suffix variants: lincolnpark → lincolnparkokc
	if city != "" {
		cityAlpha := joinAlpha(city)
		// Common abbreviations for well-known cities
		abbrevs := []string{cityAlpha}
		switch strings.ToLower(city) {
		case "oklahoma city":
			abbrevs = append(abbrevs, "okc")
		case "new york":
			abbrevs = append(abbrevs, "nyc")
		case "kansas city":
			abbrevs = append(abbrevs, "kc")
		case "salt lake city":
			abbrevs = append(abbrevs, "slc")
		case "los angeles":
			abbrevs = append(abbrevs, "la")
		}
		for _, ca := range abbrevs {
			add(coreAlpha+ca, "core+city")
			// Also try base name + city for " - " courses
			if idx := strings.Index(name, " - "); idx > 0 {
				base := strings.TrimSpace(name[:idx])
				add(joinAlpha(coreName(base))+ca, "base-core+city")
			}
		}
	}

	// 13. Hyphenated full: indian-tree-golf-club
	cleaned := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ReplaceAll(w, "'", "")
		w = strings.ReplaceAll(w, "\u2019", "")
		w = strings.ReplaceAll(w, ".", "")
		if w != "" {
			cleaned = append(cleaned, w)
		}
	}
	add(strings.Join(cleaned, "-"), "full-hyphenated")

	return slugs
}

// --- Probing functions ---

// probeConfig checks if a CPS Golf subdomain exists by hitting the Configuration endpoint.
// This needs no auth headers and returns JSON if the site exists.
func probeConfig(client *http.Client, slug string) (*CPSConfig, error) {
	configURL := fmt.Sprintf("https://%s.cps.golf/onlineresweb/Home/Configuration", slug)

	req, err := http.NewRequest("GET", configURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var config CPSConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("not JSON: %v", err)
	}

	// Validate it's actually a CPS Golf config
	if config.SiteName == "" {
		return nil, fmt.Errorf("missing siteName")
	}

	config.BaseURL = fmt.Sprintf("https://%s.cps.golf", slug)

	return &config, nil
}

func setCPSHeaders(req *http.Request, config *CPSConfig, websiteID, siteID, timezone string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if config.APIKey != "" {
		req.Header.Set("x-apikey", config.APIKey)
	}
	req.Header.Set("x-websiteid", websiteID)
	req.Header.Set("x-siteid", siteID)
	req.Header.Set("x-componentid", "1")
	req.Header.Set("x-moduleid", "7")
	req.Header.Set("x-productid", "1")
	req.Header.Set("x-terminalid", "3")
	req.Header.Set("x-timezone-offset", "420")
	if timezone == "" {
		timezone = "America/Denver"
	}
	req.Header.Set("x-timezoneid", timezone)
	req.Header.Set("x-ismobile", "false")
	req.Header.Set("client-id", "onlineresweb")
	req.Header.Set("Referer", config.BaseURL+"/onlineresweb/search-teetime")
	req.Header.Set("Origin", config.BaseURL)
}

// fetchGetAllOptions calls GetAllOptions/{siteName} to extract the real webSiteId and siteId.
// This endpoint works with the placeholder websiteId from Configuration.
func fetchGetAllOptions(client *http.Client, config *CPSConfig) (string, int, error) {
	version := strings.TrimSpace(config.BuildNumber)
	optionsURL := fmt.Sprintf("%s/onlineres/onlineapi/api/v1/onlinereservation/GetAllOptions/%s?version=%s&product=3",
		config.BaseURL, config.SiteName, url.QueryEscape(version))

	req, err := http.NewRequest("GET", optionsURL, nil)
	if err != nil {
		return "", 0, err
	}
	setCPSHeaders(req, config, "00000000-0000-0000-0000-000000000000", "1", "")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var data CPSGetAllOptionsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", 0, fmt.Errorf("parse error: %v", err)
	}

	// Try top-level webSiteId first, then reservationOptions.webSiteId
	wsid := data.WebSiteID
	if wsid == "" || wsid == "00000000-0000-0000-0000-000000000000" {
		wsid = data.ReservationOptions.WebSiteID
	}
	if wsid == "" || wsid == "00000000-0000-0000-0000-000000000000" {
		return "", 0, fmt.Errorf("no webSiteId found")
	}

	siteID := data.ReservationOptions.SiteID
	if siteID == 0 {
		siteID = 1 // fallback
	}

	return wsid, siteID, nil
}

// fetchCourses calls the OnlineCourses endpoint to get course details.
func fetchCourses(client *http.Client, config *CPSConfig) ([]CPSCourse, error) {
	coursesURL := config.BaseURL + "/onlineres/onlineapi/api/v1/onlinereservation/OnlineCourses"

	req, err := http.NewRequest("GET", coursesURL, nil)
	if err != nil {
		return nil, err
	}
	// OnlineCourses requires the real websiteId and siteId in the header
	setCPSHeaders(req, config, config.WebSiteID, fmt.Sprintf("%d", config.SiteID), "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var courses []CPSCourse
	if err := json.Unmarshal(body, &courses); err != nil {
		return nil, err
	}
	return courses, nil
}

func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()>>32&0xFFFF,
		0x4000|(time.Now().UnixNano()>>48&0x0FFF),
		0x8000|(time.Now().UnixNano()>>60&0x3FFF),
		time.Now().UnixNano()&0xFFFFFFFFFFFF,
	)
}

// fetchTeeTimeCount registers a transaction and fetches tee times for a date.
func fetchTeeTimeCount(client *http.Client, config *CPSConfig, courses []CPSCourse, date string) (int, bool, error) {
	websiteID := config.WebSiteID
	siteID := fmt.Sprintf("%d", config.SiteID)
	timezone := ""
	if len(courses) > 0 {
		siteID = fmt.Sprintf("%d", courses[0].SiteID)
		timezone = courses[0].TimezoneID
	}

	// Build courseIds from OnlineCourses response
	var courseIDs []string
	for _, c := range courses {
		courseIDs = append(courseIDs, fmt.Sprintf("%d", c.CourseID))
	}
	courseIDsParam := strings.Join(courseIDs, ",")

	// Step 1: Register transaction ID
	txnID := generateUUID()
	txnBody, _ := json.Marshal(map[string]string{"transactionId": txnID})

	txnReq, err := http.NewRequest("POST",
		config.BaseURL+"/onlineres/onlineapi/api/v1/onlinereservation/RegisterTransactionId",
		bytes.NewBuffer(txnBody))
	if err != nil {
		return 0, false, err
	}
	txnReq.Header.Set("Content-Type", "application/json")
	setCPSHeaders(txnReq, config, websiteID, siteID, timezone)

	txnResp, err := client.Do(txnReq)
	if err != nil {
		return 0, false, err
	}
	txnResp.Body.Close()

	// Step 2: Fetch tee times
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, false, err
	}
	searchDate := url.PathEscape(t.Format("Mon Jan 02 2006"))

	teeURL := fmt.Sprintf(
		"%s/onlineres/onlineapi/api/v1/onlinereservation/TeeTimes?searchDate=%s&holes=0&numberOfPlayer=0&courseIds=%s&searchTimeType=0&transactionId=%s&teeOffTimeMin=0&teeOffTimeMax=23&isChangeTeeOffTime=true&teeSheetSearchView=5&classCode=R&defaultOnlineRate=N&isUseCapacityPricing=false&memberStoreId=1&searchType=1",
		config.BaseURL, searchDate, courseIDsParam, txnID,
	)

	req, err := http.NewRequest("GET", teeURL, nil)
	if err != nil {
		return 0, false, err
	}
	setCPSHeaders(req, config, websiteID, siteID, timezone)

	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, err
	}

	var data CPSTeeTimeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, false, nil
	}

	var slots []CPSTeeTimeSlot
	if err := json.Unmarshal(data.Content, &slots); err != nil {
		return 0, false, nil
	}

	// Check if any slot has price data (use full response for this)
	hasPrice := strings.Contains(string(body), "displayPrice")

	return len(slots), hasPrice, nil
}

// --- State validation via timezone ---

// timezoneStates maps common timezones to states they belong in.
// A CPS Golf course's timezone from OnlineCourses can validate the state.
var timezoneStates = map[string][]string{
	"America/Denver":     {"CO", "MT", "WY", "NM", "UT", "AZ"},
	"America/Phoenix":    {"AZ"},
	"America/Chicago":    {"TX", "IL", "MN", "WI", "IA", "MO", "AR", "LA", "MS", "AL", "TN", "KS", "NE", "SD", "ND", "OK"},
	"America/New_York":   {"NY", "NJ", "PA", "CT", "MA", "NH", "VT", "ME", "RI", "VA", "WV", "NC", "SC", "GA", "FL", "OH", "MI", "IN", "KY", "MD", "DE", "DC"},
	"America/Los_Angeles": {"CA", "WA", "OR", "NV"},
	"America/Boise":      {"ID", "OR"},
	"America/Anchorage":  {"AK"},
	"Pacific/Honolulu":   {"HI"},
}

func validateTimezone(timezone, targetState string) bool {
	if timezone == "" {
		return true // no data — benefit of the doubt
	}
	states, ok := timezoneStates[timezone]
	if !ok {
		return true // unknown timezone — benefit of the doubt
	}
	for _, s := range states {
		if s == targetState {
			return true
		}
	}
	return false
}

// --- File reading ---

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

// --- Main ---

func main() {
	if len(os.Args) < 4 || os.Args[2] != "-f" {
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/discover-cpsgolf/main.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run cmd/discover-cpsgolf/main.go CO -f discovery/courses/denver.txt\n")
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

	log("=== CPS Golf Discovery ===")
	log("State: %s", state)
	log("Courses to probe: %d", len(courses))
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	client := &http.Client{Timeout: 10 * time.Second}

	var results []Result
	confirmed, missed, listedOnly, wrongState := 0, 0, 0, 0
	deadSlugs := map[string]bool{}

	for i, c := range courses {
		slugs := buildSlugs(c.Name, c.City)

		// Filter known-dead slugs
		var live []struct{ slug, source string }
		for _, s := range slugs {
			if !deadSlugs[s.slug] {
				live = append(live, s)
			}
		}

		slugNames := make([]string, len(slugs))
		for j, s := range slugs {
			slugNames[j] = s.slug
		}

		log("[%d/%d] %q (city: %q) — %d slug candidates", i+1, len(courses), c.Name, c.City, len(live))

		// Phase 1: Find a valid CPS Golf subdomain
		var config *CPSConfig
		var matchedSlug, matchedSource string

		for _, s := range live {
			cfg, err := probeConfig(client, s.slug)
			if err != nil {
				deadSlugs[s.slug] = true
				time.Sleep(100 * time.Millisecond)
				continue
			}
			apiKeyLog := "(none)"
			if cfg.APIKey != "" {
				apiKeyLog = cfg.APIKey[:8] + "..."
			}
			log("  %s (%s): HIT — siteName=%q apiKey=%s", s.slug, s.source, cfg.SiteName, apiKeyLog)
			config = cfg
			matchedSlug = s.slug
			matchedSource = s.source
			break
		}

		if config == nil {
			missed++
			log("  MISS — no slug matched")
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "miss", SlugsTried: slugNames,
			})
			log("")
			continue
		}

		// Phase 2: Get real websiteId and siteId via GetAllOptions
		wsid, siteID, err := fetchGetAllOptions(client, config)
		if err != nil {
			log("  ⚠️  Failed to get webSiteId: %v — skipping", err)
			missed++
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "miss", SlugsTried: slugNames,
			})
			log("")
			continue
		}
		config.WebSiteID = wsid
		config.SiteID = siteID
		log("  webSiteId=%s  siteId=%d", wsid, siteID)

		// Phase 3: Fetch course details
		cpsCourses, err := fetchCourses(client, config)
		if err != nil {
			log("  ⚠️  Failed to fetch courses: %v — treating as listed_only", err)
			listedOnly++
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "listed_only",
				Slug: matchedSlug, SlugSource: matchedSource, Config: config,
			})
			log("")
			continue
		}

		// Log courses found
		courseDescs := make([]string, len(cpsCourses))
		for j, cc := range cpsCourses {
			courseDescs[j] = fmt.Sprintf("%s (%dh)", cc.CourseName, cc.Holes)
		}
		log("  courses=%v", courseDescs)

		// State validation via timezone
		if len(cpsCourses) > 0 {
			tz := cpsCourses[0].TimezoneID
			if !validateTimezone(tz, state) {
				wrongState++
				log("  WRONG STATE — timezone %q doesn't match %s", tz, state)
				results = append(results, Result{
					Input: c.Name, City: c.City, Status: "wrong_state",
					Slug: matchedSlug, SlugSource: matchedSource,
					Config: config, Courses: cpsCourses,
				})
				log("")
				continue
			}
		}

		// Phase 4: 3-date tee time validation
		var datesChecked []string
		var teeTimes []int
		totalTimes := 0
		anyPrice := false

		for _, date := range dates {
			count, hasPrice, err := fetchTeeTimeCount(client, config, cpsCourses, date)
			if err != nil {
				log("    %s: error — %v", date, err)
				count = 0
			}
			log("    %s: %d tee times", date, count)
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
			log("  ✅ CONFIRMED — %d total tee times  courses=%v", totalTimes, courseDescs)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "confirmed",
				Slug: matchedSlug, SlugSource: matchedSource,
				Config: config, Courses: cpsCourses,
				DatesChecked: datesChecked, TeeTimes: teeTimes, HasPrice: anyPrice,
			})
		} else {
			listedOnly++
			log("  ⚠️  LISTED ONLY — 0 tee times  courses=%v", courseDescs)
			results = append(results, Result{
				Input: c.Name, City: c.City, Status: "listed_only",
				Slug: matchedSlug, SlugSource: matchedSource,
				Config: config, Courses: cpsCourses,
				DatesChecked: datesChecked, TeeTimes: teeTimes,
			})
		}
		log("")

		time.Sleep(200 * time.Millisecond)
	}

	elapsed := time.Since(startTime)

	log("========================================")
	log("=== SUMMARY")
	log("========================================")
	log("Total probed:  %d", len(courses))
	log("Confirmed:     %d", confirmed)
	log("Listed only:   %d", listedOnly)
	log("Wrong state:   %d", wrongState)
	log("Misses:        %d", missed)
	log("Elapsed:       %s", elapsed.Round(time.Millisecond))
	log("")

	if confirmed > 0 {
		log("=== CONFIRMED ===")
		for _, r := range results {
			if r.Status == "confirmed" {
				courseDescs := make([]string, len(r.Courses))
				for j, cc := range r.Courses {
					courseDescs[j] = fmt.Sprintf("%s (%dh)", cc.CourseName, cc.Holes)
				}
				log("  %-45s slug:%-30s [%s] %s.cps.golf  [%d times]  courses=%v",
					r.Input, r.Slug, r.SlugSource, r.Slug, sum(r.TeeTimes), courseDescs)
			}
		}
		log("")
	}

	if listedOnly > 0 {
		log("=== LISTED ONLY ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  %-45s slug:%-30s [%s] %s.cps.golf", r.Input, r.Slug, r.SlugSource, r.Slug)
			}
		}
		log("")
	}

	if wrongState > 0 {
		log("=== WRONG STATE ===")
		for _, r := range results {
			if r.Status == "wrong_state" {
				log("  %-45s slug:%-30s [%s]", r.Input, r.Slug, r.SlugSource)
			}
		}
		log("")
	}

	if missed > 0 {
		log("=== MISSES ===")
		for _, r := range results {
			if r.Status == "miss" {
				log("  %-45s (city: %s)", r.Input, r.City)
			}
		}
		log("")
	}

	// Save results
	os.MkdirAll("discovery/results", 0755)
	ts := time.Now().Format("2006-01-02-150405")
	outPath := fmt.Sprintf("discovery/results/cpsgolf-%s-%s.json", strings.ToLower(state), ts)

	output := map[string]any{
		"platform":   "cpsgolf",
		"state":      state,
		"timestamp":  time.Now().Format(time.RFC3339),
		"elapsed":    elapsed.String(),
		"total":      len(courses),
		"confirmed":  confirmed,
		"listedOnly": listedOnly,
		"wrongState": wrongState,
		"misses":     missed,
		"results":    results,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, data, 0644)
	log("Results saved to %s", outPath)
}

func sum(nums []int) int {
	t := 0
	for _, n := range nums {
		t += n
	}
	return t
}
