package platforms

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProphetCourseConfig struct {
	Key         string            `json:"key"`
	Metro       string            `json:"metro"`
	BaseURL     string            `json:"baseUrl"`
	Slug        string            `json:"slug"`
	CourseID    string            `json:"courseId"`
	CourseGUID  string            `json:"courseGuid"`
	SiteID      string            `json:"siteId"`
	BookingURL  string            `json:"bookingUrl"`
	Names       map[string]string `json:"names"`
	DisplayName string            `json:"displayName"`
	City        string            `json:"city"`
	State       string            `json:"state"`
}

var ProphetCourses = map[string]ProphetCourseConfig{}

// Cache successful Prophet results so WAF blocks don't lose data
var (
	prophetCache   = map[string][]DisplayTeeTime{}
	prophetCacheMu sync.RWMutex
	prophetLimiter sync.Mutex // serialize all Prophet requests to avoid WAF rate limit
)

func FetchProphet(config ProphetCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Serialize Prophet requests to avoid WAF rate limiting
	prophetLimiter.Lock()
	time.Sleep(3 * time.Second)
	prophetLimiter.Unlock()

	jar, _ := cookiejar.New(nil)
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
	baseURL := config.BaseURL

	// Pre-seed the required cookie
	psURL, _ := url.Parse(baseURL)
	jar.SetCookies(psURL, []*http.Cookie{
		{Name: "CPS.Online3.CurrentUICulture", Value: "en"},
	})

	client := http.Client{Jar: jar}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	prophetDate := fmt.Sprintf("%d-%d-%d", parsedDate.Year(), int(parsedDate.Month()), parsedDate.Day())

	// Single GET — follows redirects: Index → Index(S(...)) → nIndex(S(...)) → 200 with tee times
	indexURL := fmt.Sprintf("%s/Home/Index?CourseId=%s&Date=%s&Time=AnyTime&Player=4&Hole=18",
		baseURL, config.CourseID, prophetDate)

	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("GET", indexURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Prophet %s: GET error: %w", config.Key, err)
		}
		if resp.StatusCode == 200 {
			break
		}
		if attempt == 0 {
			log.Printf("[Prophet %s] WAF challenge (status %d), retrying...", config.Key, resp.StatusCode)
		}
		resp.Body.Close()
		if attempt < 2 {
			time.Sleep(2 * time.Second)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[Prophet %s] Non-200 status: %d, url: %s", config.Key, resp.StatusCode, resp.Request.URL.String())
		// Return cached results if available
		cacheKey := config.Key + ":" + date
		prophetCacheMu.RLock()
		cached := prophetCache[cacheKey]
		prophetCacheMu.RUnlock()
		if len(cached) > 0 {
			log.Printf("[Prophet %s] Returning %d cached tee times for %s", config.Key, len(cached), date)
		}
		return cached, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(respBody)
	// Each tee time block: teetime='3:55 PM' ... courseid ="4"
	// Price: <span class="teeTimePrice">$41.00</span>
	type rawSlot struct {
		time  string
		price float64
	}

	// Split HTML into tee time blocks by looking for teetime= attribute
	blocks := strings.Split(html, "teetime='")
	var slots []rawSlot
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]
		// Extract time
		endQuote := strings.Index(block, "'")
		if endQuote < 0 {
			continue
		}
		teeTime := block[:endQuote]

		// Skip duplicates — each tee time appears twice in the HTML
		if len(slots) > 0 && slots[len(slots)-1].time == teeTime {
			continue
		}

		// Extract price from this block
		var price float64
		priceIdx := strings.Index(block, `class="teeTimePrice">`)
		if priceIdx >= 0 {
			after := block[priceIdx+len(`class="teeTimePrice">`):]
			after = strings.TrimLeft(after, "$")
			endTag := strings.Index(after, "<")
			if endTag > 0 {
				price, _ = strconv.ParseFloat(strings.TrimSpace(after[:endTag]), 64)
			}
		}

		slots = append(slots, rawSlot{time: teeTime, price: price})
	}

	var results []DisplayTeeTime
	for _, slot := range slots {
		results = append(results, DisplayTeeTime{
			Time:       slot.time,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   4,
			Holes:      "18",
			Price:      slot.price,
			BookingURL: fmt.Sprintf("%s/Home/Index?CourseId=%s&Date=%s&Time=AnyTime&Player=4&Hole=18", baseURL, config.CourseID, prophetDate),
		})
	}

	// Cache results for WAF fallback
	cacheKey := config.Key + ":" + date
	prophetCacheMu.Lock()
	prophetCache[cacheKey] = results
	prophetCacheMu.Unlock()

	log.Printf("[Prophet %s] date=%s status=%d body=%d teetimes=%d", config.Key, date, resp.StatusCode, len(html), len(results))
	return results, nil
}
