package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TeeItUpCourseConfig struct {
	Alias       string
	FacilityID  string
	BookingURL  string
	DisplayName string
}

var TeeItUpCourses = map[string]TeeItUpCourseConfig{
	"hylandhills": {
		Alias:       "hyland-hills-park-recreation-district",
		FacilityID:  "9201",
		BookingURL:  "https://hyland-hills-park-recreation-district.book.teeitup.com/teetimes",
		DisplayName: "Hyland Hills",
	},
	"stoneycreek": {
		Alias:       "stoney-creek-golf-course",
		FacilityID:  "13099",
		BookingURL:  "https://stoney-creek-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Stoney Creek",
	},
	"commonground": {
		Alias:       "commonground-golf-course",
		FacilityID:  "5275",
		BookingURL:  "https://commonground-golf-course.book.teeitup.com/teetimes",
		DisplayName: "CommonGround",
	},
	"buffalorun": {
		Alias:       "buffalo-run-golf-course",
		FacilityID:  "513",
		BookingURL:  "https://buffalo-run-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Buffalo Run",
	},
}

type TeeItUpResponse struct {
	Teetimes []TeeItUpTeeTime `json:"teetimes"`
}

type TeeItUpTeeTime struct {
	Teetime       string          `json:"teetime"`
	MaxPlayers    int             `json:"maxPlayers"`
	BookedPlayers int             `json:"bookedPlayers"`
	Rates         []TeeItUpRate   `json:"rates"`
}

type TeeItUpRate struct {
	Name           string  `json:"name"`
	Holes          int     `json:"holes"`
	GreenFeeCart   float64 `json:"greenFeeCart"`
	GreenFeeWalking float64 `json:"greenFeeWalking"`
	AllowedPlayers []int   `json:"allowedPlayers"`
}

var denverLoc *time.Location

func init() {
	var err error
	denverLoc, err = time.LoadLocation("America/Denver")
	if err != nil {
		denverLoc = time.FixedZone("MST", -7*60*60)
	}
}

func fetchTeeItUp(config TeeItUpCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://phx-api-be-east-1b.kenna.io/v2/tee-times?date=%s&facilityIds=%s",
		date, config.FacilityID,
	)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-be-alias", config.Alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+config.Alias+".book.teeitup.com")

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

	var data []TeeItUpResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	var results []DisplayTeeTime
	for _, tt := range data[0].Teetimes {
		// Parse UTC time and convert to Denver time
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05.000Z", tt.Teetime)
		if err != nil {
			continue
		}
		t = t.In(denverLoc)
		var timeStr string = t.Format("3:04 PM")

		var openings int = tt.MaxPlayers

		// Get price and holes from first rate
		var price float64 = 0
		var holes int = 18
		if len(tt.Rates) > 0 {
			var rate TeeItUpRate = tt.Rates[0]
			holes = rate.Holes
			// Price is in cents
			if rate.GreenFeeCart > 0 {
				price = rate.GreenFeeCart / 100
			} else if rate.GreenFeeWalking > 0 {
				price = rate.GreenFeeWalking / 100
			}
		}

		var holesStr string = fmt.Sprintf("%d", holes)

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			Openings:   openings,
			Holes:      holesStr,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
