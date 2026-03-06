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

	"golf-teetimes/platforms"
)

// Website Discovery Tool
//
// Automates Phase 3 (Manual Gap Fill) by scanning course websites to identify
// which booking platform they use, then extracting config fields from HTML.
//
// Usage:
//   go run cmd/discover-websites/main.go <metro> <state> -f <courselist.txt>
//
// Course list format (3rd column optional):
//   Course Name | City
//   Course Name | City | https://booking-page-url.com
//
// If no URL is provided, the script searches DuckDuckGo for the course's
// booking page. Courses where no platform is identified are output as
// "unknown" for manual HAR capture.
//
// Examples:
//   go run cmd/discover-websites/main.go tampa FL -f discovery/courses/tampa.txt
//   go run cmd/discover-websites/main.go charlotte NC -f discovery/courses/charlotte.txt

// --- Platform signatures ---

type platformSig struct {
	Name    string
	Domains []string
}

var platformSigs = []platformSig{
	{"foreup", []string{"foreupsoftware.com"}},
	{"chronogolf", []string{"chronogolf.com"}},
	{"teesnap", []string{"teesnap.net"}},
	{"quick18", []string{"quick18.com"}},
	{"cpsgolf", []string{"cps.golf"}},
	{"courserev", []string{"courserev.ai"}},
	{"clubcaddie", []string{"clubcaddie.com"}},
	{"rguest", []string{"rguest.com"}},
	{"teeitup", []string{"teeitup.com", "teeitup.golf", "kenna.io"}},
	{"purposegolf", []string{"purposegolf.com"}},
	{"teequest", []string{"teequest.com"}},
	{"courseco", []string{"totaleintegrated.net"}},
	{"golfnow", []string{"golfnow.com"}},
	{"prophet", []string{"prophetservices.com"}},
	{"booktrump", []string{"booktrump.com"}},
	{"teeon", []string{"teeon.com"}},
	{"golfback", []string{"golfback.com"}},
	{"letsgogolf", []string{"letsgogolf.com"}},
	{"membersports", []string{"membersports.com"}},
	{"guestdesk", []string{"guestdesk.com"}},
	{"ezlinks", []string{"ezlinksgolf.com"}},
	{"resortsuite", []string{"resortsuite.com", "wso2wsas"}},
}

// --- Config extraction regexes ---

var (
	foreupCourseIDRe = regexp.MustCompile(`foreupsoftware\.com/index\.php/booking/index/(\d+)`)
	foreupClassRe    = regexp.MustCompile(`["']?booking_class["']?\s*[:=]\s*["']?(\w+)["']?`)
	foreupSchedRe    = regexp.MustCompile(`["']?schedule_id["']?\s*[:=]\s*["']?(\d+)["']?`)

	teeitupAliasRe = regexp.MustCompile(`([\w-]+)\.book(?:-v2)?\.teeitup\.(?:com|golf)`)
	quick18SubRe   = regexp.MustCompile(`([\w-]+)\.quick18\.com`)
	teesnapSubRe   = regexp.MustCompile(`([\w-]+)\.teesnap\.net`)
	cpsgolfSubRe   = regexp.MustCompile(`([\w-]+)\.cps\.golf`)
	courserevSubRe  = regexp.MustCompile(`([\w-]+)\.bookings\.courserev\.ai`)
	coursecoSubRe   = regexp.MustCompile(`([\w-]+)\.totaleintegrated\.net`)

	rguestRe           = regexp.MustCompile(`rguest\.com/onecart/golf/courses/(\d+)/([\w-]+)`)
	golfnowFacilityRe  = regexp.MustCompile(`golfnow\.com/tee-times/facility/(\d+)`)
	purposegolfSlugRe  = regexp.MustCompile(`purposegolf\.com/courses/([\w-]+)`)
	teequestSiteRe     = regexp.MustCompile(`teequest\.com/(\d+)`)
	clubcaddieKeyRe    = regexp.MustCompile(`clubcaddie\.com/webapi/view/(\w+)`)
	clubcaddieBaseRe   = regexp.MustCompile(`(https://[\w-]+\.clubcaddie\.com)`)
	chronogolfSlugRe   = regexp.MustCompile(`chronogolf\.com/club/([\w-]+)`)

	// Link patterns for finding booking pages
	bookingLinkRe = regexp.MustCompile(`(?i)href=["']([^"']*(?:tee.?time|book|reserv)[^"']*)["']`)
	allLinksRe    = regexp.MustCompile(`(?i)href=["']([^"']+)["']`)
	iframeSrcRe   = regexp.MustCompile(`(?i)(?:src|data-src)=["']([^"']+)["']`)
)

