//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Metro center coordinates and search radius
var metroCoords = map[string][3]float64{
	// slug: {lat, lng, radius_miles}
	"denver":       {39.7392, -104.9903, 35},
	"phoenix":      {33.4484, -112.0740, 40},
	"lasvegas":     {36.1699, -115.1398, 25},
	"atlanta":      {33.7490, -84.3880, 35},
	"albuquerque":  {35.4000, -106.3000, 50},
	"dallas":       {32.7767, -96.7970, 40},
	"neworleans":   {29.9511, -90.0715, 30},
	"nashville":    {36.1627, -86.7816, 30},
	"miami":        {25.7617, -80.1918, 35},
	"sanfrancisco": {37.5585, -122.2711, 45},
	"oklahomacity": {35.4676, -97.5164, 35},
	"montgomery":   {32.3668, -86.3000, 35},
	"losangeles":   {34.0522, -118.2437, 35},
	"sandiego":     {32.7157, -117.1611, 40},
}

type searchRequest struct {
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

type searchResponse struct {
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

type facility struct {
	ID    int
	Name  string
	City  string
	State string
}

func getToken(lat, lng float64) (string, *cookiejar.Jar, error) {
	url := fmt.Sprintf("https://www.golfnow.com/tee-times/search#sortby=Date&view=Grouping&lat=%.4f&lng=%.4f", lat, lng)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("sec-ch-ua", `"Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "none")
	req.Header.Set("sec-fetch-user", "?1")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	fmt.Printf("  Token page status: %d, body: %d bytes\n", resp.StatusCode, len(body))

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

	if len(html) < 500 {
		fmt.Printf("  Body: %s\n", html)
	} else {
		fmt.Printf("  Body start: %s\n", html[:500])
	}

	return "", nil, fmt.Errorf("token not found in %d byte response", len(body))
}

func searchGolfNow(lat, lng, radius float64, token string, jar *cookiejar.Jar, page int, dateStr string) (*searchResponse, error) {
	clientDate := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	body := searchRequest{
		Radius:                    int(radius),
		Latitude:                  fmt.Sprintf("%.5f", lat),
		Longitude:                 fmt.Sprintf("%.5f", lng),
		PageSize:                  30,
		PageNumber:                page,
		SearchType:                0,
		SortBy:                    "Facilities.Distance",
		SortDirection:             "0",
		Date:                      dateStr,
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
		CurrentClientDate:         clientDate,
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
	req.Header.Set("sec-ch-ua", `"Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")

	client := &http.Client{Jar: jar}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	fmt.Printf("  Search response: status=%d, %d bytes\n", resp.StatusCode, len(respBody))

	if len(respBody) > 0 && respBody[0] == '<' {
		fmt.Printf("  Got HTML (blocked). First 200: %s\n", string(respBody[:min(200, len(respBody))]))
		return nil, fmt.Errorf("blocked by bot protection (got HTML)")
	}

	var data searchResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("JSON parse error: %v (first 200 bytes: %s)", err, string(respBody[:min(200, len(respBody))]))
	}
	if data.Total == 0 {
		fmt.Printf("  0 results. Response: %s\n", string(respBody[:min(300, len(respBody))]))
	}
	return &data, nil
}

func loadExistingCourses() map[string]bool {
	existing := map[string]bool{}

	dataDir := filepath.Join("platforms", "data")
	files, _ := filepath.Glob(filepath.Join(dataDir, "*.json"))
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var courses []map[string]interface{}
		if err := json.Unmarshal(raw, &courses); err != nil {
			continue
		}
		for _, c := range courses {
			if dn, ok := c["displayName"].(string); ok && dn != "" {
				existing[normalize(dn)] = true
			}
			if names, ok := c["names"].(map[string]interface{}); ok {
				for _, v := range names {
					if s, ok := v.(string); ok {
						existing[normalize(s)] = true
					}
				}
			}
		}
	}

	return existing
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " - ", " ")
	s = strings.ReplaceAll(s, "golf course", "")
	s = strings.ReplaceAll(s, "golf club", "")
	s = strings.ReplaceAll(s, "golf links", "")
	s = strings.ReplaceAll(s, "golf resort", "")
	s = strings.ReplaceAll(s, "golf center", "")
	s = strings.ReplaceAll(s, "golf ", "")
	s = strings.ReplaceAll(s, "the ", "")
	s = strings.ReplaceAll(s, "  ", " ")
	return strings.TrimSpace(s)
}

