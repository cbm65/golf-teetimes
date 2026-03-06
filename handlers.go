package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"golf-teetimes/platforms"
)

// Rate limiter for alert creation — per IP, max 5 requests per minute
var alertRateLimit struct {
	sync.Mutex
	hits map[string][]time.Time
}

func init() {
	alertRateLimit.hits = make(map[string][]time.Time)
}

func alertRateLimited(r *http.Request) bool {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.SplitN(fwd, ",", 2)[0]
		ip = strings.TrimSpace(ip)
	}

	alertRateLimit.Lock()
	defer alertRateLimit.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	// Remove old entries
	recent := alertRateLimit.hits[ip][:0]
	for _, t := range alertRateLimit.hits[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= 5 {
		alertRateLimit.hits[ip] = recent
		return true
	}

	alertRateLimit.hits[ip] = append(recent, now)
	return false
}

type fetchResult struct {
	results []platforms.DisplayTeeTime
	err     error
	name    string
}

// Minimal singleflight — collapses concurrent calls with the same key into one
type flightCall struct {
	wg  sync.WaitGroup
	val interface{}
}

type flightGroup struct {
	mu sync.Mutex
	m  map[string]*flightCall
}

func (g *flightGroup) Do(key string, fn func() interface{}) interface{} {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*flightCall)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val
	}
	c := &flightCall{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val
}

// In-memory tee time cache — collapses concurrent user requests into one upstream fetch per metro+date
var teeTimeCache struct {
	sync.RWMutex
	entries map[string]cachedTeeTimes
}

var teeTimeFlight flightGroup

type cachedTeeTimes struct {
	data    []platforms.DisplayTeeTime
	fetched time.Time
}

const teeTimeCacheTTL = 5 * time.Minute

func init() {
	teeTimeCache.entries = make(map[string]cachedTeeTimes)

	// Evict expired cache entries every 10 minutes
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			teeTimeCache.Lock()
			for key, entry := range teeTimeCache.entries {
				if time.Since(entry.fetched) > teeTimeCacheTTL {
					delete(teeTimeCache.entries, key)
				}
			}
			teeTimeCache.Unlock()
		}
	}()
}

type MetroPageData struct {
	Date  string
	Metro Metro
}

var (
	tmplLanding = template.Must(template.ParseFiles("templates/landing.html"))
	tmplHome    = template.Must(template.ParseFiles("templates/home.html"))
	tmplAlerts  = template.Must(template.ParseFiles("templates/alerts.html"))
	tmplPrivacy = template.Must(template.ParseFiles("templates/privacy.html"))
	tmplTerms   = template.Must(template.ParseFiles("templates/terms.html"))
	tmplOptIn   = template.Must(template.ParseFiles("templates/optin.html"))
)

func handleLanding(w http.ResponseWriter, r *http.Request) {
	tmplLanding.Execute(w, GetMetroList())
}

func handleMetroHome(w http.ResponseWriter, r *http.Request, metro Metro) {
	var today string = time.Now().Format("2006-01-02")
	tmplHome.Execute(w, MetroPageData{Date: today, Metro: metro})
}

func handleMetroTeeTimes(w http.ResponseWriter, r *http.Request, metro Metro) {
	var date string = r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.Error(w, "Invalid date format", 400)
		return
	}

	// Check cache
	cacheKey := metro.Slug + ":" + date
	teeTimeCache.RLock()
	if cached, ok := teeTimeCache.entries[cacheKey]; ok && time.Since(cached.fetched) < teeTimeCacheTTL {
		teeTimeCache.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached.data)
		return
	}
	teeTimeCache.RUnlock()

	// singleflight: if multiple requests hit the same cache miss, only one fetches
	val := teeTimeFlight.Do(cacheKey, func() interface{} {
		// Double-check cache inside singleflight (another goroutine may have just filled it)
		teeTimeCache.RLock()
		if cached, ok := teeTimeCache.entries[cacheKey]; ok && time.Since(cached.fetched) < teeTimeCacheTTL {
			teeTimeCache.RUnlock()
			return cached.data
		}
		teeTimeCache.RUnlock()

		return fetchMetroTeeTimes(metro, date)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(val)
}

func fetchMetroTeeTimes(metro Metro, date string) []platforms.DisplayTeeTime {
	var ch chan fetchResult = make(chan fetchResult)
	var wg sync.WaitGroup

	for i := range platforms.Registry {
		entry := &platforms.Registry[i]
		if entry.Metro != metro.Slug || !entry.Enabled {
			continue
		}
		wg.Add(1)
		go func(e *platforms.CourseEntry) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = e.Fetch(date)
			ch <- fetchResult{results: results, err: err, name: e.Key}
		}(entry)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var allResults []platforms.DisplayTeeTime
	for result := range ch {
		if result.err != nil {
			fmt.Println("Error fetching", result.name, ":", result.err)
		} else {
			allResults = append(allResults, result.results...)
		}
	}

	for i := range allResults {
		if allResults[i].Openings > 4 {
			allResults[i].Openings = 4
		}
	}

	sort.Slice(allResults, func(i int, j int) bool {
		var iMins int = parseTimeToMinutes(allResults[i].Time)
		var jMins int = parseTimeToMinutes(allResults[j].Time)
		return iMins < jMins
	})

	// Cache results
	cacheKey := metro.Slug + ":" + date
	teeTimeCache.Lock()
	teeTimeCache.entries[cacheKey] = cachedTeeTimes{data: allResults, fetched: time.Now()}
	teeTimeCache.Unlock()

	return allResults
}

func handleMetroAlerts(w http.ResponseWriter, r *http.Request, metro Metro) {
	tmplAlerts.Execute(w, MetroPageData{Metro: metro})
}

func handlePrivacy(w http.ResponseWriter, r *http.Request) {
	tmplPrivacy.Execute(w, nil)
}

func handleTerms(w http.ResponseWriter, r *http.Request) {
	tmplTerms.Execute(w, nil)
}

func handleOptIn(w http.ResponseWriter, r *http.Request) {
	tmplOptIn.Execute(w, nil)
}

func handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var phone string = r.URL.Query().Get("phone")
	if phone == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]platforms.Alert{})
		return
	}

	var alerts []platforms.Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var filtered []platforms.Alert
	for _, a := range alerts {
		if a.Phone == phone {
			filtered = append(filtered, a)
		}
	}
	if filtered == nil {
		filtered = []platforms.Alert{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	if alertRateLimited(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests. Please try again in a minute."})
		return
	}

	var incoming platforms.Alert
	var err error
	err = json.NewDecoder(r.Body).Decode(&incoming)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body."})
		return
	}

	if incoming.Phone == "" || incoming.Course == "" || incoming.Date == "" || incoming.StartTime == "" || incoming.EndTime == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "All fields are required."})
		return
	}

	var alert platforms.Alert
	alert, err = addAlert(incoming.Phone, incoming.Course, incoming.Date, incoming.StartTime, incoming.EndTime, incoming.MinPlayers, incoming.Holes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

func handleDeleteAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var id string = r.URL.Query().Get("id")
	var phone string = r.URL.Query().Get("phone")
	if id == "" || phone == "" {
		http.Error(w, "ID and phone required", 400)
		return
	}

	var err error = deleteAlertByOwner(id, phone)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
