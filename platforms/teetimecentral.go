package platforms

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type TeeTimeCentralCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	CourseCode  string `json:"courseCode"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var TeeTimeCentralCourses = map[string]TeeTimeCentralCourseConfig{}

const ttcBaseURL = "https://aberdeen.teetimecentralonline.com/golfbooking/globalteesheet/TeeTimesAjaxGO.aspx"
const ttcStaticParams = "i=00000000-0000-0000-0000-000000000000&sid=00000000-0000-0000-0000-000000000000&siteid=4b0b003a-dc21-4cea-9944-1d267a5cf30b&ts=omni&promo=100028&p=100028&rate=Package&src=PILA&c=a"

var ttcTimeRegex = regexp.MustCompile(`<h3>(\d+:\d+ [AP]M)</h3>`)
var ttcPriceRegex = regexp.MustCompile(`\$(\d+\.\d+)</h3>`)
var ttcValueRegex = regexp.MustCompile(`value="([A-Z]{4}\|[^"]+)"`)
var ttcHolesRegex = regexp.MustCompile(`nbholes="(\d+)"`)

type ttcSlot struct {
	Time  string
	Price float64
	Holes string
}

func fetchTTCForGolfers(config TeeTimeCentralCourseConfig, date string, golfers int) (map[string]ttcSlot, error) {
	// Date from 2026-01-15 to 1/15/2026
	var parts []string = strings.Split(date, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid date: %s", date)
	}
	var month int
	var day int
	month, _ = strconv.Atoi(parts[1])
	day, _ = strconv.Atoi(parts[2])
	var dateParam string = fmt.Sprintf("%d/%d/%s", month, day, parts[0])

	var url string = fmt.Sprintf("%s?%s&crs=%s&d=%s&g=%d&num=%d",
		ttcBaseURL, ttcStaticParams, config.CourseCode, dateParam, golfers, golfers)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

	var client http.Client = http.Client{Timeout: PlatformTimeout}
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
	var times [][]string = ttcTimeRegex.FindAllStringSubmatch(html, -1)
	var prices [][]string = ttcPriceRegex.FindAllStringSubmatch(html, -1)
	var holes [][]string = ttcHolesRegex.FindAllStringSubmatch(html, -1)

	var result map[string]ttcSlot = make(map[string]ttcSlot)
	for idx := 0; idx < len(times); idx++ {
		var slot ttcSlot
		slot.Time = times[idx][1]
		if idx < len(prices) {
			slot.Price, _ = strconv.ParseFloat(prices[idx][1], 64)
		}
		slot.Holes = "18"
		if idx < len(holes) {
			slot.Holes = holes[idx][1]
		}
		result[slot.Time] = slot
	}
	return result, nil
}

func FetchTeeTimeCentral(config TeeTimeCentralCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Fetch g=1..4 in parallel
	type gResult struct {
		golfers int
		slots   map[string]ttcSlot
		err     error
	}
	var ch chan gResult = make(chan gResult, 4)
	var wg sync.WaitGroup
	for g := 1; g <= 4; g++ {
		wg.Add(1)
		go func(golfers int) {
			defer wg.Done()
			var slots map[string]ttcSlot
			var err error
			slots, err = fetchTTCForGolfers(config, date, golfers)
			ch <- gResult{golfers: golfers, slots: slots, err: err}
		}(g)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	var byGolfers [5]map[string]ttcSlot
	for res := range ch {
		if res.err != nil {
			continue
		}
		byGolfers[res.golfers] = res.slots
	}

	// Use g=1 as the base set (all available times)
	var base map[string]ttcSlot = byGolfers[1]
	if base == nil {
		return nil, nil
	}

	var results []DisplayTeeTime
	for timeStr, slot := range base {
		var openings int = 1
		if byGolfers[2] != nil {
			if _, ok := byGolfers[2][timeStr]; ok {
				openings = 2
			}
		}
		if byGolfers[3] != nil {
			if _, ok := byGolfers[3][timeStr]; ok {
				openings = 3
			}
		}
		if byGolfers[4] != nil {
			if _, ok := byGolfers[4][timeStr]; ok {
				openings = 4
			}
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      slot.Holes,
			Price:      slot.Price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
