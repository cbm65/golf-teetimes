package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Migrates all golfwithaccess courses to either teeitup or golfnow.
//
// Usage: go run cmd/migrate-golfwithaccess/main.go [--apply]
//
// Without --apply: prints a migration plan (dry run).
// With --apply: updates platforms/data/*.json files in place.
//
// Strategy:
//   1. For each course, try TeeItUp alias probing (Kenna API)
//   2. If TeeItUp miss, try GolfNow area search
//   3. Courses that match neither are reported as unresolved

// ── TeeItUp types ──

var kennaAPI = "https://phx-api-be-east-1b.kenna.io"

type Facility struct {
	ID       int       `json:"id"`
	CourseID string    `json:"courseId"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Locality string    `json:"locality"`
	Region   string    `json:"region"`
	Location []float64 `json:"location"`
	TimeZone string    `json:"timeZone"`
}

type TeeTimeResponse struct {
	Teetimes []struct {
		Teetime string `json:"teetime"`
	} `json:"teetimes"`
}

// ── GolfNow types ──

type GolfNowSearchRequest struct {
	Radius                    int    `json:"Radius"`
	Latitude                  string `json:"Latitude"`
	Longitude                 string `json:"Longitude"`
	PageSize                  int    `json:"PageSize"`
	PageNumber                int    `json:"PageNumber"`
	SearchType                int    `json:"SearchType"`
	SortBy                    string `json:"SortBy"`
	SortDirection             string `json:"SortDirection"`
	Date                      string `json:"Date"`
	HotDealsOnly              bool   `json:"HotDealsOnly"`
	PriceMin                  string `json:"PriceMin"`
	PriceMax                  string `json:"PriceMax"`
	Players                   string `json:"Players"`
	TimePeriod                string `json:"TimePeriod"`
	Holes                     string `json:"Holes"`
	FacilityType              int    `json:"FacilityType"`
	RateType                  string `json:"RateType"`
	TimeMin                   string `json:"TimeMin"`
	TimeMax                   string `json:"TimeMax"`
	SortByRollup              string `json:"SortByRollup"`
	View                      string `json:"View"`
	ExcludeFeaturedFacilities bool   `json:"ExcludeFeaturedFacilities"`
	TeeTimeCount              int    `json:"TeeTimeCount"`
	PromotedCampaignsOnly     string `json:"PromotedCampaignsOnly"`
	CurrentClientDate         string `json:"CurrentClientDate"`
}

type GolfNowSearchResponse struct {
	TTResults struct {
		Facilities []struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Address struct {
				City              string `json:"city"`
				StateProvinceCode string `json:"stateProvinceCode"`
			} `json:"address"`
			IsSimulator bool `json:"isSimulator"`
		} `json:"facilities"`
	} `json:"ttResults"`
	Total int `json:"total"`
}

// ── Source course type ──

type GWACourse struct {
	Key         string   `json:"key"`
	Metro       string   `json:"metro"`
	CourseIDs   []string `json:"courseIds"`
	Slug        string   `json:"slug"`
	BookingURL  string   `json:"bookingUrl"`
	DisplayName string   `json:"displayName"`
	City        string   `json:"city"`
	State       string   `json:"state"`
}

// ── Migration result ──

type MigrationResult struct {
	Course      GWACourse
	Target      string // "teeitup", "golfnow", or "unresolved"
	TeeItUpData *TeeItUpMatch
	GolfNowData *GolfNowMatch
}

type TeeItUpMatch struct {
	Alias      string `json:"alias"`
	FacilityID int    `json:"facilityId"`
	Name       string `json:"name"`
	TeeTimes   int    `json:"teeTimes"`
}

type GolfNowMatch struct {
	FacilityID int    `json:"facilityId"`
	Name       string `json:"name"`
	City       string `json:"city"`
	State      string `json:"state"`
}

// ── Metro coordinates for GolfNow area search ──

var metroCoords = map[string][3]float64{
	"denver":       {39.7392, -104.9903, 35},
	"phoenix":      {33.4484, -112.0740, 40},
	"tucson":       {32.2226, -110.9747, 40},
	"lasvegas":     {36.1699, -115.1398, 25},
	"atlanta":      {33.7490, -84.3880, 35},
	"albuquerque":  {35.4000, -106.3000, 50},
	"dallas":       {32.7767, -96.7970, 40},
	"neworleans":   {29.9511, -90.0715, 30},
	"nashville":    {36.1627, -86.7816, 30},
	"miami":        {25.7617, -80.1918, 35},
	"sanfrancisco": {37.5585, -122.2711, 45},
	"oklahomacity": {35.4676, -97.5164, 35},
	"losangeles":   {34.0522, -118.2437, 35},
	"sandiego":     {32.7157, -117.1611, 40},
	"charlotte":    {35.2271, -80.8431, 35},
	"austin":       {30.2672, -97.7431, 40},
	"orlando":      {28.5383, -81.3792, 40},
	"houston":      {29.7604, -95.3698, 45},
	"tampa":        {27.9506, -82.4572, 40},
	"jacksonville": {30.3322, -81.6557, 50},
	"palmsprings":  {33.8303, -116.5453, 35},
	"sanantonio":   {29.4200, -98.4900, 40},
	"sacramento":   {38.5816, -121.4944, 40},
}

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

// ── Slug / name utils (from discover-teeitup) ──

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func coreName(name string) string {
	s := name
	for _, suffix := range []string{
		" Golf Course", " Golf Club", " Golf Resort", " Golf Complex",
		" Golf Links", " Golf Center", " Country Club", " Golf & Country Club",
		" Golf and Country Club", " Golf & Tennis", " GC", " CC",
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

func generateAliases(name, city string) []struct{ alias, source string } {
	var candidates []struct{ alias, source string }
	seen := map[string]bool{}

	add := func(alias, source string) {
		if alias == "" || seen[alias] {
			return
		}
		seen[alias] = true
		candidates = append(candidates, struct{ alias, source string }{alias, source})
	}

	exact := slugify(name)
	core := slugify(coreName(name))

	add(exact, "exact")

	if strings.HasPrefix(exact, "the-") {
		add(strings.TrimPrefix(exact, "the-"), "no-the")
	} else {
		add("the-"+exact, "add-the")
	}

	suffixes := []string{
		"golf-course", "golf-club", "country-club", "golf-resort",
		"golf-complex", "golf-links", "golf-center", "gc", "golf",
		"golf-and-tennis",
	}
	for _, oldSuffix := range suffixes {
		if strings.HasSuffix(exact, "-"+oldSuffix) {
			base := strings.TrimSuffix(exact, "-"+oldSuffix)
			for _, newSuffix := range suffixes {
				if newSuffix != oldSuffix {
					add(base+"-"+newSuffix, "swap-"+newSuffix)
				}
			}
			add(base, "base-only")
			break
		}
	}

	add(core, "core")

	for _, suffix := range []string{
		"golf-course", "golf-club", "country-club", "golf-resort", "golf",
	} {
		add(core+"-"+suffix, "core+"+suffix)
	}

	if city != "" {
		citySlug := slugify(city)
		cityLower := strings.ToLower(city)
		coreLower := strings.ToLower(coreName(name))

		if strings.HasPrefix(coreLower, cityLower+" ") {
			stripped := slugify(coreLower[len(cityLower)+1:])
			add(stripped, "strip-city-prefix")
			for _, suffix := range []string{"golf-club", "golf-course", "country-club", "golf"} {
				add(stripped+"-"+suffix, "strip-city-prefix+"+suffix)
			}
		}
		if strings.HasSuffix(coreLower, " "+cityLower) {
			stripped := slugify(coreLower[:len(coreLower)-len(cityLower)-1])
			add(stripped, "strip-city-suffix")
		}

		add(exact+"-"+citySlug, "exact+city")
		add(core+"-"+citySlug, "core+city")
		add(core+"-golf-course-"+citySlug, "core-gc+city")
		add(core+"-golf-club-"+citySlug, "core-club+city")
		add(citySlug+"-"+core, "city+core")
		add(citySlug+"-"+core+"-golf-course", "city+core-gc")
		add(citySlug+"-"+core+"-golf-club", "city+core-club")
	}

	if idx := strings.LastIndex(core, "-"); idx > 0 {
		trimmed := core[:idx]
		if len(trimmed) >= 4 {
			add(trimmed, "trim-trailing")
		}
	}

	add(core+"-public-booking-engine", "core+pbe")
	add(exact+"-public-booking-engine", "exact+pbe")
	add(core+"-gc-public-booking-engine", "core-gc+pbe")
	add(exact+"-v2", "exact+v2")
	add(core+"-v2", "core+v2")
	for _, suffix := range []string{"golf-club", "golf-course", "country-club"} {
		add(core+"-"+suffix+"-v2", "core+"+suffix+"+v2")
	}

	return candidates
}

func normalizeForMatch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, suffix := range []string{
		"golf course", "golf club", "golf resort", "golf complex",
		"golf links", "golf center", "country club", "golf & tennis",
		" gc", " cc",
	} {
		s = strings.TrimSuffix(s, " "+suffix)
	}
	for _, prefix := range []string{"the ", "golf club of ", "golf club at "} {
		s = strings.TrimPrefix(s, prefix)
	}
	return strings.TrimSpace(s)
}

func fuzzyMatch(inputName, facilityName, city string) bool {
	a := normalizeForMatch(inputName)
	b := normalizeForMatch(facilityName)
	if a == "" || b == "" {
		return false
	}
	if city != "" {
		cityLower := strings.ToLower(strings.TrimSpace(city))
		a = strings.TrimSpace(strings.ReplaceAll(a, cityLower, ""))
	}
	if a == "" {
		return false
	}
	if a == b {
		return true
	}
	shorter, longer := a, b
	if len(a) > len(b) {
		shorter, longer = b, a
	}
	if len(shorter) >= 5 && strings.Contains(longer, shorter) {
		return true
	}
	skip := map[string]bool{"at": true, "of": true, "the": true, "in": true, "and": true, "a": true}
	shorterWords := strings.Fields(shorter)
	longerSet := map[string]bool{}
	for _, w := range strings.Fields(longer) {
		longerSet[w] = true
	}
	matchCount, totalCount := 0, 0
	for _, w := range shorterWords {
		if skip[w] {
			continue
		}
		totalCount++
		if longerSet[w] {
			matchCount++
		}
	}
	return totalCount >= 2 && matchCount == totalCount
}

// ── TeeItUp probing ──

func probeFacility(alias string) ([]Facility, int, error) {
	req, err := http.NewRequest("GET", kennaAPI+"/facilities", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("x-be-alias", alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+alias+".book.teeitup.com")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, nil
	}

	var facilities []Facility
	if err := json.Unmarshal(body, &facilities); err != nil {
		return nil, resp.StatusCode, nil
	}
	return facilities, resp.StatusCode, nil
}

func probeTeeTimes(alias string, facilityID int, date string) (int, error) {
	url := fmt.Sprintf("%s/v2/tee-times?date=%s&facilityIds=%d&dateMax=%s", kennaAPI, date, facilityID, date)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-be-alias", alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+alias+".book.teeitup.com")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != 200 {
		return 0, nil
	}

	var data []TeeTimeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, nil
	}
	total := 0
	for _, d := range data {
		total += len(d.Teetimes)
	}
	return total, nil
}

func tryTeeItUp(course GWACourse) *TeeItUpMatch {
	candidates := generateAliases(course.DisplayName, course.City)
	log("  TeeItUp: trying %d alias candidates", len(candidates))

	deadAliases := map[string]bool{}
	aliasCache := map[string][]Facility{}

	for _, c := range candidates {
		if deadAliases[c.alias] {
			continue
		}

		facilities, cached := aliasCache[c.alias]
		if !cached {
			var statusCode int
			var err error
			facilities, statusCode, err = probeFacility(c.alias)
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if statusCode != 200 || len(facilities) == 0 {
				deadAliases[c.alias] = true
				time.Sleep(50 * time.Millisecond)
				continue
			}
			aliasCache[c.alias] = facilities
		}

		for _, f := range facilities {
			if !strings.EqualFold(f.Region, course.State) {
				continue
			}
			if !fuzzyMatch(course.DisplayName, f.Name, course.City) {
				continue
			}

			log("  TeeItUp: HIT — alias=%q [%s] FID:%d %s (%s, %s)", c.alias, c.source, f.ID, f.Name, f.Locality, f.Region)

			// Validate with tee time probe
			d := time.Now()
			for d.Weekday() != time.Saturday {
				d = d.AddDate(0, 0, 1)
			}
			date := d.Format("2006-01-02")
			count, _ := probeTeeTimes(c.alias, f.ID, date)
			log("  TeeItUp: %s → %d tee times", date, count)

			return &TeeItUpMatch{
				Alias:      c.alias,
				FacilityID: f.ID,
				Name:       f.Name,
				TeeTimes:   count,
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

// ── GolfNow probing ──

func getGolfNowToken(lat, lng float64) (string, *cookiejar.Jar, error) {
	url := fmt.Sprintf("https://www.golfnow.com/tee-times/search#sortby=Date&view=Grouping&lat=%.4f&lng=%.4f", lat, lng)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	re := regexp.MustCompile(`__RequestVerificationToken[^>]*value="([^"]+)"`)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1], jar, nil
	}
	re = regexp.MustCompile(`data-request-verification-token="([^"]+)"`)
	m = re.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1], jar, nil
	}
	return "", nil, fmt.Errorf("token not found in %d byte response", len(body))
}

