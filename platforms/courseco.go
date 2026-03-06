package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type CourseCoCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	Subdomain   string `json:"subdomain"`
	CourseID    string `json:"courseId"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
	GatewayURL  string `json:"gatewayUrl"`
	OriginURL   string `json:"originUrl"`
}

var CourseCoCourses = map[string]CourseCoCourseConfig{}

type CourseCoResponse struct {
	TeeTimeData []CourseCoTeeTime `json:"TeeTimeData"`
}

type CourseCoTeeTime struct {
	Title         string  `json:"Title"`
	PerPlayerCost float64 `json:"PerPlayerCost"`
	AvailableSlot string  `json:"AvailableSlot"`
	Time          string  `json:"Time"`
	Allow18       bool    `json:"Allow18"`
	Allow9        bool    `json:"Allow9"`
}

func FetchCourseCo(config CourseCoCourseConfig, date string) ([]DisplayTeeTime, error) {
	gatewayBase := config.GatewayURL
	if gatewayBase == "" {
		gatewayBase = "https://courseco-gateway.totaleintegrated.net"
	}
	origin := config.OriginURL
	if origin == "" {
		origin = "https://" + config.Subdomain + ".totaleintegrated.net"
	}

	// Encode spaces but preserve commas for multi-course IDs
	escapedCourseID := strings.ReplaceAll(config.CourseID, " ", "%20")

	var apiURL string = fmt.Sprintf(
		"%s/Booking/Teetimes?IsInitTeeTimeRequest=false&TeeTimeDate=%s&CourseID=%s&StartTime=05:00&EndTime=21:00&NumOfPlayers=-1&Holes=Any&IsNineHole=0&StartPrice=0&EndPrice=&CartIncluded=false&SpecialsOnly=0&IsClosest=0&PlayerIDs=&DateFilterChange=false&DateFilterChangeNoSearch=false&SearchByGroups=true&IsPrepaidOnly=0&QueryStringFilters=null",
		gatewayBase, date, escapedCourseID,
	)

	log.Printf("[CourseCo] %s: GET %s", config.Key, apiURL)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")

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

	log.Printf("[CourseCo] %s: status %d, body %d bytes", config.Key, resp.StatusCode, len(body))

	var data CourseCoResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.TeeTimeData {
		var holes string = "18"
		if !tt.Allow18 && tt.Allow9 {
			holes = "9"
		}

		// Parse availability from "2-4" format
		var openings int = 1
		if tt.AvailableSlot != "" {
			var parts []string = strings.Split(tt.AvailableSlot, "-")
			if len(parts) == 2 {
				openings, _ = strconv.Atoi(parts[1])
			} else if len(parts) == 1 {
				openings, _ = strconv.Atoi(parts[0])
			}
		}

		results = append(results, DisplayTeeTime{
			Time:       tt.Title,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holes,
			Price:      tt.PerPlayerCost,
			BookingURL: config.BookingURL,
		})
	}

	log.Printf("[CourseCo] %s: returning %d tee times", config.Key, len(results))

	return results, nil
}
