package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const MemberSportsAPIKey = "A9814038-9E19-4683-B171-5A06B39147FC"
const MemberSportsAPIURL = "https://api.membersports.com/api/v1/golfclubs/onlineBookingTeeTimes"

type MemberSportsCourseConfig struct {
	ClubID       int
	CourseID     int
	GroupID      int
	ConfigType   int
	BookingURL   string
	NamePrefix   string   // prepended to API course names if set
	KnownCourses []string // base course names this config serves
	City         string
	State        string
}

var MemberSportsCourses = map[string]MemberSportsCourseConfig{
	"denver": {
		ClubID:     3629,
		CourseID:   20573,
		GroupID:    1,
		ConfigType: 1,
		BookingURL: "https://app.membersports.com/tee-times/3629/20573/1/1/0",
		KnownCourses: []string{"City Park", "Kennedy", "Wellshire", "Willis Case", "Overland Park", "Harvard Gulch", "Kennedy Par 3 or Footgolf"},
		City: "Denver", State: "CO",
	},
	"foxhollow": {
		ClubID:     3703,
		CourseID:   20589,
		GroupID:    7,
		ConfigType: 0,
		BookingURL: "https://app.membersports.com/tee-times/3703/20589/0/7/0",
		KnownCourses: []string{"Fox Hollow", "Homestead"},
		City: "Lakewood", State: "CO",
	},
	"foothills": {
		ClubID:     3697,
		CourseID:   4758,
		GroupID:    3,
		ConfigType: 0,
		BookingURL: "https://app.membersports.com/tee-times/3697/4758/0/3/0",
		KnownCourses: []string{"Foothills 18", "Foothills Executive 9", "Foothills Par 3", "Meadows"},
		City: "Lakewood", State: "CO",
	},
	"brokentee": {
		ClubID:     3689,
		CourseID:   4748,
		GroupID:    0,
		ConfigType: 0,
		BookingURL: "https://app.membersports.com/tee-times/3689/4748/0/0/0",
		NamePrefix: "Broken Tee",
		KnownCourses: []string{"Broken Tee"},
		City: "Englewood", State: "CO",
	},
	"coalcreek": {
		ClubID:     3663,
		CourseID:   4714,
		GroupID:    0,
		ConfigType: 0,
		BookingURL: "https://app.membersports.com/tee-times/3663/4714/0/0/0",
		KnownCourses: []string{"Coal Creek"},
		City: "Louisville", State: "CO",
	},
}

type MemberSportsRequest struct {
	ConfigurationTypeId int    `json:"configurationTypeId"`
	Date                string `json:"date"`
	GolfClubGroupId     int    `json:"golfClubGroupId"`
	GolfClubId          int    `json:"golfClubId"`
	GolfCourseId        int    `json:"golfCourseId"`
	GroupSheetTypeId    int    `json:"groupSheetTypeId"`
}

type MemberSportsItem struct {
	Name                     string  `json:"name"`
	Price                    float64 `json:"price"`
	PlayerCount              int     `json:"playerCount"`
	HolesRequirementTypeId   int     `json:"holesRequirementTypeId"`
	GolfCourseNumberOfHoles  int     `json:"golfCourseNumberOfHoles"`
}

type MemberSportsSlot struct {
	TeeTime int                 `json:"teeTime"`
	Items   []MemberSportsItem  `json:"items"`
}

func fetchMemberSports(config MemberSportsCourseConfig, date string) ([]DisplayTeeTime, error) {
	var reqBody MemberSportsRequest = MemberSportsRequest{
		ConfigurationTypeId: config.ConfigType,
		Date:                date,
		GolfClubGroupId:     config.GroupID,
		GolfClubId:          config.ClubID,
		GolfCourseId:        config.CourseID,
		GroupSheetTypeId:    0,
	}

	var jsonData []byte
	var err error
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", MemberSportsAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", MemberSportsAPIKey)

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

	var slots []MemberSportsSlot
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

			var courseName string = strings.TrimSpace(item.Name)
			if config.NamePrefix != "" {
				if courseName == "Championship" || courseName == "" {
					courseName = config.NamePrefix
				} else {
					courseName = config.NamePrefix + " " + courseName
				}
			}

			results = append(results, DisplayTeeTime{
				Time:       timeStr,
				Course:     courseName,
				City:       config.City,
				State:      config.State,
				Openings:   openings,
				Holes:      holes,
				Price:      item.Price,
				BookingURL: config.BookingURL,
			})
		}
	}

	return results, nil
}