type golfNowFacility struct {
	ID    int
	Name  string
	City  string
	State string
}

func searchGolfNowArea(lat, lng, radius float64, token string, jar *cookiejar.Jar) ([]golfNowFacility, error) {
	seen := map[int]golfNowFacility{}
	searchDate := time.Now().Add(3 * 24 * time.Hour).Format("Jan 02 2006")

	for page := 0; page <= 20; page++ {
		body := GolfNowSearchRequest{
			Radius:                    int(radius),
			Latitude:                  fmt.Sprintf("%.5f", lat),
			Longitude:                 fmt.Sprintf("%.5f", lng),
			PageSize:                  30,
			PageNumber:                page,
			SearchType:                0,
			SortBy:                    "Facilities.Distance",
			SortDirection:             "0",
			Date:                      searchDate,
			HotDealsOnly:              false,
			PriceMin:                  "0",
			PriceMax:                  "10000",
			Players:                   "0",
			TimePeriod:                "3",
			Holes:                     "3",
			FacilityType:              0,
			RateType:                  "all",
			TimeMin:                   "0",
			TimeMax:                   "48",
			SortByRollup:              "Facilities.Distance",
			View:                      "Course",
			ExcludeFeaturedFacilities: false,
			TeeTimeCount:              20,
			PromotedCampaignsOnly:     "false",
			CurrentClientDate:         time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		}

		jsonData, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "https://www.golfnow.com/api/tee-times/tee-time-results", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("Origin", "https://www.golfnow.com")
		req.Header.Set("Referer", "https://www.golfnow.com/tee-times/search")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("__requestverificationtoken", token)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")

		client := &http.Client{Jar: jar}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if len(respBody) > 0 && respBody[0] == '<' {
			return nil, fmt.Errorf("blocked by bot protection")
		}

		var data GolfNowSearchResponse
		if err := json.Unmarshal(respBody, &data); err != nil {
			return nil, err
		}

		for _, f := range data.TTResults.Facilities {
			if f.IsSimulator {
				continue
			}
			if _, exists := seen[f.ID]; !exists {
				seen[f.ID] = golfNowFacility{
					ID:    f.ID,
					Name:  f.Name,
					City:  f.Address.City,
					State: f.Address.StateProvinceCode,
				}
			}
		}

		if len(data.TTResults.Facilities) < 30 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	var result []golfNowFacility
	for _, f := range seen {
		result = append(result, f)
	}
	return result, nil
}

// normalizeGolfNow for GolfNow name matching (matches golfnow-discover.go)
func normalizeGolfNow(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " - ", " ")
	s = strings.ReplaceAll(s, "golf course", "")
	s = strings.ReplaceAll(s, "golf club", "")
	s = strings.ReplaceAll(s, "golf links", "")
	s = strings.ReplaceAll(s, "golf resort", "")
	s = strings.ReplaceAll(s, "golf center", "")
	s = strings.ReplaceAll(s, "golf & tennis", "")
	s = strings.ReplaceAll(s, "golf", "")
	s = strings.ReplaceAll(s, "the ", "")
	s = strings.ReplaceAll(s, "  ", " ")
	return strings.TrimSpace(s)
}

