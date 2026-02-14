package platforms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type MemberSportsCourseConfig struct {
	Key          string   `json:"key"`
	Metro        string   `json:"metro"`
	APIKey       string   `json:"apiKey"`
	APIURL       string   `json:"apiUrl"`
	ClubID       int      `json:"clubId"`
	CourseID     int      `json:"courseId"`
	GroupID      int      `json:"groupId"`
	ConfigType   int      `json:"configType"`
	BookingURL   string   `json:"bookingUrl"`
	NamePrefix   string            `json:"namePrefix"`
	Names        map[string]string `json:"names"`
	KnownCourses []string          `json:"knownCourses"`
	City         string   `json:"city"`
	State        string   `json:"state"`
}

var MemberSportsCourses = map[string]MemberSportsCourseConfig{}

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

func FetchMemberSports(config MemberSportsCourseConfig, date string) ([]DisplayTeeTime, error) {
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
	req, err = http.NewRequest("POST", config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", config.APIKey)

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
			// Check names map first (like Chronogolf)
			if mapped, ok := config.Names[courseName]; ok {
				courseName = mapped
			} else {
				if config.NamePrefix != "" {
					if courseName == "Championship" || courseName == "" {
						courseName = config.NamePrefix
					} else {
						courseName = config.NamePrefix + " - " + courseName
					}
				}

				// Normalize variants to use " - " separator so the frontend
				// getBaseCourse() groups them under one dropdown entry.
				// "City Park Back Nine" → "City Park - Back Nine"
				// "Kennedy (Babe Lind / West)" → "Kennedy - Babe Lind / West"
				if strings.HasSuffix(courseName, " Back Nine") {
					courseName = strings.TrimSuffix(courseName, " Back Nine") + " - Back Nine"
				}
				if idx := strings.Index(courseName, " ("); idx > 0 && strings.HasSuffix(courseName, ")") {
					base := courseName[:idx]
					variant := courseName[idx+2 : len(courseName)-1]
					courseName = base + " - " + variant
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
