package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type ChronogolfCourseConfig struct {
	CourseIDs         string
	ClubID            string // for club-level API (alternative to CourseIDs/v2)
	NumericCourseID   string
	AffiliationTypeID string
	BookingURL        string
	Names             map[string]string // map API name to display name
	City              string
	State             string
}

var ChronogolfCourses = map[string]ChronogolfCourseConfig{
	"southsuburban": {
		CourseIDs:  "d75de2f0-634d-4dc5-b426-20d406a6f7cd,482fb33a-fa4a-48fb-85e1-e0492fe39d1a,68c1a9d5-f402-4d54-a1c5-991363899bc8",
		BookingURL: "https://www.chronogolf.com/club/south-suburban-golf-club",
		Names: map[string]string{
			"SSG 18 Hole Course":      "South Suburban",
			"SSG 9 Hole Par 3 Course": "South Suburban Par 3",
		},
		City: "Centennial", State: "CO",
	},
	"lonetree": {
		CourseIDs:  "001a7f2d-2c20-4bd9-8f91-3df9d051f737",
		BookingURL: "https://www.chronogolf.com/club/lone-tree-golf-club-hotel",
		Names: map[string]string{
			"LTH 18 Hole Course": "Lone Tree",
		},
		City: "Lone Tree", State: "CO",
	},
	"littleton": {
		CourseIDs:  "6a1ad175-7c4f-4692-a58f-7879e72ed9e9,c98df576-e507-44d7-9ece-7d59154fd143",
		BookingURL: "https://www.chronogolf.com/club/littleton-golf-tennis-club",
		Names: map[string]string{
			"LGT 18 Hole Course": "Littleton",
		},
		City: "Littleton", State: "CO",
	},
	"familysports": {
		CourseIDs:  "34b44f75-a475-4ec1-b5d3-e3089b66cf86",
		BookingURL: "https://www.chronogolf.com/club/family-sports-golf-course",
		Names: map[string]string{
			"FSC 9 Hole Course": "Family Sports",
		},
		City: "Centennial", State: "CO",
	},
	"highlandmeadows": {
		CourseIDs:  "438c425c-961a-4768-9ccd-c164e8bd8fb4",
		BookingURL: "https://www.chronogolf.com/club/highland-meadows-golf-club-colorado",
		Names: map[string]string{
			"Highland Meadows": "Highland Meadows",
		},
		City: "Windsor", State: "CO",
	},
	"broadlands": {
		CourseIDs:  "1af93d4a-dba3-481b-9980-487e5e516146",
		BookingURL: "https://www.chronogolf.com/club/broadlands-golf-course",
		Names: map[string]string{
			"Broadlands": "Broadlands",
		},
		City: "Broomfield", State: "CO",
	},
	"clubatsunrise": {
		CourseIDs:  "5b099aff-aa7c-4e50-8413-39da67a133e0",
		BookingURL: "https://www.chronogolf.com/club/the-club-at-sunrise",
		Names: map[string]string{
			"The Club at Sunrise": "Club at Sunrise",
		},
		City: "Las Vegas", State: "NV",
	},
}

type ChronogolfResponse struct {
	Status   string           `json:"status"`
	TeeTimes []ChronogolfSlot `json:"teetimes"`
}

type ChronogolfSlot struct {
	StartTime     string              `json:"start_time"`
	MaxPlayerSize int                 `json:"max_player_size"`
	Course        ChronogolfCourseAPI `json:"course"`
	DefaultPrice  ChronogolfPrice     `json:"default_price"`
}

type ChronogolfCourseAPI struct {
	Name          string `json:"name"`
	BookableHoles []int  `json:"bookable_holes"`
}

type ChronogolfPrice struct {
	GreenFee float64 `json:"green_fee"`
}

type ChronogolfClubSlot struct {
	StartTime     string                   `json:"start_time"`
	OutOfCapacity bool                     `json:"out_of_capacity"`
	GreenFees     []ChronogolfClubGreenFee `json:"green_fees"`
}

type ChronogolfClubGreenFee struct {
	GreenFee float64 `json:"green_fee"`
}

func formatHoles(holes []int) string {
	if len(holes) == 0 {
		return "18"
	}
	var max int = holes[0]
	for _, h := range holes {
		if h > max {
			max = h
		}
	}
	return strconv.Itoa(max)
}

func fetchChronogolf(config ChronogolfCourseConfig, date string) ([]DisplayTeeTime, error) {
	if config.ClubID != "" {
		return fetchChronogolfClub(config, date)
	}
	var allSlots []ChronogolfSlot
	var page int = 1

	for {
		var url string = fmt.Sprintf(
			"https://www.chronogolf.com/marketplace/v2/teetimes?start_date=%s&course_ids=%s&holes=9,18&page=%d",
			date, config.CourseIDs, page,
		)

		var req *http.Request
		var err error
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

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

		var data ChronogolfResponse
		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, err
		}

		if len(data.TeeTimes) == 0 {
			break
		}

		allSlots = append(allSlots, data.TeeTimes...)

		if len(data.TeeTimes) < 24 {
			break
		}

		page++
	}

	var results []DisplayTeeTime
	for _, slot := range allSlots {
		var hours int
		var mins int
		fmt.Sscanf(slot.StartTime, "%d:%d", &hours, &mins)

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

		var courseName string = slot.Course.Name
		var displayName string = config.Names[courseName]
		if displayName != "" {
			courseName = displayName
		}

		var holesStr string = formatHoles(slot.Course.BookableHoles)

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     courseName,
			City:       config.City,
			State:      config.State,
			Openings:   slot.MaxPlayerSize,
			Holes:      holesStr,
			Price:      slot.DefaultPrice.GreenFee,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}

func fetchChronogolfClub(config ChronogolfCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://www.chronogolf.com/marketplace/clubs/%s/teetimes?date=%s&course_id=%s&affiliation_type_ids%%5B%%5D=%s&nb_holes=18",
		config.ClubID, date, config.NumericCourseID, config.AffiliationTypeID,
	)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

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

	var slots []ChronogolfClubSlot
	err = json.Unmarshal(body, &slots)
	if err != nil {
		return nil, err
	}

	var displayName string
	for _, v := range config.Names {
		displayName = v
		break
	}

	var results []DisplayTeeTime
	for _, slot := range slots {
		if slot.OutOfCapacity {
			continue
		}

		var hours int
		var mins int
		fmt.Sscanf(slot.StartTime, "%d:%d", &hours, &mins)

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

		var price float64
		if len(slot.GreenFees) > 0 {
			price = slot.GreenFees[0].GreenFee
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     displayName,
			City:       config.City,
			State:      config.State,
			Openings:   len(slot.GreenFees),
			Holes:      "18",
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
