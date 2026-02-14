package platforms

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Quick18CourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	Subdomain   string `json:"subdomain"`
	Domain      string `json:"domain"`
	BookingURL  string `json:"bookingUrl"`
	DisplayName string `json:"displayName"`
	NamePrefix  string `json:"namePrefix"`
	Holes       string `json:"holes"`
	City        string `json:"city"`
	State       string `json:"state"`
}

var Quick18Courses = map[string]Quick18CourseConfig{}

var quick18TimeRegex *regexp.Regexp = regexp.MustCompile(`mtrxTeeTimes">\s*(\d+:\d+)<div class="be_tee_time_ampm">(AM|PM)</div>`)
var quick18CourseRegex *regexp.Regexp = regexp.MustCompile(`mtrxCourse">(.*?)</td>`)
var quick18PlayersRegex *regexp.Regexp = regexp.MustCompile(`matrixPlayers">(.*?)</td>`)
var quick18PriceRegex *regexp.Regexp = regexp.MustCompile(`mtrxPrice">\$([\d.]+)</div>`)

func parseQuick18Players(text string) int {
	// "1 to 4 players" -> 4, "1 or 2 players" -> 2, "1 player" -> 1
	text = strings.TrimSpace(text)
	var numRegex *regexp.Regexp = regexp.MustCompile(`(\d+)`)
	var matches []string = numRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return 1
	}
	var last string = matches[len(matches)-1]
	var n int
	var err error
	n, err = strconv.Atoi(last)
	if err != nil {
		return 1
	}
	return n
}

func FetchQuick18(config Quick18CourseConfig, date string) ([]DisplayTeeTime, error) {
	// Convert date from 2026-01-15 to 20260115
	var dateClean string = strings.ReplaceAll(date, "-", "")

	var domain string = "quick18.com"
	if config.Domain != "" {
		domain = config.Domain
	}
	var url string = fmt.Sprintf("https://%s.%s/teetimes/searchmatrix?teedate=%s", config.Subdomain, domain, dateClean)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

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

	var html string = string(body)

	var times [][]string = quick18TimeRegex.FindAllStringSubmatch(html, -1)
	var courses [][]string = quick18CourseRegex.FindAllStringSubmatch(html, -1)
	var players [][]string = quick18PlayersRegex.FindAllStringSubmatch(html, -1)
	var prices [][]string = quick18PriceRegex.FindAllStringSubmatch(html, -1)

	var results []DisplayTeeTime
	for i := 0; i < len(times); i++ {
		var timeStr string = times[i][1] + " " + times[i][2]

		var courseName string = config.DisplayName
		var holes string = "18"
		if i < len(courses) {
			var htmlCourse string = strings.TrimSpace(courses[i][1])
			if htmlCourse != "" {
				lc := strings.ToLower(htmlCourse)
				if strings.Contains(lc, "back 9") || strings.Contains(lc, "front 9") || strings.HasSuffix(lc, " 9") {
					holes = "9"
				}
				if config.NamePrefix != "" {
					if strings.HasPrefix(htmlCourse, config.NamePrefix) {
						courseName = htmlCourse
					} else {
						courseName = config.NamePrefix + " " + htmlCourse
					}
				}
			}
		}
		if config.Holes != "" {
			holes = config.Holes
		}

		var openings int = 1
		if i < len(players) {
			openings = parseQuick18Players(players[i][1])
		}

		var price float64 = 0
		if i < len(prices) {
			price, _ = strconv.ParseFloat(prices[i][1], 64)
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     courseName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holes,
			Price:      price,
			BookingURL: config.BookingURL + "?teedate=" + dateClean,
		})
	}

	return results, nil
}
