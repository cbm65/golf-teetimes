package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type GuestDeskCourse struct {
	CourseID    int    `json:"courseId"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

type GuestDeskSiteConfig struct {
	Key         string            `json:"key"`
	Metro       string            `json:"metro"`
	AccountSlug string            `json:"accountSlug"`
	PackageID   int               `json:"packageId"`
	BookingURL  string            `json:"bookingUrl"`
	Courses     []GuestDeskCourse `json:"courses"`
}

var GuestDeskCourses = map[string]GuestDeskSiteConfig{}

type guestDeskAvailRequest struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Adults    int    `json:"adults"`
	CourseIds []int  `json:"courseIds"`
}

type guestDeskTeeTime struct {
	CourseID int     `json:"CourseId"`
	TeeTime  string  `json:"TeeTime"`
	Rate     float64 `json:"Rate"`
}

func fetchGuestDeskForAdults(config GuestDeskSiteConfig, date string, adults int, courseIds []int) ([]guestDeskTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://reservations.guestdesk.com/%s/%d/en/Golf/Availability?webnode=main1&corsdomain=reservations.guestdesk.com",
		config.AccountSlug, config.PackageID,
	)

	var reqBody guestDeskAvailRequest = guestDeskAvailRequest{
		StartDate: date,
		EndDate:   date,
		Adults:    adults,
		CourseIds: courseIds,
	}
	var bodyBytes []byte
	var err error
	bodyBytes, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
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

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var times []guestDeskTeeTime
	err = json.Unmarshal(body, &times)
	if err != nil {
		return nil, err
	}
	return times, nil
}

func FetchGuestDesk(config GuestDeskSiteConfig, date string) ([]DisplayTeeTime, error) {
	// Collect all course IDs and build lookup
	var courseIds []int
	var courseMap = map[int]GuestDeskCourse{}
	for _, c := range config.Courses {
		courseIds = append(courseIds, c.CourseID)
		courseMap[c.CourseID] = c
	}

	// Fetch for adults=1,2,3,4 in parallel
	type adultResult struct {
		adults int
		times  []guestDeskTeeTime
		err    error
	}
	var ch = make(chan adultResult, 4)
	var wg sync.WaitGroup
	for g := 1; g <= 4; g++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			var t []guestDeskTeeTime
			var e error
			t, e = fetchGuestDeskForAdults(config, date, n, courseIds)
			ch <- adultResult{adults: n, times: t, err: e}
		}(g)
	}
	go func() { wg.Wait(); close(ch) }()

	// Key: "courseId|time" -> max adults that returned it
	type teeTimeInfo struct {
		courseID int
		teeTime string
		rate    float64
		maxAdults int
	}
	var infoMap = map[string]*teeTimeInfo{}

	for res := range ch {
		if res.err != nil {
			continue
		}
		for _, tt := range res.times {
			var key string = fmt.Sprintf("%d|%s", tt.CourseID, tt.TeeTime)
			if existing, ok := infoMap[key]; ok {
				if res.adults > existing.maxAdults {
					existing.maxAdults = res.adults
				}
			} else {
				infoMap[key] = &teeTimeInfo{
					courseID:   tt.CourseID,
					teeTime:   tt.TeeTime,
					rate:      tt.Rate,
					maxAdults: res.adults,
				}
			}
		}
	}

	var results []DisplayTeeTime
	for _, info := range infoMap {
		course, ok := courseMap[info.courseID]
		if !ok {
			continue
		}

		// Parse "2026-02-18T07:45:00"
		var t time.Time
		var err error
		t, err = time.Parse("2006-01-02T15:04:05", info.teeTime)
		if err != nil {
			continue
		}

		results = append(results, DisplayTeeTime{
			Time:       t.Format("3:04 PM"),
			Course:     course.DisplayName,
			City:       course.City,
			State:      course.State,
			Openings:   info.maxAdults,
			Holes:      "18",
			Price:      info.rate,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
