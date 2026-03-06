package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LetsGoGolfCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	Slug        string `json:"slug"`
	ProgramID   int    `json:"programId"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var LetsGoGolfCourses = map[string]LetsGoGolfCourseConfig{}

type letsGoGolfTeeTimeGroup struct {
	TeeOffAtLocal string    `json:"tee_off_at_local"`
	StartingRate  float64   `json:"starting_rate"`
	Players       []int     `json:"players"`
	AmenityCodes  []string  `json:"amenity_codes"`
}

type letsGoGolfResponse struct {
	TeeTimeGroups []letsGoGolfTeeTimeGroup `json:"tee_time_groups"`
}

func FetchLetsGoGolf(config LetsGoGolfCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://api.letsgo.golf/api/courses/reservations_group?date=%s&slug=%s&programId=%d&allCartSelected=true&allRatesSelected=true&min_hour=5&max_hour=21&min_price=0&max_price=500",
		date, config.Slug, config.ProgramID,
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

	var data letsGoGolfResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, g := range data.TeeTimeGroups {
		// Parse time "2026-02-18T13:48:00.000Z"
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05.000Z", g.TeeOffAtLocal)
		if err != nil {
			continue
		}

		// Openings = max player count in the players array
		var openings int
		for _, p := range g.Players {
			if p > openings {
				openings = p
			}
		}
		if openings <= 0 {
			continue
		}

		// Holes from amenity codes
		var holes string = "18"
		for _, code := range g.AmenityCodes {
			if code == "is_9_holes" {
				holes = "9"
				break
			}
		}

		results = append(results, DisplayTeeTime{
			Time:       t.Format("3:04 PM"),
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holes,
			Price:      g.StartingRate,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
