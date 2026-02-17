package platforms

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TeeQuestCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	SiteID      string `json:"siteId"`
	CourseTag   string `json:"courseTag"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
	BookingURL  string `json:"bookingUrl"`
}

var TeeQuestCourses = map[string]TeeQuestCourseConfig{}

var (
	tqTimeRe  = regexp.MustCompile(`class="time-only">([^<]+)`)
	tqRateRe  = regexp.MustCompile(`class="rate">\$?([\d.]+)`)
	tqDescRe  = regexp.MustCompile(`(?s)class="booking-desc">\s*([^<]+)`)
	tqSlotsRe = regexp.MustCompile(`book-button book-(\d) availabl`)
)

func FetchTeeQuest(config TeeQuestCourseConfig, date string) ([]DisplayTeeTime, error) {
	jar, _ := cookiejar.New(nil)
	client := http.Client{Jar: jar, Timeout: PlatformTimeout}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	tqDate := fmt.Sprintf("%d/%d/%d 12:00:00 AM", int(parsedDate.Month()), parsedDate.Day(), parsedDate.Year())

	baseURL := fmt.Sprintf("https://teetimes.teequest.com/%s", config.SiteID)

	form := url.Values{
		"PaymentTab":      {"pay-at-course"},
		"Search.CourseTag": {config.CourseTag},
		"Search.Date":     {tqDate},
		"Search.Time":     {"Anytime"},
		"Search.Players":  {"0"},
		"Search.ImAMember": {"false"},
		"Search.MemberId": {""},
		"Search.Email":    {""},
	}

	req, err := http.NewRequest("POST", baseURL+"?paymentTab=pay-at-course", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://teetimes.teequest.com")
	req.Header.Set("Referer", baseURL+"?paymentTab=pay-at-course")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TeeQuest %s: %w", config.Key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	blocks := strings.Split(html, "<li>")
	var results []DisplayTeeTime
	for _, block := range blocks {
		timeMatch := tqTimeRe.FindStringSubmatch(block)
		if timeMatch == nil {
			continue
		}
		teeTime := strings.TrimSpace(timeMatch[1])

		var price float64
		rateMatch := tqRateRe.FindStringSubmatch(block)
		if rateMatch != nil {
			price, _ = strconv.ParseFloat(rateMatch[1], 64)
		}

		var holes string = "18"
		descMatch := tqDescRe.FindStringSubmatch(block)
		if descMatch != nil {
			desc := strings.TrimSpace(descMatch[1])
			if strings.Contains(desc, "9 holes") {
				holes = "9"
			}
		}

		slotMatches := tqSlotsRe.FindAllStringSubmatch(block, -1)
		openings := len(slotMatches)
		if openings == 0 {
			openings = 4
		}

		results = append(results, DisplayTeeTime{
			Time:       teeTime,
			Course:     config.DisplayName,
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
