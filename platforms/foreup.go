package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ForeUpCourseConfig struct {
	Key          string `json:"key"`
	Metro        string `json:"metro"`
	CourseID     string `json:"courseId"`
	BookingClass string `json:"bookingClass"`
	ScheduleID   string `json:"scheduleId"`
	BookingURL   string `json:"bookingUrl"`
	DisplayName  string `json:"displayName"`
	City         string `json:"city"`
	State        string `json:"state"`
}

var ForeUpCourses = map[string]ForeUpCourseConfig{}

type ForeUpTeeTime struct {
	Time           string  `json:"time"`
	AvailableSpots int     `json:"available_spots"`
	GreenFee       float64 `json:"green_fee"`
	TeeSheetHoles  int     `json:"teesheet_holes"`
	CourseName     string  `json:"course_name"`
}

func FetchForeUp(config ForeUpCourseConfig, date string) ([]DisplayTeeTime, error) {
	// ForeUp expects date as MM-DD-YYYY
	var t time.Time
	var err error
	t, err = time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	var foreUpDate string = t.Format("01-02-2006")

	var url string = fmt.Sprintf(
		"https://foreupsoftware.com/index.php/api/booking/times?time=all&date=%s&holes=all&players=0&booking_class=%s&schedule_id=%s&schedule_ids%%5B%%5D=%s&specials_only=0&api_key=no_limits",
		foreUpDate, config.BookingClass, config.ScheduleID, config.ScheduleID,
	)

	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://foreupsoftware.com/index.php/booking/"+config.CourseID)

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

	var data []ForeUpTeeTime
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data {
		if tt.AvailableSpots <= 0 {
			continue
		}

		// Parse time "2026-02-09 13:48"
		var parsed time.Time
		parsed, err = time.Parse("2006-01-02 15:04", tt.Time)
		if err != nil {
			continue
		}
		var timeStr string = parsed.Format("3:04 PM")

		var holes string = "18"
		if tt.TeeSheetHoles == 9 {
			holes = "9"
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   tt.AvailableSpots,
			Holes:      holes,
			Price:      tt.GreenFee,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
