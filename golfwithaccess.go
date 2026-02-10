package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type GolfWithAccessCourseConfig struct {
	CourseIDs   []string
	Slug        string
	BookingURL  string
	DisplayName string
	City        string
	State       string
}

var GolfWithAccessCourses = map[string]GolfWithAccessCourseConfig{
	"lookoutmountain": {
		CourseIDs:   []string{"fd506bf4-ae6a-4a92-ae3f-7f847f098fb2"},
		Slug:        "lookout-mountain-golf-club",
		BookingURL:  "https://golfwithaccess.com/course/lookout-mountain-golf-club/reserve-tee-time",
		DisplayName: "Lookout Mountain Golf Club",
		City:        "Phoenix",
		State:       "AZ",
	},
	"akchinsoutherndunes": {
		CourseIDs:   []string{"2598416d-4e75-44d3-bbe7-811969e14a95"},
		Slug:        "ak-chin-southern-dunes-golf-club",
		BookingURL:  "https://golfwithaccess.com/course/ak-chin-southern-dunes-golf-club/reserve-tee-time",
		DisplayName: "Ak-Chin Southern Dunes",
		City:        "Maricopa",
		State:       "AZ",
	},
	"troonnorthpinnacle": {
		CourseIDs:   []string{"4bf6e82f-697f-46d1-8fad-2de5a6083477"},
		Slug:        "troon-north-golf-club-pinnacle-course",
		BookingURL:  "https://golfwithaccess.com/course/troon-north-golf-club/reserve-tee-time",
		DisplayName: "Troon North Pinnacle",
		City:        "Scottsdale",
		State:       "AZ",
	},
	"troonnorthmonument": {
		CourseIDs:   []string{"f800515d-41dd-4ae7-a853-57e8092284aa"},
		Slug:        "troon-north-golf-club-monument-course",
		BookingURL:  "https://golfwithaccess.com/course/troon-north-golf-club/reserve-tee-time",
		DisplayName: "Troon North Monument",
		City:        "Scottsdale",
		State:       "AZ",
	},
	"kierland": {
		CourseIDs:   []string{"ab5ad653-b217-4119-bcb7-80dd0aecffaa", "6a9798ab-617e-43aa-9ab3-28229a6a83cf", "b0e5b8a6-fe88-4cb8-8500-56cbba88e531"},
		Slug:        "the-westin-kierland-golf-club",
		BookingURL:  "https://golfwithaccess.com/course/the-westin-kierland-golf-club/reserve-tee-time",
		DisplayName: "Kierland Golf Club",
		City:        "Scottsdale",
		State:       "AZ",
	},
	"eaglemountain": {
		CourseIDs:   []string{"698b68b9-908e-416e-8368-a043e2a90072"},
		Slug:        "eagle-mountain-golf-club",
		BookingURL:  "https://golfwithaccess.com/course/eagle-mountain-golf-club/reserve-tee-time",
		DisplayName: "Eagle Mountain Golf Club",
		City:        "Fountain Hills",
		State:       "AZ",
	},
	"lascolinas": {
		CourseIDs:   []string{"b6a25387-b6b4-42a5-a118-58664df8836a"},
		Slug:        "las-colinas-golf-club",
		BookingURL:  "https://golfwithaccess.com/course/las-colinas-golf-club/reserve-tee-time",
		DisplayName: "Las Colinas Golf Club",
		City:        "Queen Creek",
		State:       "AZ",
	},
	"phoenician": {
		CourseIDs:   []string{"b40cfca0-6caf-422e-8459-ebaf9da89c3e"},
		Slug:        "the-phoenician",
		BookingURL:  "https://golfwithaccess.com/course/the-phoenician/reserve-tee-time",
		DisplayName: "The Phoenician Golf Club",
		City:        "Scottsdale",
		State:       "AZ",
	},
	"estrella": {
		CourseIDs:   []string{"5e7260b2-d5b3-45ba-9030-b0d1d3dcdf17"},
		Slug:        "golf-club-of-estrella",
		BookingURL:  "https://golfwithaccess.com/course/golf-club-of-estrella/reserve-tee-time",
		DisplayName: "Golf Club of Estrella",
		City:        "Goodyear",
		State:       "AZ",
	},
	"mountainfalls": {
		CourseIDs:   []string{"b2ff3f68-2ac9-41b1-8924-e79a9040c4a7"},
		Slug:        "mountain-falls-golf-club",
		BookingURL:  "https://golfwithaccess.com/course/mountain-falls-golf-club/reserve-tee-time",
		DisplayName: "Mountain Falls",
		City:        "Pahrump",
		State:       "NV",
	},
}

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

func fetchGolfWithAccess(config GolfWithAccessCourseConfig, date string) ([]DisplayTeeTime, error) {
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
