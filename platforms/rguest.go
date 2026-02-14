package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type RGuestCourseConfig struct {
	Key          string `json:"key"`
	Metro        string `json:"metro"`
	TenantID     string `json:"tenantId"`
	PropertyID   string `json:"propertyId"`
	CourseID     string `json:"courseId"`
	PlayerTypeID string `json:"playerTypeId"`
	BookingURL   string `json:"bookingUrl"`
	DisplayName  string `json:"displayName"`
	City         string `json:"city"`
	State        string `json:"state"`
}

var RGuestCourses = map[string]RGuestCourseConfig{}

type RGuestResponse struct {
	Success           bool              `json:"success"`
	AvailableTeeSlots []RGuestTeeGroup  `json:"availableTeeSlots"`
}

type RGuestTeeGroup struct {
	Slots []RGuestSlot `json:"slots"`
}

type RGuestSlot struct {
	ScheduleDateTime string         `json:"scheduleDateTime"`
	Availability     int            `json:"availability"`
	RateType         []RGuestRate   `json:"rateType"`
}

type RGuestRate struct {
	Name     string        `json:"name"`
	HoleType int           `json:"holeType"`
	Rates    RGuestFees    `json:"rates"`
}

type RGuestFees struct {
	GreenFee float64 `json:"greenFee"`
	CartFee  float64 `json:"cartFee"`
}

func FetchRGuest(config RGuestCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Generate a UUID-format session ID
	var r *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var sessionID string = fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		r.Uint32(), r.Uint32()&0xffff, r.Uint32()&0xffff, r.Uint32()&0xffff, r.Int63()&0xffffffffffff)

	// Step 1: Generate token to register the session
	var tokenURL string = fmt.Sprintf(
		"https://book.rguest.com/wbe-admin-service/generatetoken/v2/tenants/%s/propertyId/%s/appName/NA",
		config.TenantID, config.PropertyID,
	)
	var tokenReq *http.Request
	var err error
	tokenReq, err = http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return nil, err
	}
	tokenReq.Header.Set("Accept", "application/json, text/plain, */*")
	tokenReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	tokenReq.Header.Set("Referer", "https://book.rguest.com/onecart/golf/courses/"+config.TenantID+"/"+config.PropertyID)
	tokenReq.Header.Set("wbesessionid", sessionID)

	var client http.Client
	var tokenResp *http.Response
	tokenResp, err = client.Do(tokenReq)
	if err != nil {
		return nil, err
	}
	var tokenBody []byte
	tokenBody, _ = io.ReadAll(tokenResp.Body)
	tokenResp.Body.Close()

	// Extract JWT token from response
	var tokenData struct {
		Token string `json:"token"`
	}
	json.Unmarshal(tokenBody, &tokenData)

	// Step 2: Fetch available tee slots
	var url string = fmt.Sprintf(
		"https://book.rguest.com/wbe-golf-service/golf/tenants/%s/propertyId/%s/getAvailableTeeSlots?fromDate=%s&toDate=%s&courseId=%s&playerTypeId=%s&holes=0&appName=golf&dateTime=%sT06:00:00",
		config.TenantID, config.PropertyID, date, date, config.CourseID, config.PlayerTypeID, date,
	)

	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://book.rguest.com/onecart/golf/courses/%s/%s?date=%s&id=%s", config.TenantID, config.PropertyID, date, config.CourseID))
	req.Header.Set("wbesessionid", sessionID)
	req.Header.Set("propertydttm", date+"T00:00:00")
	req.Header.Set("timezone", "America/Phoenix")
	if tokenData.Token != "" {
		req.Header.Set("Authorization", "Bearer "+tokenData.Token)
	}

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


	var data RGuestResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}


	var results []DisplayTeeTime
	for _, group := range data.AvailableTeeSlots {
		for _, slot := range group.Slots {
			if slot.Availability <= 0 {
				continue
			}

			// Parse "2026-02-09T14:20:00"
			var t time.Time
			t, err = time.Parse("2006-01-02T15:04:05", slot.ScheduleDateTime)
			if err != nil {
				continue
			}
			var timeStr string = t.Format("3:04 PM")

			// Get the standard public rate ("Online Resort" or first non-pass/non-AZ rate)
			var price float64 = 0
			var holes string = "18"
			for _, rt := range slot.RateType {
				var total float64 = rt.Rates.GreenFee + rt.Rates.CartFee
				if total > 0 && (price == 0 || total > price) {
					price = total
					if rt.HoleType == 9 {
						holes = "9"
					} else {
						holes = "18"
					}
				}
			}

			results = append(results, DisplayTeeTime{
				Time:       timeStr,
				Course:     config.DisplayName,
				City:       config.City,
				State:      config.State,
				Openings:   slot.Availability,
				Holes:      holes,
				Price:      price,
				BookingURL: config.BookingURL,
			})
		}
	}

	return results, nil
}
