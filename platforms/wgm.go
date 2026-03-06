package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type WGMCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	CourseID    string `json:"courseId"`
	Holes       string `json:"holes"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var WGMCourses = map[string]WGMCourseConfig{}

type wgmResponse struct {
	Data []wgmTeeTime `json:"data"`
}

type wgmTeeTime struct {
	StartTime string    `json:"startTime"`
	FreeSlots int       `json:"freeSlots"`
	Date      string    `json:"date"`
	Rates     []wgmRate `json:"rates"`
}

type wgmRate struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func FetchWGM(config WGMCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://xq8v7un6ad.execute-api.us-east-1.amazonaws.com/prod/course/%s/tee-times?cached=false&date=%s",
		config.CourseID, date,
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

	var data wgmResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var holes string = config.Holes
	if holes == "" {
		holes = "18"
	}

	var results []DisplayTeeTime
	for _, tt := range data.Data {
		if tt.FreeSlots <= 0 {
			continue
		}

		var formatted string
		formatted, err = formatTime(tt.StartTime)
		if err != nil {
			continue
		}

		var price float64 = wgmGuestPrice(tt.Rates)

		results = append(results, DisplayTeeTime{
			Time:       formatted,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   tt.FreeSlots,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL + "?date=" + date,
		})
	}

	return results, nil
}

// wgmGuestPrice finds the guest walking rate, falling back to any guest rate, then first rate.
func wgmGuestPrice(rates []wgmRate) float64 {
	for _, r := range rates {
		if strings.Contains(r.Name, "Guest") && strings.Contains(r.Name, "Walking") {
			return r.Price
		}
	}
	for _, r := range rates {
		if strings.Contains(r.Name, "Guest") {
			return r.Price
		}
	}
	if len(rates) > 0 {
		return rates[0].Price
	}
	return 0
}

// formatTime converts "06:33" to "6:33 AM"
func formatTime(t string) (string, error) {
	if len(t) < 5 {
		return "", fmt.Errorf("invalid time: %s", t)
	}
	var hour, min int
	_, err := fmt.Sscanf(t, "%d:%d", &hour, &min)
	if err != nil {
		return "", err
	}
	var ampm string = "AM"
	if hour >= 12 {
		ampm = "PM"
	}
	if hour > 12 {
		hour -= 12
	}
	if hour == 0 {
		hour = 12
	}
	return fmt.Sprintf("%d:%02d %s", hour, min, ampm), nil
}