func tryGolfNow(course GWACourse, gnCache map[string][]golfNowFacility, tokenCache map[string][2]any) *GolfNowMatch {
	coords, ok := metroCoords[course.Metro]
	if !ok {
		log("  GolfNow: no metro coords for %q", course.Metro)
		return nil
	}

	// Use cached area results if available
	facilities, cached := gnCache[course.Metro]
	if !cached {
		log("  GolfNow: searching %s area...", course.Metro)

		var token string
		var jar *cookiejar.Jar
		if tc, ok := tokenCache[course.Metro]; ok {
			token = tc[0].(string)
			jar = tc[1].(*cookiejar.Jar)
		} else {
			var err error
			token, jar, err = getGolfNowToken(coords[0], coords[1])
			if err != nil {
				log("  GolfNow: token error: %v", err)
				gnCache[course.Metro] = nil
				return nil
			}
			tokenCache[course.Metro] = [2]any{token, jar}
		}

		var err error
		facilities, err = searchGolfNowArea(coords[0], coords[1], coords[2], token, jar)
		if err != nil {
			log("  GolfNow: search error: %v", err)
			gnCache[course.Metro] = nil
			return nil
		}
		gnCache[course.Metro] = facilities
		log("  GolfNow: found %d facilities in %s", len(facilities), course.Metro)
	}

	if facilities == nil {
		return nil
	}

	// Match by normalized name
	inputNorm := normalizeGolfNow(course.DisplayName)
	for _, f := range facilities {
		fNorm := normalizeGolfNow(f.Name)
		if inputNorm == fNorm {
			log("  GolfNow: EXACT match — facilityId=%d %s (%s, %s)", f.ID, f.Name, f.City, f.State)
			return &GolfNowMatch{FacilityID: f.ID, Name: f.Name, City: f.City, State: f.State}
		}
	}

	// Fuzzy containment
	for _, f := range facilities {
		if fuzzyMatch(course.DisplayName, f.Name, course.City) {
			log("  GolfNow: FUZZY match — facilityId=%d %s (%s, %s)", f.ID, f.Name, f.City, f.State)
			return &GolfNowMatch{FacilityID: f.ID, Name: f.Name, City: f.City, State: f.State}
		}
	}

	return nil
}

