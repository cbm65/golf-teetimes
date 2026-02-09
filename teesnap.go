package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type TeeSnapCourseConfig struct {
	Subdomain   string
	CourseID    string
	BookingURL  string
	DisplayName string
	City        string
	State       string
}

var TeeSnapCourses = map[string]TeeSnapCourseConfig{
	"sundance": {
		Subdomain:   "sundancegolfclub",
		CourseID:    "1801",
		BookingURL:  "https://sundancegolfclub.teesnap.net",
		DisplayName: "Sundance Golf Club",
		City:        "Buckeye",
		State:       "AZ",
	},
}

type TeeSnapResponse struct {
	TeeTimes TeeSnapTeeTimesOuter `json:"teeTimes"`
}

type TeeSnapTeeTimesOuter struct {
	TeeTimes []TeeSnapTeeTime `json:"teeTimes"`
}

type TeeSnapTeeTime struct {
	Prices         []TeeSnapPrice      `json:"prices"`
	TeeOffSections []TeeSnapTeeOff     `json:"teeOffSections"`
	TeeTime        string              `json:"teeTime"`
}

type TeeSnapPrice struct {
	RoundType string `json:"roundType"`
	Price     string `json:"price"`
}

type TeeSnapTeeOff struct {
	Bookings []int `json:"bookings"`
}

func fetchTeeSnap(config TeeSnapCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://%s.teesnap.net/customer-api/teetimes-day?course=%s&date=%s&players=1&holes=18&addons=off",
		config.Subdomain, config.CourseID, date,
	)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://"+config.Subdomain+".teesnap.net/")

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

	var data TeeSnapResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.TeeTimes.TeeTimes {
		// Count booked slots across all tee-off sections
		var booked int = 0
		for _, sec := range tt.TeeOffSections {
			booked += len(sec.Bookings)
		}
		var openings int = 4 - booked
		if openings <= 0 {
			continue
		}

		// Parse time "2026-02-09T07:14:00"
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05", tt.TeeTime)
		if err != nil {
			continue
		}
		var timeStr string = t.Format("3:04 PM")

		// Get 18-hole price, fall back to first
		var price float64 = 0
		for _, p := range tt.Prices {
			if p.RoundType == "EIGHTEEN_HOLE" {
				price, _ = strconv.ParseFloat(p.Price, 64)
				break
			}
		}
		if price == 0 && len(tt.Prices) > 0 {
			price, _ = strconv.ParseFloat(tt.Prices[0].Price, 64)
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      "18",
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
