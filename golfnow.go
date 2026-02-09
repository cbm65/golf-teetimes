package main

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
	FacilityID  int
	SearchURL   string
	BookingURL  string
	DisplayName string
	City        string
	State       string
}

var GolfNowCourses = map[string]GolfNowCourseConfig{
	"murphycreek": {
		FacilityID:  17879,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17879-murphy-creek-golf-course/search",
		BookingURL:  "https://www.auroragov.org/things_to_do/golf/book_a_tee_time",
		DisplayName: "Murphy Creek",
		City: "Aurora", State: "CO",
	},
	"springhill": {
		FacilityID:  17876,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17876-springhill-golf-course/search",
		BookingURL:  "https://www.auroragov.org/things_to_do/golf/book_a_tee_time",
		DisplayName: "Springhill",
		City: "Aurora", State: "CO",
	},
	"meadowhills": {
		FacilityID:  17880,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17880-meadow-hills-golf-course/search",
		BookingURL:  "https://www.auroragov.org/things_to_do/golf/book_a_tee_time",
		DisplayName: "Meadow Hills",
		City: "Aurora", State: "CO",
	},
	"aurorahills": {
		FacilityID:  17878,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17878-aurora-hills-golf-course/search",
		BookingURL:  "https://www.auroragov.org/things_to_do/golf/book_a_tee_time",
		DisplayName: "Aurora Hills",
		City: "Aurora", State: "CO",
	},
	"saddlerock": {
		FacilityID:  17877,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17877-saddle-rock-golf-course/search",
		BookingURL:  "https://www.auroragov.org/things_to_do/golf/book_a_tee_time",
		DisplayName: "Saddle Rock",
		City: "Aurora", State: "CO",
	},
	"raccooncreek": {
		FacilityID:  515,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/515-raccoon-creek-golf-course/search",
		BookingURL:  "https://raccooncreek.ezlinksgolf.com/index.html#/search",
		DisplayName: "Raccoon Creek",
		City: "Littleton", State: "CO",
	},
	"arrowhead": {
		FacilityID:  453,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/453-arrowhead-golf-club/search",
		BookingURL:  "https://arrowheadpp.ezlinksgolf.com/index.html#/search",
		DisplayName: "Arrowhead",
		City: "Littleton", State: "CO",
	},
	"tpcscottsdale": {
		FacilityID:  7076,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/7076-tpc-scottsdale-champions-course/search",
		BookingURL:  "https://tpcscottsdale.ezlinksgolf.com/index.html?utm_source=google&utm_medium=organic#/preSearch",
		DisplayName: "TPC Scottsdale Champions",
		City: "Scottsdale", State: "AZ",
	},
	"tpcscottsdalestadium": {
		FacilityID:  3482,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3482-the-stadium-course-at-tpc-scottsdale/search",
		BookingURL:  "https://tpcscottsdale.ezlinksgolf.com/index.html?utm_source=google&utm_medium=organic#/preSearch",
		DisplayName: "TPC Scottsdale Stadium",
		City: "Scottsdale", State: "AZ",
	},
	"ravengolfclub": {
		FacilityID:  1446,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1446-raven-golf-club/search",
		BookingURL:  "https://ravenpp.ezlinksgolf.com/index.html#/search?_ga=2.47721895.1848503108.1640768769-232493065.1640768769",
		DisplayName: "Raven Golf Club",
		City: "Phoenix", State: "AZ",
	},
	"stonecreek": {
		FacilityID:  122,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/122-stonecreek-golf-club/search",
		BookingURL:  "https://stonecreekpp.ezlinksgolf.com/index.html#/search?_ga=2.136584944.1737062094.1641355590-2105607762.1641355590",
		DisplayName: "Stonecreek Golf Club",
		City: "Phoenix", State: "AZ",
	},
	"verrado": {
		FacilityID:  14378,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/14378-verrado-golf-club-victory-course/search",
		BookingURL:  "https://verrado.ezlinksgolf.com/index.html#/search",
		DisplayName: "Verrado Victory",
		City: "Buckeye", State: "AZ",
	},
	"verradofounders": {
		FacilityID:  1707,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1707-verrado-golf-club-founders-course/search",
		BookingURL:  "https://verrado.ezlinksgolf.com/index.html#/search",
		DisplayName: "Verrado Founders",
		City: "Buckeye", State: "AZ",
	},
	"quintero": {
		FacilityID:  6388,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6388-quintero-golf-club/search",
		BookingURL:  "https://quintero2.ezlinksgolf.com/index.html#/search",
		DisplayName: "Quintero Golf Club",
		City: "Peoria", State: "AZ",
	},
	"longbow": {
		FacilityID:  3021,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3021-longbow-golf-club/search",
		BookingURL:  "https://longbow.ezlinksgolf.com/index.html#/search",
		DisplayName: "Longbow Golf Club",
		City: "Mesa", State: "AZ",
	},
	"superstitionsprings": {
		FacilityID:  120,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/120-superstition-springs-golf-club/search",
		BookingURL:  "https://superstitionspringspp.ezlinksgolf.com/index.html#/search?_ga=2.260740365.183174987.1641547646-232493065.1640768769",
		DisplayName: "Superstition Springs",
		City: "Mesa", State: "AZ",
	},
	"ocotillo": {
		FacilityID:  253,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/253-ocotillo-golf-club/search",
		BookingURL:  "https://ocotillo.ezlinksgolf.com/index.html#/search",
		DisplayName: "Ocotillo Golf Club",
		City: "Chandler", State: "AZ",
	},
	"dovevalleyranch": {
		FacilityID:  115,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/115-dove-valley-ranch-golf-club/search",
		BookingURL:  "https://dovevalley.ezlinksgolf.com/index.html#/search",
		DisplayName: "Dove Valley Ranch",
		City: "Cave Creek", State: "AZ",
	},
	"mccormickranchpine": {
		FacilityID:  7078,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/7078-mccormick-ranch-golf-club-pine-course/search",
		BookingURL:  "https://mccormickranch.ezlinksgolf.com/index.html#/search",
		DisplayName: "McCormick Ranch Pine",
		City: "Scottsdale", State: "AZ",
	},
	"mccormickranchpalm": {
		FacilityID:  1356,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1356-mccormick-ranch-golf-club-palm-course/search",
		BookingURL:  "https://mccormickranch.ezlinksgolf.com/index.html#/search",
		DisplayName: "McCormick Ranch Palm",
		City: "Scottsdale", State: "AZ",
	},
	"talkingstickoodham": {
		FacilityID:  12968,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/12968-talking-stick-golf-club-oodham-north/search",
		BookingURL:  "https://talkingstickdailyfee.ezlinksgolf.com/index.html#/search",
		DisplayName: "Talking Stick O'odham",
		City: "Scottsdale", State: "AZ",
	},
	"talkingstickpiipaash": {
		FacilityID:  814,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/814-talking-stick-golf-club-piipaash-south/search",
		BookingURL:  "https://talkingstickdailyfee.ezlinksgolf.com/index.html#/search",
		DisplayName: "Talking Stick Piipaash",
		City: "Scottsdale", State: "AZ",
	},
	"whirlwinddevilsclaw": {
		FacilityID:  110,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/110-whirlwind-golf-club-devils-claw/search",
		BookingURL:  "https://whirlwindbest.ezlinksgolf.com/index.html#/search",
		DisplayName: "Whirlwind Devil's Claw",
		City: "Chandler", State: "AZ",
	},
	"whirlwindcattail": {
		FacilityID:  13192,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/13192-whirlwind-golf-club-cattail/search",
		BookingURL:  "https://whirlwindbest.ezlinksgolf.com/index.html#/search",
		DisplayName: "Whirlwind Cattail",
		City: "Chandler", State: "AZ",
	},
	"westernskies": {
		FacilityID:  123,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/123-western-skies-golf-club/search",
		BookingURL:  "https://westernskies.ezlinksgolf.com/index.html#/search",
		DisplayName: "Western Skies Golf Club",
		City: "Gilbert", State: "AZ",
	},
	"kokopelli": {
		FacilityID:  121,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/121-kokopelli-golf-club/search",
		BookingURL:  "https://kokopellipp.ezlinksgolf.com/index.html#/search",
		DisplayName: "Kokopelli Golf Club",
		City: "Gilbert", State: "AZ",
	},
}

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
	var client http.Client
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

func fetchGolfNow(config GolfNowCourseConfig, date string) ([]DisplayTeeTime, error) {
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

	var client http.Client
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
