package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CPSGolfCourseConfig struct {
	Key        string            `json:"key"`
	Metro      string            `json:"metro"`
	BaseURL    string            `json:"baseUrl"`
	APIKey     string            `json:"apiKey"`
	WebsiteID  string            `json:"websiteId"`
	SiteID     string            `json:"siteId"`
	CourseIDs  string            `json:"courseIds"`
	BookingURL string            `json:"bookingUrl"`
	Names      map[string]string `json:"names"`
	City       string            `json:"city"`
	State      string            `json:"state"`
	Timezone   string            `json:"timezone"`
}

var CPSGolfCourses = map[string]CPSGolfCourseConfig{}

type CPSGolfResponse struct {
	TransactionID string          `json:"transactionId"`
	IsSuccess     bool            `json:"isSuccess"`
	Content       json.RawMessage `json:"content"`
}

type CPSGolfSlot struct {
	StartTime              string           `json:"startTime"`
	CourseName             string           `json:"courseName"`
	Holes                  int              `json:"holes"`
	Participants           int              `json:"participants"`
	MaxPlayer              int              `json:"maxPlayer"`
	ShItemPrices           []CPSGolfPrice   `json:"shItemPrices"`
}

type CPSGolfPrice struct {
	DisplayPrice float64 `json:"displayPrice"`
}

func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()>>32&0xFFFF,
		0x4000|(time.Now().UnixNano()>>48&0x0FFF),
		0x8000|(time.Now().UnixNano()>>60&0x3FFF),
		time.Now().UnixNano()&0xFFFFFFFFFFFF,
	)
}

func formatCPSDate(date string) string {
	var t time.Time
	var err error
	t, err = time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Mon Jan 02 2006")
}

func setCPSHeaders(req *http.Request, config CPSGolfCourseConfig) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if config.APIKey != "" {
		req.Header.Set("x-apikey", config.APIKey)
	}
	req.Header.Set("x-websiteid", config.WebsiteID)
	req.Header.Set("x-siteid", config.SiteID)
	req.Header.Set("x-componentid", "1")
	req.Header.Set("x-moduleid", "7")
	req.Header.Set("x-productid", "1")
	req.Header.Set("x-terminalid", "3")
	req.Header.Set("x-timezone-offset", "420")
	tz := config.Timezone
	if tz == "" {
		tz = "America/Denver"
	}
	req.Header.Set("x-timezoneid", tz)
	req.Header.Set("x-ismobile", "false")
	req.Header.Set("client-id", "onlineresweb")
	req.Header.Set("Referer", config.BaseURL+"/onlineresweb/search-teetime")
	req.Header.Set("Origin", config.BaseURL)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "Sat, 01 Jan 2000 00:00:00 GMT")
	req.Header.Set("If-Modified-Since", "0")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}

func FetchCPSGolf(config CPSGolfCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Step 1: Register a transaction ID
	var txnID string = generateUUID()
	var txnBody []byte
	var err error
	txnBody, err = json.Marshal(map[string]string{"transactionId": txnID})
	if err != nil {
		return nil, err
	}

	var txnReq *http.Request
	txnReq, err = http.NewRequest("POST", config.BaseURL+"/onlineres/onlineapi/api/v1/onlinereservation/RegisterTransactionId", bytes.NewBuffer(txnBody))
	if err != nil {
		return nil, err
	}
	txnReq.Header.Set("Content-Type", "application/json")
	setCPSHeaders(txnReq, config)

	var client http.Client
	var txnResp *http.Response
	txnResp, err = client.Do(txnReq)
	if err != nil {
		return nil, err
	}
	txnResp.Body.Close()

	// Step 2: Fetch tee times
	var searchDate string = formatCPSDate(date)
	var encodedDate string = url.PathEscape(searchDate)
	var teeURL string = fmt.Sprintf(
		"%s/onlineres/onlineapi/api/v1/onlinereservation/TeeTimes?searchDate=%s&holes=0&numberOfPlayer=0&courseIds=%s&searchTimeType=0&transactionId=%s&teeOffTimeMin=0&teeOffTimeMax=23&isChangeTeeOffTime=true&teeSheetSearchView=5&classCode=R&defaultOnlineRate=N&isUseCapacityPricing=false&memberStoreId=1&searchType=1",
		config.BaseURL, encodedDate, config.CourseIDs, txnID,
	)

	var req *http.Request
	req, err = http.NewRequest("GET", teeURL, nil)
	if err != nil {
		return nil, err
	}
	setCPSHeaders(req, config)
	req.Header.Set("x-requestid", generateUUID())

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

	var data CPSGolfResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, nil
	}

	var slots []CPSGolfSlot
	err = json.Unmarshal(data.Content, &slots)
	if err != nil {
		return nil, nil
	}

	var results []DisplayTeeTime
	for _, slot := range slots {
		// Parse time from "2026-02-08T12:00:00"
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05", slot.StartTime)
		if err != nil {
			continue
		}
		var timeStr string = t.Format("3:04 PM")

		var openings int = slot.MaxPlayer

		var price float64 = 0
		if len(slot.ShItemPrices) > 0 {
			price = slot.ShItemPrices[0].DisplayPrice
		}

		var courseName string = slot.CourseName
		var displayName string = config.Names[courseName]
		if displayName != "" {
			courseName = displayName
		}

		var holes string = fmt.Sprintf("%d", slot.Holes)

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     courseName,
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
