package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type EZLinksCourseConfig struct {
	BaseURL    string
	CourseIDs  []int
	BookingURL string
	Names      map[string]string
}

var EZLinksCourses = map[string]EZLinksCourseConfig{
	"aurora": {
		BaseURL:    "https://cityofaurora.ezlinksgolf.com",
		CourseIDs:  []int{24453, 24452, 24456, 24454, 6386, 6474, 19921, 19197, 6516},
		BookingURL: "https://cityofaurora.ezlinksgolf.com/index.html",
		Names: map[string]string{
			"Murphy Creek Golf Course":  "Murphy Creek",
			"Saddle Rock Golf Course":   "Saddle Rock",
			"Springhill Golf Course":    "Springhill",
			"Meadow Hills Golf Course":  "Meadow Hills",
			"Aurora Hills Golf Course":  "Aurora Hills",
		},
	},
}

type EZLinksSearchRequest struct {
	P01 []int  `json:"p01"`
	P02 string `json:"p02"`
	P03 string `json:"p03"`
	P04 string `json:"p04"`
	P05 int    `json:"p05"`
	P06 int    `json:"p06"`
	P07 bool   `json:"p07"`
}

type EZLinksResponse struct {
	R05 []EZLinksRateType `json:"r05"`
	R06 []EZLinksSlot     `json:"r06"`
}

type EZLinksRateType struct {
	R01 int    `json:"r01"`
	R02 int    `json:"r02"`
	R03 string `json:"r03"`
}

type EZLinksSlot struct {
	R06 int     `json:"r06"` // rate type ID
	R08 float64 `json:"r08"` // price
	R11 int     `json:"r11"` // available spots
	R14 int     `json:"r14"` // max players
	R16 string  `json:"r16"` // course name
	R24 string  `json:"r24"` // formatted time
	R28 string  `json:"r28"` // holes flags
}

func fetchEZLinks(config EZLinksCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Format date from "2026-02-10" to "02/10/2026"
	var parts []string = strings.Split(date, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid date format: %s", date)
	}
	var formattedDate string = parts[1] + "/" + parts[2] + "/" + parts[0]

	var reqBody EZLinksSearchRequest = EZLinksSearchRequest{
		P01: config.CourseIDs,
		P02: formattedDate,
		P03: "5:00 AM",
		P04: "7:00 PM",
		P05: 0,
		P06: 2,
		P07: false,
	}

	var jsonData []byte
	var err error
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", config.BaseURL+"/api/search/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")

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

	var data EZLinksResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	// Find the "Public" rate type ID from r05
	var publicRateID int = 0
	for _, rt := range data.R05 {
		if rt.R03 == "Public" {
			publicRateID = rt.R02
			break
		}
	}

	var results []DisplayTeeTime
	for _, slot := range data.R06 {
		// Only show Public rate entries
		if publicRateID != 0 && slot.R06 != publicRateID {
			continue
		}

		var courseName string = slot.R16
		var displayName string = config.Names[courseName]
		if displayName != "" {
			courseName = displayName
		}

		// Determine holes from r28 field
		// "9,15" = 9 holes, "1,15" or "1,7,15" = 18 holes
		var holes string = "18"
		if strings.HasPrefix(slot.R28, "9") {
			holes = "9"
		}

		results = append(results, DisplayTeeTime{
			Time:       slot.R24,
			Course:     courseName,
			Openings:   slot.R11,
			Holes:      holes,
			Price:      slot.R08,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