// --- Types ---

type courseInput struct {
	Name string
	City string
	URL  string
}

type extractedConfig struct {
	Platform string                 `json:"platform"`
	Status   string                 `json:"status"` // ready, partial, unknown, needs-url
	Config   map[string]interface{} `json:"config,omitempty"`
	Missing  []string               `json:"missing,omitempty"`
	URL      string                 `json:"url,omitempty"`
	Notes    string                 `json:"notes,omitempty"`
}

type result struct {
	Name   string          `json:"name"`
	City   string          `json:"city"`
	Result extractedConfig `json:"result"`
}

// --- Helpers ---

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.NewReplacer(
		"golf course", "", "golf club", "", "golf resort", "",
		"country club", "", "golf complex", "", "golf links", "",
		"golf center", "", " gc", "", " cc", "",
	).Replace(s)
	s = strings.TrimPrefix(s, "the ")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteByte('-')
		}
	}
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}

// --- Known course collection ---

func allKnownNames() map[string]bool {
	names := map[string]bool{}

	for _, c := range platforms.TeeItUpCourses {
		names[c.DisplayName] = true
		for _, n := range c.Names {
			names[n] = true
		}
	}
	for _, c := range platforms.GolfNowCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.ChronogolfCourses {
		for _, n := range c.Names {
			names[n] = true
		}
	}
	for _, c := range platforms.ForeUpCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.CPSGolfCourses {
		for _, n := range c.Names {
			names[n] = true
		}
	}
	for _, c := range platforms.MemberSportsCourses {
		for _, kn := range c.KnownCourses {
			names[kn] = true
		}
	}
	for _, c := range platforms.ClubCaddieCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.Quick18Courses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.CourseRevCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.RGuestCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.CourseCoCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.TeeSnapCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.ProphetCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.PurposeGolfCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.TeeQuestCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.ResortSuiteCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.BookTrumpCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.TeeOnCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.GolfBackCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.TeeTimeCentralCourses {
		names[c.DisplayName] = true
	}
	for _, c := range platforms.LetsGoGolfCourses {
		names[c.DisplayName] = true
	}
	for _, s := range platforms.GuestDeskCourses {
		for _, gc := range s.Courses {
			names[gc.DisplayName] = true
		}
	}

	return names
}

// --- Input parsing ---

func parseCourseList(path string) ([]courseInput, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var courses []courseInput
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
		u := ""
		if len(parts) > 2 {
			u = strings.TrimSpace(parts[2])
		}
		if name != "" {
			courses = append(courses, courseInput{Name: name, City: city, URL: u})
		}
	}
	return courses, scanner.Err()
}

// --- HTTP ---

func fetchHTML(client *http.Client, rawURL string) (string, string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return string(body), resp.Request.URL.String(), nil
}

// --- URL discovery ---

