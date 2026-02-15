package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PurposeGolfCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	CourseID    int    `json:"courseId"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
	BookingURL  string `json:"bookingUrl"`
}

var PurposeGolfCourses = map[string]PurposeGolfCourseConfig{}

type purposeGolfSlot struct {
	Time             string  `json:"Time"`
	TimeFormatted    string  `json:"TimeFormatted"`
	DateFormatted    string  `json:"DateFormatted"`
	Rate             int     `json:"Rate"`
	AvailableGolfers int     `json:"AvailableGolfers"`
	Inactive         bool    `json:"Inactive"`
	CourseId         int     `json:"CourseId"`
	FormattedPrice   string  `json:"FormattedPrice"`
}

func FetchPurposeGolf(config PurposeGolfCourseConfig, date string) ([]DisplayTeeTime, error) {
	url := fmt.Sprintf("https://booking.purposegolf.com/api/courses/%d/teeTimes", config.CourseID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", fmt.Sprintf("https://booking.purposegolf.com/courses/%s/%d/teetimes", config.Slug, config.CourseID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PurposeGolf %s: %w", config.Key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var slots []purposeGolfSlot
	if err := json.Unmarshal(body, &slots); err != nil {
		return nil, nil
	}

	// Parse target date to match against slot times
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	targetDay := targetDate.Format("2006-01-02")

	var results []DisplayTeeTime
	for _, slot := range slots {
		if slot.Inactive {
			continue
		}
		// Filter by date — Time is "2026-02-15T15:12:00"
		slotTime, err := time.Parse("2006-01-02T15:04:05", slot.Time)
		if err != nil {
			continue
		}
		if slotTime.Format("2006-01-02") != targetDay {
			continue
		}

		results = append(results, DisplayTeeTime{
			Time:       slot.TimeFormatted,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   slot.AvailableGolfers,
			Holes:      "18",
			Price:      float64(slot.Rate) / 100,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
