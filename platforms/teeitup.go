package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TeeItUpCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	Alias       string `json:"alias"`
	FacilityID  string `json:"facilityId"` // optional — omit from URL if empty, API returns all courses for alias
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var TeeItUpCourses = map[string]TeeItUpCourseConfig{}

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

func TeeItUpTimezone(state string) *time.Location {
	var tz string
	switch state {
	case "AZ":
		tz = "America/Phoenix"
	case "GA":
		tz = "America/New_York"
	case "NV":
		tz = "America/Los_Angeles"
	default:
		tz = "America/Denver"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

func FetchTeeItUp(config TeeItUpCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string
	if config.FacilityID != "" {
		url = fmt.Sprintf(
			"https://phx-api-be-east-1b.kenna.io/v2/tee-times?date=%s&facilityIds=%s&dateMax=%s",
			date, config.FacilityID, date,
		)
	} else {
		url = fmt.Sprintf(
			"https://phx-api-be-east-1b.kenna.io/v2/tee-times?date=%s&dateMax=%s",
			date, date,
		)
	}

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
		// API sometimes returns an error object instead of array — treat as no results
		return nil, nil
	}

	if len(data) == 0 {
		return nil, nil
	}

	var loc *time.Location = TeeItUpTimezone(config.State)
	var bookingURL string = "https://" + config.Alias + ".book.teeitup.com/teetimes"

	var results []DisplayTeeTime
	for _, tt := range data[0].Teetimes {
		// Parse UTC time and convert to local time
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05.000Z", tt.Teetime)
		if err != nil {
			continue
		}
		t = t.In(loc)
		var timeStr string = t.Format("3:04 PM")

		var openings int = tt.MaxPlayers

		// Get price and holes from cheapest rate
		var price float64 = 0
		var holes int = 18
		if len(tt.Rates) > 0 {
			holes = tt.Rates[0].Holes
			var bestPrice float64 = 0
			for _, rate := range tt.Rates {
				var ratePrice float64 = 0
				if rate.GreenFeeCart > 0 {
					ratePrice = rate.GreenFeeCart / 100
				} else if rate.GreenFeeWalking > 0 {
					ratePrice = rate.GreenFeeWalking / 100
				}
				if ratePrice > 0 && (bestPrice == 0 || ratePrice < bestPrice) {
					bestPrice = ratePrice
				}
				if rate.Holes > 0 {
					holes = rate.Holes
				}
			}
			price = bestPrice
		}

		var holesStr string = fmt.Sprintf("%d", holes)

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holesStr,
			Price:      price,
			BookingURL: bookingURL,
		})
	}

	return results, nil
}
