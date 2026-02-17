package platforms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ClubCaddieCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	BaseURL     string `json:"baseUrl"`
	APIKey      string `json:"apiKey"`
	CourseID    string `json:"courseId"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var ClubCaddieCourses = map[string]ClubCaddieCourseConfig{}

type ClubCaddieSlot struct {
	StartTime        string                  `json:"StartTime"`
	PlayersAvailable int                     `json:"PlayersAvailable"`
	HighestPrice     float64                 `json:"HighestPrice"`
	LowestPrice      float64                 `json:"LowestPrice"`
	HoleGroup        string                  `json:"HoleGroup"`
	PricingPlan      []ClubCaddiePricingPlan `json:"PricingPlan"`
}

type ClubCaddiePricingPlan struct {
	HoleRate9  *float64 `json:"HoleRate_9"`
	HoleRate18 *float64 `json:"HoleRate_18"`
	TitleType  string   `json:"TitleType"`
}

func FetchClubCaddie(config ClubCaddieCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Format date as MM/DD/YYYY
	var t time.Time
	var err error
	t, err = time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	var formDate string = t.Format("01/02/2006")
	var encodedDate string = url.QueryEscape(formDate)

	// Step 1: GET the slots page to establish a session and get Interaction ID
	var pageURL string = fmt.Sprintf("%s/webapi/view/%s/slots?date=%s&player=1&ratetype=any",
		config.BaseURL, config.APIKey, encodedDate)

	var client http.Client = http.Client{Timeout: PlatformTimeout}
	var pageResp *http.Response
	pageResp, err = client.Get(pageURL)
	if err != nil {
		return nil, err
	}
	var pageBody []byte
	pageBody, err = io.ReadAll(pageResp.Body)
	pageResp.Body.Close()
	if err != nil {
		return nil, err
	}

	// Extract Interaction ID from the page
	var interactionRe *regexp.Regexp = regexp.MustCompile(`Interaction=([a-zA-Z0-9]+)`)
	var interactionMatch []string = interactionRe.FindStringSubmatch(string(pageBody))
	var interaction string = ""
	if len(interactionMatch) > 1 {
		interaction = interactionMatch[1]
	}

	// Step 2: POST to get tee times
	var formData url.Values = url.Values{
		"date":        {formDate},
		"player":      {"1"},
		"holes":       {"any"},
		"fromtime":    {"0"},
		"totime":      {"23"},
		"minprice":    {"0"},
		"maxprice":    {"9999"},
		"ratetype":    {"any"},
		"HoleGroup":   {"front"},
		"CourseId":    {config.CourseID},
		"apikey":      {config.APIKey},
		"Interaction": {interaction},
	}

	var req *http.Request
	req, err = http.NewRequest("POST", config.BaseURL+"/webapi/TeeTimes", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", config.BaseURL)
	req.Header.Set("Referer", pageURL)

	// Copy cookies from page response
	for _, cookie := range pageResp.Cookies() {
		req.AddCookie(cookie)
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

	var html string = string(body)

	// Extract slot JSON from hidden inputs
	var re *regexp.Regexp = regexp.MustCompile(`name="slot" value="([^"]+)"`)
	var matches [][]string = re.FindAllStringSubmatch(html, -1)

	var results []DisplayTeeTime
	for _, match := range matches {
		var decoded string
		decoded, err = url.QueryUnescape(match[1])
		if err != nil {
			continue
		}

		var slot ClubCaddieSlot
		err = json.Unmarshal([]byte(decoded), &slot)
		if err != nil {
			continue
		}

		// Parse time from "08:36:00"
		var slotTime time.Time
		slotTime, err = time.Parse("15:04:05", slot.StartTime)
		if err != nil {
			continue
		}
		var timeStr string = slotTime.Format("3:04 PM")

		var price float64 = slot.LowestPrice

		var holes string = "18"
		if len(slot.PricingPlan) > 0 {
			var pp ClubCaddiePricingPlan = slot.PricingPlan[0]
			if pp.HoleRate18 != nil {
				holes = "18"
			} else if pp.HoleRate9 != nil {
				holes = "9"
			}
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   slot.PlayersAvailable,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