func searchDuckDuckGo(client *http.Client, query string) []string {
	encoded := url.QueryEscape(query)
	searchURL := "https://lite.duckduckgo.com/lite/?q=" + encoded

	html, _, err := fetchHTML(client, searchURL)
	if err != nil {
		return nil
	}

	// Extract result URLs from DuckDuckGo lite HTML
	// Results are in <a> tags with class="result-link" or just plain links after "Web results"
	re := regexp.MustCompile(`<a[^>]+href="(https?://[^"]+)"[^>]*class="result-link"`)
	matches := re.FindAllStringSubmatch(html, 5)

	// Fallback: try alternate DDG lite format
	if len(matches) == 0 {
		re = regexp.MustCompile(`<a[^>]+rel="nofollow"[^>]+href="(https?://[^"]+)"`)
		matches = re.FindAllStringSubmatch(html, 5)
	}

	var urls []string
	seen := map[string]bool{}
	for _, m := range matches {
		u := m[1]
		// Skip DDG internal links and ad links
		if strings.Contains(u, "duckduckgo.com") || strings.Contains(u, "duck.co") {
			continue
		}
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	return urls
}

func findBookingURL(client *http.Client, name, city, state string) (string, string) {
	// Strategy 1: Search for the course's tee time page
	query := fmt.Sprintf("%s %s %s tee times book online", name, city, state)
	urls := searchDuckDuckGo(client, query)

	for _, u := range urls {
		// Check if the URL itself contains a known platform domain
		for _, sig := range platformSigs {
			for _, domain := range sig.Domains {
				if strings.Contains(u, domain) {
					return u, "search"
				}
			}
		}
	}

	// Strategy 2: Fetch top results and scan for platform links
	for _, u := range urls {
		// Skip aggregators and directories
		lower := strings.ToLower(u)
		if strings.Contains(lower, "yelp.com") || strings.Contains(lower, "google.com") ||
			strings.Contains(lower, "facebook.com") || strings.Contains(lower, "tripadvisor") ||
			strings.Contains(lower, "mapquest") || strings.Contains(lower, "yellowpages") {
			continue
		}
		return u, "search"
	}

	// Strategy 3: Try common domain patterns
	slug := slugify(name)
	patterns := []string{
		"https://www." + slug + ".com",
		"https://" + slug + ".com",
		"https://www." + slug + "golf.com",
		"https://www." + strings.ReplaceAll(strings.ToLower(name), " ", "") + ".com",
	}

	for _, p := range patterns {
		req, err := http.NewRequest("HEAD", p, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			return resp.Request.URL.String(), "domain-guess"
		}
	}

	return "", ""
}

// --- Platform identification ---

func identifyPlatform(html string) (string, []string) {
	// Collect all URLs from HTML (href, src, data-src, inline URLs)
	var allURLs []string

	for _, m := range allLinksRe.FindAllStringSubmatch(html, -1) {
		allURLs = append(allURLs, m[1])
	}
	for _, m := range iframeSrcRe.FindAllStringSubmatch(html, -1) {
		allURLs = append(allURLs, m[1])
	}

	// Also check raw HTML for platform domains (catches inline JS, comments, etc.)
	combined := html + " " + strings.Join(allURLs, " ")

	for _, sig := range platformSigs {
		for _, domain := range sig.Domains {
			if strings.Contains(combined, domain) {
				// Collect all matching URLs for this platform
				var matchedURLs []string
				for _, u := range allURLs {
					if strings.Contains(u, domain) {
						matchedURLs = append(matchedURLs, u)
					}
				}
				return sig.Name, matchedURLs
			}
		}
	}

	return "", nil
}

// --- Config extractors ---

func extractForeUp(html string, urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
		"bookingUrl":  "",
	}
	var missing []string

	// Extract courseId from URL
	for _, u := range urls {
		if m := foreupCourseIDRe.FindStringSubmatch(u); m != nil {
			config["courseId"] = m[1]
		}
	}
	if m := foreupCourseIDRe.FindStringSubmatch(html); m != nil {
		config["courseId"] = m[1]
	}

	// Extract bookingClass and scheduleId from page JS
	if m := foreupClassRe.FindStringSubmatch(html); m != nil {
		config["bookingClass"] = m[1]
	}
	if m := foreupSchedRe.FindStringSubmatch(html); m != nil {
		config["scheduleId"] = m[1]
	}

	if config["courseId"] == nil {
		missing = append(missing, "courseId")
	}
	if config["bookingClass"] == nil {
		missing = append(missing, "bookingClass")
	}
	if config["scheduleId"] == nil {
		missing = append(missing, "scheduleId")
	}

	status := "ready"
	if len(missing) > 0 {
		status = "partial"
	}
	return extractedConfig{Platform: "foreup", Status: status, Config: config, Missing: missing}
}

