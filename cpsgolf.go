package main

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
	BaseURL    string
	APIKey     string
	WebsiteID  string
	SiteID     string
	CourseIDs  string
	BookingURL string
	Names      map[string]string
}

var CPSGolfCourses = map[string]CPSGolfCourseConfig{
	"greenvalleyranch": {
		BaseURL:    "https://greenvalleyranch.cps.golf",
		APIKey:     "8ea2914e-cac2-48a7-a3e5-e0f41350bf3a",
		WebsiteID:  "e6b92812-d6c4-4f86-7eea-08d9fadf154d",
		SiteID:     "2",
		CourseIDs:  "1",
		BookingURL: "https://greenvalleyranch.cps.golf/onlineresweb/search-teetime",
		Names: map[string]string{
			"Green Valley Ranch": "Green Valley Ranch",
		},
	},
}

type CPSGolfResponse struct {
	TransactionID string        `json:"transactionId"`
	IsSuccess     bool          `json:"isSuccess"`
	Content       []CPSGolfSlot `json:"content"`
}

type CPSGolfSlot struct {
	StartTime   string           `json:"startTime"`
	CourseName  string           `json:"courseName"`
	Holes       int              `json:"holes"`
	Participants int             `json:"participants"`
	BookingList []interface{}    `json:"bookingList"`
	ShItemPrices []CPSGolfPrice  `json:"shItemPrices"`
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
	req.Header.Set("x-apikey", config.APIKey)
	req.Header.Set("x-websiteid", config.WebsiteID)
	req.Header.Set("x-siteid", config.SiteID)
	req.Header.Set("x-componentid", "1")
	req.Header.Set("x-moduleid", "7")
	req.Header.Set("x-productid", "1")
	req.Header.Set("x-terminalid", "3")
	req.Header.Set("x-timezone-offset", "420")
	req.Header.Set("x-timezoneid", "America/Denver")
	req.Header.Set("x-ismobile", "false")
	req.Header.Set("client-id", "onlineresweb")
}

func fetchCPSGolf(config CPSGolfCourseConfig, date string) ([]DisplayTeeTime, error) {
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
	var encodedDate string = url.QueryEscape(searchDate)
	var teeURL string = fmt.Sprintf(
		"%s/onlineres/onlineapi/api/v1/onlinereservation/TeeTimes?searchDate=%s&holes=18&numberOfPlayer=0&courseIds=%s&searchTimeType=0&transactionId=%s&teeOffTimeMin=0&teeOffTimeMax=23&isChangeTeeOffTime=true&teeSheetSearchView=5&classCode=R&defaultOnlineRate=N&isUseCapacityPricing=false&memberStoreId=1&searchType=1",
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
		return nil, err
	}

	var results []DisplayTeeTime
	for _, slot := range data.Content {
		// Parse time from "2026-02-08T12:00:00"
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05", slot.StartTime)
		if err != nil {
			continue
		}
		var timeStr string = t.Format("3:04 PM")

		var openings int = slot.Participants - len(slot.BookingList)
		if openings < 0 {
			openings = 0
		}

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
			Openings:   openings,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
