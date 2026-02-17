package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type GolfWithAccessCourseConfig struct {
	Key         string   `json:"key"`
	Metro       string   `json:"metro"`
	CourseIDs   []string `json:"courseIds"`
	Slug        string   `json:"slug"`
	BookingURL  string   `json:"bookingUrl"`
	DisplayName string   `json:"displayName"`
	City        string   `json:"city"`
	State       string   `json:"state"`
}

var GolfWithAccessCourses = map[string]GolfWithAccessCourseConfig{}

type GolfWithAccessResponse struct {
	TeeTimes []GolfWithAccessTeeTime `json:"teeTimes"`
}

type GolfWithAccessTeeTime struct {
	DayTime     GolfWithAccessDayTime `json:"dayTime"`
	Players     GolfWithAccessPlayers `json:"players"`
	HolesOption string                `json:"holesOption"`
	DisplayRate GolfWithAccessRate    `json:"displayRate"`
}

type GolfWithAccessDayTime struct {
	Year   int `json:"year"`
	Month  int `json:"month"`
	Day    int `json:"day"`
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type GolfWithAccessPlayers struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type GolfWithAccessRate struct {
	Price    GolfWithAccessPrice `json:"price"`
	RateType string              `json:"rateType"`
}

type GolfWithAccessPrice struct {
	Dollars GolfWithAccessDollars `json:"dollars"`
}

type GolfWithAccessDollars struct {
	Value string `json:"value"`
	Cents int    `json:"cents"`
}

func FetchGolfWithAccess(config GolfWithAccessCourseConfig, date string) ([]DisplayTeeTime, error) {
	var courseParams []string
	for _, id := range config.CourseIDs {
		courseParams = append(courseParams, "courseIds="+id)
	}
	var url string = fmt.Sprintf(
		"https://golfwithaccess.com/api/v1/tee-times?%s&players=1&startAt=00%%3A00%%3A00&endAt=23%%3A59%%3A59&day=%s",
		strings.Join(courseParams, "&"), date,
	)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
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

	var data GolfWithAccessResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.TeeTimes {
		// Format time
		var hour int = tt.DayTime.Hour
		var period string = "AM"
		if hour >= 12 {
			period = "PM"
			if hour > 12 {
				hour = hour - 12
			}
		}
		if hour == 0 {
			hour = 12
		}
		var timeStr string = fmt.Sprintf("%d:%02d %s", hour, tt.DayTime.Minute, period)

		// Holes
		var holes string = "18"
		if tt.HolesOption == "NINE" {
			holes = "9"
		}

		// Price from public rate
		var price float64 = 0
		if tt.DisplayRate.Price.Dollars.Value != "" {
			price, _ = strconv.ParseFloat(tt.DisplayRate.Price.Dollars.Value, 64)
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   tt.Players.Max,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