func extractTeeItUp(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}
	var missing []string

	for _, u := range urls {
		if m := teeitupAliasRe.FindStringSubmatch(u); m != nil {
			config["alias"] = m[1]
		}
	}

	if config["alias"] == nil {
		missing = append(missing, "alias")
	}
	// facilityId is optional (API returns all courses for alias if empty)
	config["facilityId"] = ""

	status := "ready"
	if len(missing) > 0 {
		status = "partial"
	}
	return extractedConfig{Platform: "teeitup", Status: status, Config: config, Missing: missing}
}

func extractQuick18(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := quick18SubRe.FindStringSubmatch(u); m != nil {
			sub := m[1]
			config["subdomain"] = sub
			config["bookingUrl"] = "https://" + sub + ".quick18.com"
			return extractedConfig{Platform: "quick18", Status: "ready", Config: config}
		}
	}
	return extractedConfig{Platform: "quick18", Status: "partial", Config: config, Missing: []string{"subdomain"}}
}

func extractTeeSnap(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := teesnapSubRe.FindStringSubmatch(u); m != nil {
			sub := m[1]
			config["subdomain"] = sub
			config["bookingUrl"] = "https://" + sub + ".teesnap.net"
			// courseId still needs extraction from the page
			return extractedConfig{Platform: "teesnap", Status: "partial", Config: config, Missing: []string{"courseId"}}
		}
	}
	return extractedConfig{Platform: "teesnap", Status: "partial", Config: config, Missing: []string{"subdomain", "courseId"}}
}

func extractCPSGolf(urls []string, html string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
		"apiKey":      "",
		"websiteId":   "",
		"siteId":      "1",
		"courseIds":    "",
	}

	for _, u := range urls {
		if m := cpsgolfSubRe.FindStringSubmatch(u); m != nil {
			sub := m[1]
			// Check for legacy V3 interface
			if strings.Contains(u, "V3/") || strings.Contains(u, "v3/") {
				return extractedConfig{
					Platform: "cpsgolf",
					Status:   "unknown",
					Notes:    "Legacy V3 interface — NOT supported, skip",
				}
			}
			config["baseUrl"] = "https://" + sub + ".cps.golf"
			config["bookingUrl"] = "https://" + sub + ".cps.golf/onlineresweb/search-teetime"
			break
		}
	}

	// Try to extract websiteId from inline JS
	wsRe := regexp.MustCompile(`websiteId["']?\s*[:=]\s*["']([^"']+)["']`)
	if m := wsRe.FindStringSubmatch(html); m != nil {
		config["websiteId"] = m[1]
	}

	var missing []string
	if config["baseUrl"] == nil {
		missing = append(missing, "baseUrl")
	}
	if config["websiteId"] == "" {
		missing = append(missing, "websiteId")
	}

	status := "partial"
	if config["baseUrl"] != nil {
		// baseUrl found, websiteId/apiKey auto-detected at runtime
		status = "ready"
	}
	return extractedConfig{Platform: "cpsgolf", Status: status, Config: config, Missing: missing}
}

func extractCourseRev(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := courserevSubRe.FindStringSubmatch(u); m != nil {
			sub := m[1]
			config["subDomain"] = sub
			config["bookingUrl"] = "https://" + sub + ".bookings.courserev.ai/tee-times"
			// courseId needs API call to course/mco/details
			return extractedConfig{Platform: "courserev", Status: "partial", Config: config, Missing: []string{"courseId"}}
		}
	}
	return extractedConfig{Platform: "courserev", Status: "partial", Config: config, Missing: []string{"subDomain", "courseId"}}
}

