package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type TenForeCourseConfig struct {
	Key          string `json:"key"`
	Metro        string `json:"metro"`
	GolfCourseID string `json:"golfCourseId"`
	BookingURL   string `json:"bookingUrl"`
	DisplayName  string `json:"displayName"`
	City         string `json:"city"`
	State        string `json:"state"`
}

var TenForeCourses = map[string]TenForeCourseConfig{}

type tenForeTeeTime struct {
	Time       string            `json:"time"`
	Spots      int               `json:"spots"`
	FeePrice18 float64           `json:"feePrice18"`
	FeePrice9  float64           `json:"feePrice9"`
	Back9      bool              `json:"back9"`
	Customers  []json.RawMessage `json:"customers"`
}

func FetchTenFore(config TenForeCourseConfig, date string) ([]DisplayTeeTime, error) {
	var client http.Client = http.Client{Timeout: PlatformTimeout}

	var courseID int
	var err error
	courseID, err = strconv.Atoi(config.GolfCourseID)
	if err != nil {
		return nil, fmt.Errorf("tenfore: invalid golfCourseId %q", config.GolfCourseID)
	}

	var postBody []byte
	postBody, err = json.Marshal(map[string]any{
		"golfCourseId": courseID,
		"subCourseId":  nil,
		"dateFrom":     date,
		"appId":        23,
	})
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", "https://swan.tenfore.golf/api/BookingEngineV4/booking-times", bytes.NewReader(postBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

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

	var apiResp struct {
		Successful bool             `json:"successful"`
		Data       []tenForeTeeTime `json:"data"`
	}
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, err
	}
	if !apiResp.Successful {
		return nil, fmt.Errorf("tenfore: API returned unsuccessful for %s on %s", config.GolfCourseID, date)
	}

	var results []DisplayTeeTime
	for _, tt := range apiResp.Data {
		// spots=0: fully booked — skip
		// spots>0, customers present: partially booked — customers length is open slots
		// spots>0, customers empty: fully open — spots is available count
		if tt.Spots <= 0 {
			continue
		}
		var openings int
		if len(tt.Customers) > 0 {
			openings = len(tt.Customers)
		} else {
			openings = tt.Spots
		}

		var t time.Time
		t, err = time.Parse("15:04:05", tt.Time)
		if err != nil {
			continue
		}

		var holes string = "18"
		var price float64 = tt.FeePrice18
		if tt.Back9 {
			holes = "9"
			price = tt.FeePrice9
		}

		results = append(results, DisplayTeeTime{
			Time:       t.Format("3:04 PM"),
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
