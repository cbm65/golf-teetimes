package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type ChronogolfCourseConfig struct {
	Key               string            `json:"key"`
	Metro             string            `json:"metro"`
	CourseIDs         string            `json:"courseIds"`
	ClubID            string            `json:"clubId"`
	NumericCourseID   string            `json:"numericCourseId"`
	AffiliationTypeID string            `json:"affiliationTypeId"`
	BookingURL        string            `json:"bookingUrl"`
	Names             map[string]string `json:"names"`
	City              string            `json:"city"`
	State             string            `json:"state"`
}

var ChronogolfCourses = map[string]ChronogolfCourseConfig{}

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
	GreenFee json.Number `json:"green_fee"`
}

type ChronogolfClubSlot struct {
	StartTime     string                   `json:"start_time"`
	OutOfCapacity bool                     `json:"out_of_capacity"`
	GreenFees     []ChronogolfClubGreenFee `json:"green_fees"`
}

type ChronogolfClubGreenFee struct {
	GreenFee json.Number `json:"green_fee"`
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

func toFloat(n json.Number) float64 {
	f, _ := n.Float64()
	return f
}

func FetchChronogolf(config ChronogolfCourseConfig, date string) ([]DisplayTeeTime, error) {
	if config.ClubID != "" {
		return FetchChronogolfClub(config, date)
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
		if displayName == "" {
			trimmed := strings.TrimSpace(courseName)
			for k, v := range config.Names {
				if strings.EqualFold(strings.TrimSpace(k), trimmed) {
					displayName = v
					break
				}
			}
		}
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
			Price:      toFloat(slot.DefaultPrice.GreenFee),
			BookingURL: config.BookingURL + "?date=" + date + "&step=teetimes",
		})
	}

	return results, nil
}

func FetchChronogolfClub(config ChronogolfCourseConfig, date string) ([]DisplayTeeTime, error) {
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
			price = toFloat(slot.GreenFees[0].GreenFee)
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     displayName,
			City:       config.City,
			State:      config.State,
			Openings:   len(slot.GreenFees),
			Holes:      "18",
			Price:      price,
			BookingURL: config.BookingURL + "?date=" + date + "&step=teetimes",
		})
	}

	return results, nil
}