func extractClubCaddie(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := clubcaddieKeyRe.FindStringSubmatch(u); m != nil {
			config["apiKey"] = m[1]
		}
		if m := clubcaddieBaseRe.FindStringSubmatch(u); m != nil {
			config["baseUrl"] = m[1]
			config["bookingUrl"] = m[1] + "/webapi/view/" + fmt.Sprint(config["apiKey"])
		}
	}

	var missing []string
	if config["apiKey"] == nil {
		missing = append(missing, "apiKey")
	}
	if config["baseUrl"] == nil {
		missing = append(missing, "baseUrl")
	}
	missing = append(missing, "courseId") // always needs HAR

	return extractedConfig{Platform: "clubcaddie", Status: "partial", Config: config, Missing: missing}
}

func extractRGuest(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := rguestRe.FindStringSubmatch(u); m != nil {
			config["tenantId"] = m[1]
			config["propertyId"] = m[2]
			config["bookingUrl"] = "https://book.rguest.com/onecart/golf/courses/" + m[1] + "/" + m[2]
			// courseId and playerTypeId need HAR/API
			return extractedConfig{Platform: "rguest", Status: "partial", Config: config, Missing: []string{"courseId", "playerTypeId"}}
		}
	}
	return extractedConfig{Platform: "rguest", Status: "partial", Config: config, Missing: []string{"tenantId", "propertyId", "courseId", "playerTypeId"}}
}

func extractGolfNow(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := golfnowFacilityRe.FindStringSubmatch(u); m != nil {
			config["facilityId"] = m[1] // Note: should be int in final JSON
			config["searchUrl"] = "https://www.golfnow.com/tee-times/facility/" + m[1] + "/search"
			config["bookingUrl"] = "https://www.golfnow.com/tee-times/facility/" + m[1] + "/search"
			return extractedConfig{Platform: "golfnow", Status: "ready", Config: config, Notes: "facilityId should be integer in JSON"}
		}
	}
	return extractedConfig{Platform: "golfnow", Status: "partial", Config: config, Missing: []string{"facilityId"}}
}