// ── Apply migration ──

func applyMigration(results []MigrationResult) error {
	// Load existing platform files
	teeitupPath := "platforms/data/teeitup.json"
	golfnowPath := "platforms/data/golfnow.json"
	gwaPath := "platforms/data/golfwithaccess.json"

	var teeitupCourses []map[string]any
	var golfnowCourses []map[string]any

	raw, _ := os.ReadFile(teeitupPath)
	json.Unmarshal(raw, &teeitupCourses)

	raw, _ = os.ReadFile(golfnowPath)
	json.Unmarshal(raw, &golfnowCourses)

	var remaining []map[string]any // courses that couldn't be migrated

	for _, r := range results {
		switch r.Target {
		case "teeitup":
			entry := map[string]any{
				"key":         r.Course.Key,
				"metro":       r.Course.Metro,
				"alias":       r.TeeItUpData.Alias,
				"facilityId":  fmt.Sprintf("%d", r.TeeItUpData.FacilityID),
				"displayName": r.Course.DisplayName,
				"city":        r.Course.City,
				"state":       r.Course.State,
			}
			teeitupCourses = append(teeitupCourses, entry)

		case "golfnow":
			entry := map[string]any{
				"key":         r.Course.Key,
				"metro":       r.Course.Metro,
				"facilityId":  r.GolfNowData.FacilityID,
				"searchUrl":   fmt.Sprintf("https://www.golfnow.com/tee-times/facility/%d/search", r.GolfNowData.FacilityID),
				"bookingUrl":  fmt.Sprintf("https://www.golfnow.com/tee-times/facility/%d/search", r.GolfNowData.FacilityID),
				"displayName": r.Course.DisplayName,
				"city":        r.Course.City,
				"state":       r.Course.State,
			}
			golfnowCourses = append(golfnowCourses, entry)

		default:
			// Keep in golfwithaccess
			entry := map[string]any{
				"key":         r.Course.Key,
				"metro":       r.Course.Metro,
				"courseIds":    r.Course.CourseIDs,
				"slug":        r.Course.Slug,
				"bookingUrl":  r.Course.BookingURL,
				"displayName": r.Course.DisplayName,
				"city":        r.Course.City,
				"state":       r.Course.State,
			}
			remaining = append(remaining, entry)
		}
	}

	// Sort each list by metro then key for consistency
	sortByMetroKey := func(courses []map[string]any) {
		sort.Slice(courses, func(i, j int) bool {
			mi, _ := courses[i]["metro"].(string)
			mj, _ := courses[j]["metro"].(string)
			if mi != mj {
				return mi < mj
			}
			ki, _ := courses[i]["key"].(string)
			kj, _ := courses[j]["key"].(string)
			return ki < kj
		})
	}

	sortByMetroKey(teeitupCourses)
	sortByMetroKey(golfnowCourses)

	// Write files
	writeJSON := func(path string, data any) error {
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, out, 0644)
	}

	if err := writeJSON(teeitupPath, teeitupCourses); err != nil {
		return fmt.Errorf("writing teeitup.json: %w", err)
	}
	if err := writeJSON(golfnowPath, golfnowCourses); err != nil {
		return fmt.Errorf("writing golfnow.json: %w", err)
	}
	if err := writeJSON(gwaPath, remaining); err != nil {
		return fmt.Errorf("writing golfwithaccess.json: %w", err)
	}

	return nil
}

