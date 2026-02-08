package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const DenverAPIKey = "A9814038-9E19-4683-B171-5A06B39147FC"
const DenverAPIURL = "https://api.membersports.com/api/v1/golfclubs/onlineBookingTeeTimes"
const DenverClubID = 3629
const DenverCourseID = 20573
const DenverBookingURL = "https://app.membersports.com/tee-times/3629/20573/1/1/0"

type DenverRequest struct {
	ConfigurationTypeId int    `json:"configurationTypeId"`
	Date                string `json:"date"`
	GolfClubGroupId     int    `json:"golfClubGroupId"`
	GolfClubId          int    `json:"golfClubId"`
	GolfCourseId        int    `json:"golfCourseId"`
	GroupSheetTypeId    int    `json:"groupSheetTypeId"`
}

type DenverItem struct {
	Name                   string  `json:"name"`
	Price                  float64 `json:"price"`
	PlayerCount            int     `json:"playerCount"`
	HolesRequirementTypeId int     `json:"holesRequirementTypeId"`
}

type DenverSlot struct {
	TeeTime int          `json:"teeTime"`
	Items   []DenverItem `json:"items"`
}

func fetchDenver(date string) ([]DisplayTeeTime, error) {
	var reqBody DenverRequest = DenverRequest{
		ConfigurationTypeId: 1,
		Date:                date,
		GolfClubGroupId:     1,
		GolfClubId:          DenverClubID,
		GolfCourseId:        DenverCourseID,
		GroupSheetTypeId:    0,
	}

	var jsonData []byte
	var err error
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", DenverAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", DenverAPIKey)

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

	var slots []DenverSlot
	err = json.Unmarshal(body, &slots)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, slot := range slots {
		for _, item := range slot.Items {
			var hours int = slot.TeeTime / 60
			var mins int = slot.TeeTime % 60

			var period string = "AM"
			if hours >= 12 {
				period = "PM"
			}
			if hours > 12 {
				hours = hours - 12
			}
			if hours == 0 {
				hours = 12
			}
			var timeStr string = fmt.Sprintf("%d:%02d %s", hours, mins, period)

			var openings int = 4 - item.PlayerCount
			if openings < 0 {
				openings = 0
			}
			var holes string = "9"
			if item.HolesRequirementTypeId == 2 {
				holes = "18"
			}

			results = append(results, DisplayTeeTime{
				Time:       timeStr,
				Course:     item.Name,
				Openings:   openings,
				Holes:      holes,
				Price:      item.Price,
				BookingURL: DenverBookingURL,
			})
		}
	}

	return results, nil
}
