package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type BookTrumpCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	BaseUrl     string `json:"baseUrl"`
	SponsorID   string `json:"sponsorId"`
	CourseID    string `json:"courseId"`
	PropertyID  string `json:"propertyId"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var BookTrumpCourses = map[string]BookTrumpCourseConfig{}

type bookTrumpResponse struct {
	Success         string              `json:"success"`
	TeeTimeResponse []bookTrumpTeeTime  `json:"teetime_response"`
}

type bookTrumpTeeTime struct {
	CourseID         string `json:"CourseID"`
	CourseName       string `json:"CourseName"`
	TeeTime          string `json:"TeeTime"`
	TWFourTime       string `json:"TWFourTime"`
	TeeTimeFee       string `json:"TeeTimeFee"`
	PlayersAvailable string `json:"PlayersAvailable"`
	NoOfHoles        int    `json:"NoOfHoles"`
}

func FetchBookTrump(config BookTrumpCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Convert date from 2006-01-02 to 01/02/2006
	var t time.Time
	var err error
	t, err = time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("BookTrump %s: invalid date: %v", config.Key, err)
	}
	var apiDate string = t.Format("01/02/2006")

	var url string = fmt.Sprintf(
		"%s/teetimes/fetch/teetimeavailability/?sponsorId=%s&courseId=%s&date=%s&propertyId=%s&noOfGolfers=2&vendor=&",
		config.BaseUrl, config.SponsorID, config.CourseID, apiDate, config.PropertyID,
	)

	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Referer", config.BaseUrl+"/tee-times")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	var client http.Client
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BookTrump %s: %v", config.Key, err)
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("BookTrump %s: read error: %v", config.Key, err)
	}

	var data bookTrumpResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("BookTrump %s: parse error: %v", config.Key, err)
	}

	var results []DisplayTeeTime
	for _, tt := range data.TeeTimeResponse {
		if tt.CourseID != config.CourseID {
			continue
		}

		var players int
		players, _ = strconv.Atoi(tt.PlayersAvailable)
		if players <= 0 {
			continue
		}

		var price float64
		price, _ = strconv.ParseFloat(tt.TeeTimeFee, 64)

		var holes string = "18"
		if tt.NoOfHoles == 9 {
			holes = "9"
		}

		results = append(results, DisplayTeeTime{
			Time:       tt.TeeTime,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   players,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