func main() {
	apply := false
	for _, arg := range os.Args[1:] {
		if arg == "--apply" {
			apply = true
		}
	}

	// Load golfwithaccess courses
	raw, err := os.ReadFile("platforms/data/golfwithaccess.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading golfwithaccess.json: %v\n", err)
		os.Exit(1)
	}
	var courses []GWACourse
	if err := json.Unmarshal(raw, &courses); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing golfwithaccess.json: %v\n", err)
		os.Exit(1)
	}

	log("=== GolfWithAccess Migration ===")
	log("Courses to migrate: %d", len(courses))
	log("Mode: %s", map[bool]string{true: "APPLY", false: "DRY RUN"}[apply])
	log("")

	var results []MigrationResult
	gnCache := map[string][]golfNowFacility{}
	tokenCache := map[string][2]any{}
	var teeitupCount, golfnowCount, unresolvedCount int

	for i, course := range courses {
		log("[%d/%d] %s (%s, %s)", i+1, len(courses), course.DisplayName, course.City, course.State)

		// Try TeeItUp first
		tiu := tryTeeItUp(course)
		if tiu != nil {
			results = append(results, MigrationResult{
				Course:      course,
				Target:      "teeitup",
				TeeItUpData: tiu,
			})
			teeitupCount++
			log("  → TeeItUp (alias=%s, FID=%d, teeTimes=%d)", tiu.Alias, tiu.FacilityID, tiu.TeeTimes)
			log("")
			continue
		}

		log("  TeeItUp: MISS")

		// Try GolfNow
		gn := tryGolfNow(course, gnCache, tokenCache)
		if gn != nil {
			results = append(results, MigrationResult{
				Course:      course,
				Target:      "golfnow",
				GolfNowData: gn,
			})
			golfnowCount++
			log("  → GolfNow (facilityId=%d)", gn.FacilityID)
			log("")
			continue
		}

		log("  GolfNow: MISS")
		log("  → UNRESOLVED")
		results = append(results, MigrationResult{
			Course: course,
			Target: "unresolved",
		})
		unresolvedCount++
		log("")
	}

	// Summary
	log("========================================")
	log("=== MIGRATION SUMMARY")
	log("========================================")
	log("Total:      %d", len(courses))
	log("→ TeeItUp:  %d", teeitupCount)
	log("→ GolfNow:  %d", golfnowCount)
	log("→ Unresolved: %d", unresolvedCount)
	log("")

	if teeitupCount > 0 {
		log("=== TEEITUP ASSIGNMENTS ===")
		for _, r := range results {
			if r.Target == "teeitup" {
				log("  %-45s alias=%-35s FID=%d", r.Course.DisplayName, r.TeeItUpData.Alias, r.TeeItUpData.FacilityID)
			}
		}
		log("")
	}

	if golfnowCount > 0 {
		log("=== GOLFNOW ASSIGNMENTS ===")
		for _, r := range results {
			if r.Target == "golfnow" {
				log("  %-45s facilityId=%d", r.Course.DisplayName, r.GolfNowData.FacilityID)
			}
		}
		log("")
	}

	if unresolvedCount > 0 {
		log("=== UNRESOLVED (need manual assignment) ===")
		for _, r := range results {
			if r.Target == "unresolved" {
				log("  %-45s (%s, %s)", r.Course.DisplayName, r.Course.City, r.Course.State)
			}
		}
		log("")
	}

	// Save results JSON regardless
	resultsOut := map[string]any{
		"timestamp":  time.Now().Format(time.RFC3339),
		"total":      len(courses),
		"teeitup":    teeitupCount,
		"golfnow":    golfnowCount,
		"unresolved": unresolvedCount,
		"results":    results,
	}
	resultsJSON, _ := json.MarshalIndent(resultsOut, "", "  ")
	os.MkdirAll("discovery/results", 0755)
	resultsFile := fmt.Sprintf("discovery/results/gwa-migration-%s.json", time.Now().Format("2006-01-02-150405"))
	os.WriteFile(resultsFile, resultsJSON, 0644)
	log("Results saved to %s", resultsFile)

	if apply {
		log("")
		log("Applying migration...")
		if err := applyMigration(results); err != nil {
			log("ERROR: %v", err)
			os.Exit(1)
		}
		log("✅ Migration applied:")
		log("  - platforms/data/teeitup.json updated (+%d courses)", teeitupCount)
		log("  - platforms/data/golfnow.json updated (+%d courses)", golfnowCount)
		if unresolvedCount > 0 {
			log("  - platforms/data/golfwithaccess.json still has %d unresolved courses", unresolvedCount)
		} else {
			log("  - platforms/data/golfwithaccess.json is now empty")
		}
	} else {
		log("")
		log("Dry run complete. Run with --apply to update JSON files.")
	}
}
