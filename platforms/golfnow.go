package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

type GolfNowCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	FacilityID  int    `json:"facilityId"`
	SearchURL   string `json:"searchUrl"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var GolfNowCourses = map[string]GolfNowCourseConfig{}

type GolfNowSearchRequest struct {
	Radius                     int     `json:"Radius"`
	Latitude                   float64 `json:"Latitude"`
	Longitude                  float64 `json:"Longitude"`
	PageSize                   int     `json:"PageSize"`
	PageNumber                 int     `json:"PageNumber"`
	SearchType                 int     `json:"SearchType"`
	SortBy                     string  `json:"SortBy"`
	SortDirection              int     `json:"SortDirection"`
	Date                       string  `json:"Date"`
	BestDealsOnly              bool    `json:"BestDealsOnly"`
	PriceMin                   string  `json:"PriceMin"`
	PriceMax                   string  `json:"PriceMax"`
	Players                    string  `json:"Players"`
	TimePeriod                 string  `json:"TimePeriod"`
	Holes                      string  `json:"Holes"`
	FacilityType               int     `json:"FacilityType"`
	RateType                   string  `json:"RateType"`
	TimeMin                    string  `json:"TimeMin"`
	TimeMax                    string  `json:"TimeMax"`
	FacilityId                 int     `json:"FacilityId"`
	SortByRollup               string  `json:"SortByRollup"`
	View                       string  `json:"View"`
	ExcludeFeaturedFacilities  bool    `json:"ExcludeFeaturedFacilities"`
	TeeTimeCount               int     `json:"TeeTimeCount"`
	PromotedCampaignsOnly      bool    `json:"PromotedCampaignsOnly"`
}

type GolfNowResponse struct {
	TTResults GolfNowResults `json:"ttResults"`
	Total     int            `json:"total"`
}

type GolfNowResults struct {
	TeeTimes []GolfNowTeeTime `json:"teeTimes"`
}

type GolfNowTeeTime struct {
	Time                  string             `json:"time"`
	FormattedTime         string             `json:"formattedTime"`
	FormattedTimeMeridian string             `json:"formattedTimeMeridian"`
	DisplayRate           float64            `json:"displayRate"`
	MultipleHolesRate     json.Number        `json:"multipleHolesRate"`
	PlayerRule            int                `json:"playerRule"`
	Facility              GolfNowFacility    `json:"facility"`
	TeeTimeRates          []GolfNowRate      `json:"teeTimeRates"`
}

type GolfNowFacility struct {
	FacilityId int    `json:"facilityId"`
	Name       string `json:"name"`
}

type GolfNowRate struct {
	HoleCount int    `json:"holeCount"`
	RateName  string `json:"rateName"`
}

func formatGolfNowDate(date string) string {
	var t time.Time
	var err error
	t, err = time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Jan 02 2006")
}

func getVerificationToken(facilityURL string) (string, error) {
	var client http.Client = http.Client{Timeout: PlatformTimeout}
	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", facilityURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var html string = string(body)

	// Look for the token in a meta tag or hidden input
	var re *regexp.Regexp = regexp.MustCompile(`__RequestVerificationToken[^>]*value="([^"]+)"`)
	var matches []string = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Also try meta tag format
	re = regexp.MustCompile(`name="__RequestVerificationToken"[^>]*content="([^"]+)"`)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Try data attribute format
	re = regexp.MustCompile(`data-request-verification-token="([^"]+)"`)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("verification token not found")
}

func FetchGolfNow(config GolfNowCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Step 1: Get verification token
	var token string
	var err error
	token, err = getVerificationToken(config.SearchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	// Step 2: Search for tee times
	var searchDate string = formatGolfNowDate(date)
	var reqBody GolfNowSearchRequest = GolfNowSearchRequest{
		Radius:                    35,
		Latitude:                  39.6855,
		Longitude:                 -104.7076,
		PageSize:                  50,
		PageNumber:                0,
		SearchType:                1,
		SortBy:                    "Date",
		SortDirection:             0,
		Date:                      searchDate,
		BestDealsOnly:             false,
		PriceMin:                  "0",
		PriceMax:                  "10000",
		Players:                   "0",
		TimePeriod:                "3",
		Holes:                     "3",
		FacilityType:              0,
		RateType:                  "all",
		TimeMin:                   "0",
		TimeMax:                   "48",
		FacilityId:                config.FacilityID,
		SortByRollup:              "Date.MinDate",
		View:                      "Grouping",
		ExcludeFeaturedFacilities: true,
		TeeTimeCount:              50,
		PromotedCampaignsOnly:     false,
	}

	var jsonData []byte
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", "https://www.golfnow.com/api/tee-times/tee-time-results", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://www.golfnow.com")
	req.Header.Set("Referer", config.SearchURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("__requestverificationtoken", token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

	var client http.Client = http.Client{Timeout: PlatformTimeout}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for HTML response (Cloudflare block)
	if len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("blocked by bot protection")
	}

	var data GolfNowResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.TTResults.TeeTimes {
		var timeStr string = tt.FormattedTime + " " + tt.FormattedTimeMeridian

		var holes string = tt.MultipleHolesRate.String()

		// playerRule is a bitmask: bit 0 = 1 player, bit 1 = 2 players, etc.
		// Highest set bit + 1 = max openings
		var openings int = 0
		var rule int = tt.PlayerRule
		for rule > 0 {
			openings++
			rule = rule >> 1
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holes,
			Price:      tt.DisplayRate,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
