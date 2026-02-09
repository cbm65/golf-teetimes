package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Quick18CourseConfig struct {
	Subdomain   string
	Domain      string // "quick18.com" or "play18.com", defaults to quick18.com
	BookingURL  string
	DisplayName string
	NamePrefix  string // if set, prepend to course name from HTML
	Holes       string
	City        string
	State       string
}

var Quick18Courses = map[string]Quick18CourseConfig{
	"papago": {
		Subdomain:   "papago",
		BookingURL:  "https://papago.quick18.com/teetimes/searchmatrix",
		DisplayName: "Papago Golf Course",
		Holes:       "18",
		City:        "Phoenix",
		State:       "AZ",
	},
	"grayhawk": {
		Subdomain:   "grayhawk",
		BookingURL:  "https://grayhawk.quick18.com/teetimes/searchmatrix",
		NamePrefix:  "Grayhawk",
		Holes:       "18",
		City:        "Scottsdale",
		State:       "AZ",
	},
	"trilogyvistancia": {
		Subdomain:   "trilogyvistancia",
		BookingURL:  "https://trilogyvistancia.quick18.com/teetimes/searchmatrix",
		DisplayName: "Trilogy at Vistancia",
		Holes:       "18",
		City:        "Peoria",
		State:       "AZ",
	},
	"coyotelakes": {
		Subdomain:   "coyotelakes",
		BookingURL:  "https://coyotelakes.quick18.com/teetimes/searchmatrix",
		NamePrefix:  "Coyote Lakes",
		Holes:       "18",
		City:        "Surprise",
		State:       "AZ",
	},
	"sunridgecanyon": {
		Subdomain:   "sunridgecanyon",
		Domain:      "play18.com",
		BookingURL:  "https://sunridgecanyon.play18.com/teetimes/searchmatrix",
		DisplayName: "SunRidge Canyon",
		Holes:       "18",
		City:        "Fountain Hills",
		State:       "AZ",
	},
	"orangetree": {
		Subdomain:   "orangetree",
		BookingURL:  "https://orangetree.quick18.com/teetimes/searchmatrix",
		DisplayName: "Orange Tree Golf Course",
		Holes:       "18",
		City:        "Scottsdale",
		State:       "AZ",
	},
	"goldcanyon": {
		Subdomain:   "goldcanyon",
		BookingURL:  "https://goldcanyon.quick18.com/teetimes/searchmatrix",
		NamePrefix:  "Gold Canyon",
		Holes:       "18",
		City:        "Gold Canyon",
		State:       "AZ",
	},
	"redmountainranch": {
		Subdomain:   "redmountain",
		Domain:      "play18.com",
		BookingURL:  "https://redmountain.play18.com/teetimes/searchmatrix",
		DisplayName: "Red Mountain Ranch Country Club",
		Holes:       "18",
		City:        "Mesa",
		State:       "AZ",
	},
	"thorncreek": {
		Subdomain:   "thorncreek",
		BookingURL:  "https://thorncreek.quick18.com/teetimes/searchmatrix",
		DisplayName: "Thorncreek Golf Club",
		Holes:       "18",
		City:        "Thornton",
		State:       "CO",
	},
}

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

func fetchQuick18(config Quick18CourseConfig, date string) ([]DisplayTeeTime, error) {
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
		if config.NamePrefix != "" && i < len(courses) {
			var htmlCourse string = strings.TrimSpace(courses[i][1])
			if htmlCourse != "" {
				if strings.HasPrefix(htmlCourse, config.NamePrefix) {
					courseName = htmlCourse
				} else {
					courseName = config.NamePrefix + " " + htmlCourse
				}
			}
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
			Holes:      config.Holes,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