func extractChronogolf(urls []string, html string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":   slugify(name),
		"metro": metro,
		"city":  city,
		"state": state,
	}

	var slug string
	for _, u := range urls {
		if m := chronogolfSlugRe.FindStringSubmatch(u); m != nil {
			slug = m[1]
			config["bookingUrl"] = "https://www.chronogolf.com/club/" + slug
		}
	}

	// Try to extract from __NEXT_DATA__
	nextDataRe := regexp.MustCompile(`<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)
	if m := nextDataRe.FindStringSubmatch(html); m != nil {
		// Parse the JSON to extract course UUIDs and affiliationTypeId
		var nd map[string]interface{}
		if json.Unmarshal([]byte(m[1]), &nd) == nil {
			// Navigate: props.pageProps.club
			if props, ok := nd["props"].(map[string]interface{}); ok {
				if pp, ok := props["pageProps"].(map[string]interface{}); ok {
					if club, ok := pp["club"].(map[string]interface{}); ok {
						if affID, ok := club["defaultAffiliationTypeId"]; ok {
							config["affiliationTypeId"] = fmt.Sprintf("%.0f", affID)
						}
						if courses, ok := club["courses"].([]interface{}); ok {
							var uuids []string
							names := map[string]string{}
							for _, c := range courses {
								if cm, ok := c.(map[string]interface{}); ok {
									if uuid, ok := cm["uuid"].(string); ok {
										uuids = append(uuids, uuid)
									}
									if cname, ok := cm["name"].(string); ok {
										displayName := name + " - " + cname
										names[cname] = displayName
									}
								}
							}
							config["courseIds"] = strings.Join(uuids, ",")
							if len(names) > 0 {
								config["names"] = names
							}
						}
					}
				}
			}
		}
	}

	var missing []string
	if slug == "" {
		missing = append(missing, "bookingUrl")
	}
	if config["courseIds"] == nil {
		missing = append(missing, "courseIds")
		config["courseIds"] = ""
	}
	if config["affiliationTypeId"] == nil {
		missing = append(missing, "affiliationTypeId")
		config["affiliationTypeId"] = ""
	}
	if config["names"] == nil {
		config["names"] = map[string]string{}
	}
	config["clubId"] = ""
	config["numericCourseId"] = ""

	status := "ready"
	if len(missing) > 0 {
		status = "partial"
	}
	return extractedConfig{Platform: "chronogolf", Status: status, Config: config, Missing: missing}
}

func extractPurposeGolf(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := purposegolfSlugRe.FindStringSubmatch(u); m != nil {
			config["slug"] = m[1]
			config["bookingUrl"] = "https://booking.purposegolf.com/courses/" + m[1] + "/teetimes"
			// courseId (int) needs to be found from the page
			return extractedConfig{Platform: "purposegolf", Status: "partial", Config: config, Missing: []string{"courseId"}}
		}
	}
	return extractedConfig{Platform: "purposegolf", Status: "partial", Config: config, Missing: []string{"slug", "courseId"}}
}

func extractTeeQuest(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := teequestSiteRe.FindStringSubmatch(u); m != nil {
			config["siteId"] = m[1]
			config["bookingUrl"] = "https://teetimes.teequest.com/" + m[1] + "?paymentTab=pay-at-course"
			// courseTag needs to be found from the page
			return extractedConfig{Platform: "teequest", Status: "partial", Config: config, Missing: []string{"courseTag"}}
		}
	}
	return extractedConfig{Platform: "teequest", Status: "partial", Config: config, Missing: []string{"siteId", "courseTag"}}
}

func extractCourseCo(urls []string, name, city, state, metro string) extractedConfig {
	config := map[string]interface{}{
		"key":         slugify(name),
		"metro":       metro,
		"displayName": name,
		"city":        city,
		"state":       state,
	}

	for _, u := range urls {
		if m := coursecoSubRe.FindStringSubmatch(u); m != nil {
			config["subdomain"] = m[1]
			// courseId and gateway/origin URLs need HAR
			return extractedConfig{Platform: "courseco", Status: "partial", Config: config, Missing: []string{"courseId", "bookingUrl", "gatewayUrl", "originUrl"}}
		}
	}
	return extractedConfig{Platform: "courseco", Status: "partial", Config: config, Missing: []string{"subdomain", "courseId"}}
}

func extractGeneric(platform string, name, city, state, metro string) extractedConfig {
	return extractedConfig{
		Platform: platform,
		Status:   "partial",
		Config: map[string]interface{}{
			"key":         slugify(name),
			"metro":       metro,
			"displayName": name,
			"city":        city,
			"state":       state,
		},
		Notes:   "Platform identified but config extraction not automated — capture HAR",
		Missing: []string{"all config fields"},
	}
}

// --- Main discovery logic ---

func discoverCourse(client *http.Client, c courseInput, metro, state string) result {
	res := result{Name: c.Name, City: c.City}

	// Step 1: Get a URL to scan
	targetURL := c.URL
	source := "provided"
	if targetURL == "" {
		targetURL, source = findBookingURL(client, c.Name, c.City, state)
		if targetURL == "" {
			res.Result = extractedConfig{Status: "needs-url", Notes: "Could not find website — add URL as 3rd column and re-run"}
			return res
		}
	}

	log("  URL: %s (%s)", targetURL, source)

	// Step 2: Fetch the page
	html, finalURL, err := fetchHTML(client, targetURL)
	if err != nil {
		log("  Fetch error: %v", err)
		res.Result = extractedConfig{Status: "needs-url", Notes: fmt.Sprintf("Fetch failed: %v — try a different URL", err)}
		return res
	}
	if finalURL != targetURL {
		log("  Redirected to: %s", finalURL)
	}

	// Step 3: Scan page for platform signatures
	platform, matchedURLs := identifyPlatform(html)

	// Step 4: If not found on homepage, follow booking-related links (one level deep)
	if platform == "" {
		var bookingLinks []string

		// Find booking links in page HTML
		for _, m := range bookingLinkRe.FindAllStringSubmatch(html, -1) {
			link := m[1]
			if strings.HasPrefix(link, "/") {
				parsed, err := url.Parse(finalURL)
				if err == nil {
					link = parsed.Scheme + "://" + parsed.Host + link
				}
			}
			if strings.HasPrefix(link, "http") {
				bookingLinks = append(bookingLinks, link)
			}
		}

		// Also check all links for platform domains
		for _, m := range allLinksRe.FindAllStringSubmatch(html, -1) {
			link := m[1]
			for _, sig := range platformSigs {
				for _, domain := range sig.Domains {
					if strings.Contains(link, domain) {
						platform = sig.Name
						matchedURLs = append(matchedURLs, link)
					}
				}
			}
			if platform != "" {
				break
			}
		}

		// Follow up to 3 booking links
		if platform == "" {
			seen := map[string]bool{finalURL: true}
			for i, link := range bookingLinks {
				if i >= 3 || seen[link] {
					continue
				}
				seen[link] = true
				log("  Following booking link: %s", link)

				subHTML, _, err := fetchHTML(client, link)
				if err != nil {
					continue
				}
				time.Sleep(500 * time.Millisecond)

				platform, matchedURLs = identifyPlatform(subHTML)
				if platform != "" {
					html = subHTML // use this page for config extraction
					break
				}
			}
		}
	}

	// Step 5: Extract config based on identified platform
	if platform == "" {
		res.Result = extractedConfig{Status: "unknown", URL: finalURL, Notes: "No known platform found in page source — needs manual HAR"}
		return res
	}

	log("  Platform: %s", platform)

	switch platform {
	case "foreup":
		res.Result = extractForeUp(html, matchedURLs, c.Name, c.City, state, metro)
	case "teeitup":
		res.Result = extractTeeItUp(matchedURLs, c.Name, c.City, state, metro)
	case "quick18":
		res.Result = extractQuick18(matchedURLs, c.Name, c.City, state, metro)
	case "teesnap":
		res.Result = extractTeeSnap(matchedURLs, c.Name, c.City, state, metro)
	case "cpsgolf":
		res.Result = extractCPSGolf(matchedURLs, html, c.Name, c.City, state, metro)
	case "courserev":
		res.Result = extractCourseRev(matchedURLs, c.Name, c.City, state, metro)
	case "clubcaddie":
		res.Result = extractClubCaddie(matchedURLs, c.Name, c.City, state, metro)
	case "rguest":
		res.Result = extractRGuest(matchedURLs, c.Name, c.City, state, metro)
	case "golfnow":
		res.Result = extractGolfNow(matchedURLs, c.Name, c.City, state, metro)
	case "chronogolf":
		res.Result = extractChronogolf(matchedURLs, html, c.Name, c.City, state, metro)
	case "purposegolf":
		res.Result = extractPurposeGolf(matchedURLs, c.Name, c.City, state, metro)
	case "teequest":
		res.Result = extractTeeQuest(matchedURLs, c.Name, c.City, state, metro)
	case "courseco":
		res.Result = extractCourseCo(matchedURLs, c.Name, c.City, state, metro)
	case "prophet":
		res.Result = extractedConfig{Platform: "prophet", Status: "partial", Notes: "Prophet identified but DISABLED (AWS WAF blocks requests)"}
	default:
		res.Result = extractGeneric(platform, c.Name, c.City, state, metro)
	}

	res.Result.URL = finalURL
	return res
}

// --- Main ---

func main() {
	if len(os.Args) < 5 {
		printUsage()
		os.Exit(1)
	}

	metro := os.Args[1]
	state := strings.ToUpper(os.Args[2])
	if os.Args[3] != "-f" {
		printUsage()
		os.Exit(1)
	}
	courseFile := os.Args[4]

	// Build known course names
	known := allKnownNames()

	// Parse course list
	courses, err := parseCourseList(courseFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", courseFile, err)
		os.Exit(1)
	}

	// Filter to uncovered courses
	var uncovered []courseInput
	var coveredCount int
	for _, c := range courses {
		if known[c.Name] {
			coveredCount++
		} else {
			uncovered = append(uncovered, c)
		}
	}

	log("=== Website Discovery ===")
	log("Metro: %s  State: %s", metro, state)
	log("Total courses: %d  Already covered: %d  To discover: %d", len(courses), coveredCount, len(uncovered))
	log("")

	if len(uncovered) == 0 {
		log("All courses already covered!")
		return
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var ready, partial, needsURL, unknown []result

	for i, c := range uncovered {
		log("[%d/%d] %s (%s)", i+1, len(uncovered), c.Name, c.City)

		res := discoverCourse(client, c, metro, state)

		switch res.Result.Status {
		case "ready":
			ready = append(ready, res)
			log("  ✅ READY: %s", res.Result.Platform)
		case "partial":
			partial = append(partial, res)
			log("  🔶 PARTIAL: %s — missing: %s", res.Result.Platform, strings.Join(res.Result.Missing, ", "))
		case "needs-url":
			needsURL = append(needsURL, res)
			log("  🔗 NEEDS URL")
		default:
			unknown = append(unknown, res)
			log("  ❌ UNKNOWN")
		}

		log("")
		time.Sleep(1 * time.Second)
	}

	// --- Summary ---
	log("════════════════════════════════════════")
	log("  RESULTS SUMMARY")
	log("════════════════════════════════════════")
	log("")

	if len(ready) > 0 {
		log("=== READY — append to JSON (%d) ===", len(ready))
		for _, r := range ready {
			data, _ := json.MarshalIndent(r.Result.Config, "  ", "  ")
			log("  [%s.json] %s", r.Result.Platform, r.Name)
			log("  %s", string(data))
			log("")
		}
	}

	if len(partial) > 0 {
		log("=== PARTIAL — platform identified, needs more fields (%d) ===", len(partial))
		for _, r := range partial {
			log("  [%s] %-45s  missing: %s", r.Result.Platform, r.Name, strings.Join(r.Result.Missing, ", "))
			if r.Result.Notes != "" {
				log("         note: %s", r.Result.Notes)
			}
		}
		log("")
	}

	if len(needsURL) > 0 {
		log("=== NEEDS URL — add URL as 3rd column in course list and re-run (%d) ===", len(needsURL))
		for _, r := range needsURL {
			log("  %-45s  %s", r.Name, r.Result.Notes)
		}
		log("")
	}

	if len(unknown) > 0 {
		log("=== UNKNOWN — needs manual HAR capture (%d) ===", len(unknown))
		for _, r := range unknown {
			url := ""
			if r.Result.URL != "" {
				url = " → " + r.Result.URL
			}
			log("  %-45s%s", r.Name, url)
		}
		log("")
	}

	log("Ready: %d  Partial: %d  Needs URL: %d  Unknown: %d", len(ready), len(partial), len(needsURL), len(unknown))

	// Save results
	os.MkdirAll("discovery/results", 0755)
	ts := time.Now().Format("2006-01-02-150405")
	outPath := fmt.Sprintf("discovery/results/websites-%s-%s.json", metro, ts)

	output := map[string]any{
		"metro":     metro,
		"state":     state,
		"timestamp": time.Now().Format(time.RFC3339),
		"summary": map[string]int{
			"total":    len(uncovered),
			"ready":    len(ready),
			"partial":  len(partial),
			"needsUrl": len(needsURL),
			"unknown":  len(unknown),
		},
		"ready":    ready,
		"partial":  partial,
		"needsUrl": needsURL,
		"unknown":  unknown,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile(outPath, data, 0644)
	log("Results saved to %s", outPath)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Website Discovery Tool — Automates Phase 3 (Manual Gap Fill)\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  go run cmd/discover-websites/main.go <metro> <state> -f <courselist.txt>\n\n")
	fmt.Fprintf(os.Stderr, "Course list format (3rd column optional):\n")
	fmt.Fprintf(os.Stderr, "  Course Name | City\n")
	fmt.Fprintf(os.Stderr, "  Course Name | City | https://booking-url.com\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  go run cmd/discover-websites/main.go tampa FL -f discovery/courses/tampa.txt\n")
	fmt.Fprintf(os.Stderr, "  go run cmd/discover-websites/main.go charlotte NC -f discovery/courses/charlotte.txt\n")
}
