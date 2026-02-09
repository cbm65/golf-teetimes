package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type CourseRevCourseConfig struct {
	SubDomain   string
	CourseID    int
	BookingURL  string
	DisplayName string
	City        string
	State       string
}

var CourseRevCourses = map[string]CourseRevCourseConfig{
	"wigwamgold": {
		SubDomain:   "wigwamgolfclub",
		CourseID:    137,
		BookingURL:  "https://wigwamgolfclub.bookings.courserev.ai/tee-times?courseId=137",
		DisplayName: "Wigwam Gold Course",
		City:        "Litchfield Park",
		State:       "AZ",
	},
	"wigwamblue": {
		SubDomain:   "wigwamgolfclub",
		CourseID:    138,
		BookingURL:  "https://wigwamgolfclub.bookings.courserev.ai/tee-times?courseId=138",
		DisplayName: "Wigwam Blue Course",
		City:        "Litchfield Park",
		State:       "AZ",
	},
	"wigwamred": {
		SubDomain:   "wigwamgolfclub",
		CourseID:    139,
		BookingURL:  "https://wigwamgolfclub.bookings.courserev.ai/tee-times?courseId=139",
		DisplayName: "Wigwam Red Course",
		City:        "Litchfield Park",
		State:       "AZ",
	},
}

type CourseRevRequest struct {
	CourseID    int    `json:"courseId"`
	BookingDate string `json:"bookingDate"`
	TeeTime     string `json:"teeTime"`
	Holes       string `json:"holes"`
}

type CourseRevResponse struct {
	Count   int                `json:"count"`
	Records []CourseRevTeeTime `json:"records"`
}

type CourseRevTeeTime struct {
	TeeDate  string             `json:"teeDate"`
	TeeTime  string             `json:"teeTime"`
	FreeSlots int               `json:"freeSlots"`
	Holes    int                `json:"holes"`
	Products []CourseRevProduct `json:"products"`
}

type CourseRevProduct struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func fetchCourseRev(config CourseRevCourseConfig, date string) ([]DisplayTeeTime, error) {
	var reqBody CourseRevRequest = CourseRevRequest{
		CourseID:    config.CourseID,
		BookingDate: date,
		TeeTime:     "anytime",
		Holes:       "18",
	}

	var jsonBody []byte
	var err error
	jsonBody, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", "https://api.courserev.ai/v2/prioritee/tee-time/list", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("api-key", "7b9284c665e9d913e093b497a8135d5e32537434")
	req.Header.Set("app-type", "white-label")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://"+config.SubDomain+".bookings.courserev.ai")
	req.Header.Set("Referer", "https://"+config.SubDomain+".bookings.courserev.ai/")

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

	var data CourseRevResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.Records {
		// Parse time "09:32" to "9:32 AM"
		var hour int = 0
		var minute int = 0
		fmt.Sscanf(tt.TeeTime, "%d:%d", &hour, &minute)

		var period string = "AM"
		var displayHour int = hour
		if hour >= 12 {
			period = "PM"
			if hour > 12 {
				displayHour = hour - 12
			}
		}
		if hour == 0 {
			displayHour = 12
		}
		var timeStr string = fmt.Sprintf("%d:%02d %s", displayHour, minute, period)

		var holes string = strconv.Itoa(tt.Holes)

		// Get price from first product
		var price float64 = 0
		if len(tt.Products) > 0 {
			price = tt.Products[0].Price
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   tt.FreeSlots,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
