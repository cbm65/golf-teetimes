package platforms

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
)

type TeeOnCourseConfig struct {
	Key           string `json:"key"`
	Metro         string `json:"metro"`
	CourseCode    string `json:"courseCode"`
	CourseGroupID string `json:"courseGroupId"`
	DisplayName   string `json:"displayName"`
	City          string `json:"city"`
	State         string `json:"state"`
	BookingURL    string `json:"bookingUrl"`
}

var TeeOnCourses = map[string]TeeOnCourseConfig{}

var (
	toTimeRe    = regexp.MustCompile(`(?s)<p class="time">\s*(\d{1,2}:\d{2})<span class="am-pm">(\w+)</span>`)
	toPriceRe   = regexp.MustCompile(`<p class="price">\$?([\d.]+)`)
	toPlayersRe = regexp.MustCompile(`(\d+)\s*-\s*(\d+)\s*Players`)
	toHolesRe   = regexp.MustCompile(`(\d+)\s*Holes`)
)

func FetchTeeOn(config TeeOnCourseConfig, date string) ([]DisplayTeeTime, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: PlatformTimeout}
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"

	// Step 1: GET TrailSearch — follows redirect to WebBookingAllTimesLanding, establishes session
	trailURL := "https://tee-on.com/PubGolf/servlet/com.teeon.teesheet.servlets.trail.TrailSearch?CourseGroupID=" + config.CourseGroupID + "&CourseCode=" + config.CourseCode + "&FromCourseWebsite=true&OriginalReferrer=" + config.BookingURL
	log.Printf("[TeeOn] %s: TrailSearch GET %s", config.Key, trailURL)
	trailReq, _ := http.NewRequest("GET", trailURL, nil)
	trailReq.Header.Set("User-Agent", ua)
	trailResp, err := client.Do(trailReq)
	if err != nil {
		log.Printf("[TeeOn] %s: TrailSearch error: %v", config.Key, err)
		return nil, fmt.Errorf("TeeOn %s trail: %w", config.Key, err)
	}
	trailBody, _ := io.ReadAll(trailResp.Body)
	trailResp.Body.Close()
	log.Printf("[TeeOn] %s: TrailSearch final URL %s, status %d, body %d bytes", config.Key, trailResp.Request.URL.String(), trailResp.StatusCode, len(trailBody))

	// Step 2: POST form to change date (the changeDate JS function does this)
	postURL := "https://tee-on.com/PubGolf/servlet/com.teeon.teesheet.servlets.golfersection.WebBookingAllTimesLanding?CourseCode=" + config.CourseCode + "&Referrer=tee-on.com"
	form := "Date=" + date + "&CourseCode=" + config.CourseCode + "&CourseGroupID=" + config.CourseGroupID
	log.Printf("[TeeOn] %s: POST %s body=%s", config.Key, postURL, form)
	req, _ := http.NewRequest("POST", postURL, strings.NewReader(form))
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", trailResp.Request.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[TeeOn] %s: HTTP error: %v", config.Key, err)
		return nil, fmt.Errorf("TeeOn %s: %w", config.Key, err)
	}
	defer resp.Body.Close()

	log.Printf("[TeeOn] %s: HTTP status %d", config.Key, resp.StatusCode)

	if resp.StatusCode != 200 {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TeeOn] %s: body read error: %v", config.Key, err)
		return nil, err
	}
	html := string(body)

	log.Printf("[TeeOn] %s: response length %d bytes", config.Key, len(html))

	blocks := strings.Split(html, `search-results-tee-times-box`)
	log.Printf("[TeeOn] %s: found %d blocks (first is preamble)", config.Key, len(blocks))

	var results []DisplayTeeTime
	for i, block := range blocks[1:] {
		timeMatch := toTimeRe.FindStringSubmatch(block)
		if timeMatch == nil {
			if i < 3 {
				truncated := block
				if len(truncated) > 200 {
					truncated = truncated[:200]
				}
				log.Printf("[TeeOn] %s: block %d no time match, starts with: %s", config.Key, i, truncated)
			}
			continue
		}
		teeTime := timeMatch[1] + " " + timeMatch[2]

		var price float64
		priceMatch := toPriceRe.FindStringSubmatch(block)
		if priceMatch != nil {
			price, _ = strconv.ParseFloat(priceMatch[1], 64)
		}

		openings := 4
		playersMatch := toPlayersRe.FindStringSubmatch(block)
		if playersMatch != nil {
			openings, _ = strconv.Atoi(playersMatch[2])
		}

		holes := "18"
		holesMatch := toHolesRe.FindStringSubmatch(block)
		if holesMatch != nil {
			holes = holesMatch[1]
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

	log.Printf("[TeeOn] %s: returning %d tee times", config.Key, len(results))

	return results, nil
}