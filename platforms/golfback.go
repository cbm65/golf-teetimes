package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GolfBackCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	CourseID    string `json:"courseId"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var GolfBackCourses = map[string]GolfBackCourseConfig{}

type golfBackResponse struct {
	Data []golfBackTeeTime `json:"data"`
}

type golfBackTeeTime struct {
	LocalDateTime string          `json:"localDateTime"`
	Rates         []golfBackRate  `json:"rates"`
	IsAvailable   bool            `json:"isAvailable"`
	PlayersMax    int             `json:"playersMax"`
}

type golfBackRate struct {
	Holes     int     `json:"holes"`
	IsPrimary bool    `json:"isPrimary"`
	Price     float64 `json:"price"`
}

func FetchGolfBack(config GolfBackCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://api.golfback.com/api/v1/courses/%s/date/%s/teetimes",
		config.CourseID, date,
	)

	var body []byte
	body, _ = json.Marshal(map[string]any{"sessionId": nil})

	var req *http.Request
	var err error
	req, err = http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

	var client http.Client = http.Client{Timeout: PlatformTimeout}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respBody []byte
	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data golfBackResponse
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.Data {
		if !tt.IsAvailable {
			continue
		}
		if tt.PlayersMax <= 0 {
			continue
		}

		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05", tt.LocalDateTime)
		if err != nil {
			continue
		}

		// Find primary 18-hole rate, fall back to first 18-hole, then first rate
		var price float64
		var holes string = "18"
		for _, r := range tt.Rates {
			if r.IsPrimary && r.Holes == 18 {
				price = r.Price
				break
			}
		}
		if price == 0 {
			for _, r := range tt.Rates {
				if r.Holes == 18 {
					price = r.Price
					break
				}
			}
		}
		if price == 0 && len(tt.Rates) > 0 {
			price = tt.Rates[0].Price
			if tt.Rates[0].Holes == 9 {
				holes = "9"
			}
		}

		results = append(results, DisplayTeeTime{
			Time:       t.Format("3:04 PM"),
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   tt.PlayersMax,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