func fuzzyMatch(name string, existing map[string]bool) bool {
	n := normalize(name)
	if existing[n] {
		return true
	}
	// Check base name (before " - ") for multi-course facilities
	base := strings.ToLower(strings.TrimSpace(name))
	if idx := strings.Index(base, " - "); idx > 0 {
		if existing[normalize(base[:idx])] {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run discovery/golfnow-discover.go <metro>")
		fmt.Println("Available metros:", strings.Join(metroNames(), ", "))
		os.Exit(1)
	}

	metro := strings.ToLower(os.Args[1])
	coords, ok := metroCoords[metro]
	if !ok {
		fmt.Printf("Unknown metro: %s\n", metro)
		fmt.Println("Available:", strings.Join(metroNames(), ", "))
		os.Exit(1)
	}

	lat, lng, radius := coords[0], coords[1], coords[2]
	fmt.Printf("Searching GolfNow for %s (%.4f, %.4f, %d mi radius)...\n", metro, lat, lng, int(radius))

	// Get verification token
	fmt.Print("Getting verification token... ")
	token, jar, err := getToken(lat, lng)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")

	// Search across multiple dates for better coverage
	seen := map[int]facility{}
	for dayOffset := 1; dayOffset <= 7; dayOffset++ {
		searchDate := time.Now().Add(time.Duration(dayOffset) * 24 * time.Hour)
		dateStr := searchDate.Format("Jan 02 2006")
		page := 0
		for {
			fmt.Printf("Fetching %s page %d... ", dateStr, page)
			data, err := searchGolfNow(lat, lng, radius, token, jar, page, dateStr)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				break
			}
			fmt.Printf("%d facilities (total: %d)\n", len(data.TTResults.Facilities), data.Total)

			for _, f := range data.TTResults.Facilities {
				if f.IsSimulator {
					continue
				}
				if _, exists := seen[f.ID]; !exists {
					seen[f.ID] = facility{
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
			page++
			if page > 20 {
				break
			}
		}
	}

	fmt.Printf("\nFound %d unique GolfNow facilities\n", len(seen))

	// Load existing courses
	existing := loadExistingCourses()
	fmt.Printf("Loaded %d existing course names\n\n", len(existing))

	// Split into already-added and missing
	var missing []facility
	var matched []facility
	for _, f := range seen {
		if fuzzyMatch(f.Name, existing) {
			matched = append(matched, f)
		} else {
			missing = append(missing, f)
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(matched, func(i, j int) bool { return matched[i].Name < matched[j].Name })

	fmt.Printf("=== ALREADY ADDED (%d) ===\n", len(matched))
	for _, f := range matched {
		fmt.Printf("  ✓ %s (%s, %s) [facilityId=%d]\n", f.Name, f.City, f.State, f.ID)
	}

	fmt.Printf("\n=== MISSING - NOT YET ADDED (%d) ===\n", len(missing))
	for _, f := range missing {
		fmt.Printf("  ✗ %s (%s, %s) [facilityId=%d]\n", f.Name, f.City, f.State, f.ID)
	}

	// Also output as golfnow.json entries ready to paste
	if len(missing) > 0 {
		fmt.Printf("\n=== READY-TO-ADD JSON ENTRIES ===\n")
		var entries []map[string]interface{}
		for _, f := range missing {
			key := strings.ToLower(f.Name)
			key = strings.ReplaceAll(key, " ", "-")
			key = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(key, "")
			key = regexp.MustCompile(`-+`).ReplaceAllString(key, "-")
			key = strings.Trim(key, "-")

			entries = append(entries, map[string]interface{}{
				"key":         key,
				"metro":       metro,
				"facilityId":  f.ID,
				"searchUrl":   fmt.Sprintf("https://www.golfnow.com/tee-times/facility/%d/search", f.ID),
				"bookingUrl":  fmt.Sprintf("https://www.golfnow.com/tee-times/facility/%d/search", f.ID),
				"displayName": f.Name,
				"city":        f.City,
				"state":       f.State,
			})
		}
		out, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Println(string(out))
	}
}

func metroNames() []string {
	var names []string
	for k := range metroCoords {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
