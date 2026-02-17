package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"sync"
	"time"
	"golf-teetimes/platforms"
)

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
}

type MetroPageData struct {
	Date  string
	Metro Metro
}

func handleLanding(w http.ResponseWriter, r *http.Request) {
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/landing.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.Execute(w, GetMetroList())
}

func handleMetroHome(w http.ResponseWriter, r *http.Request, metro Metro) {
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/home.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var today string = time.Now().Format("2006-01-02")
	tmpl.Execute(w, MetroPageData{Date: today, Metro: metro})
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

	// Launch MemberSports fetches for this metro
	for key, config := range platforms.MemberSportsCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.MemberSportsCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchMemberSports(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch Chronogolf fetches for this metro
	for key, config := range platforms.ChronogolfCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.ChronogolfCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchChronogolf(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch CPS Golf fetches for this metro
	for key, config := range platforms.CPSGolfCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.CPSGolfCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchCPSGolf(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch GolfNow fetches for this metro
	for key, config := range platforms.GolfNowCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.GolfNowCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchGolfNow(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch TeeItUp fetches for this metro
	for key, config := range platforms.TeeItUpCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.TeeItUpCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchTeeItUp(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch ClubCaddie fetches for this metro
	for key, config := range platforms.ClubCaddieCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.ClubCaddieCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchClubCaddie(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch Quick18 fetches for this metro
	for key, config := range platforms.Quick18Courses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.Quick18CourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchQuick18(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch GolfWithAccess fetches for this metro
	for key, config := range platforms.GolfWithAccessCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.GolfWithAccessCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchGolfWithAccess(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch CourseRev fetches for this metro
	for key, config := range platforms.CourseRevCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.CourseRevCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchCourseRev(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch RGuest fetches for this metro
	for key, config := range platforms.RGuestCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.RGuestCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchRGuest(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch CourseCo fetches for this metro
	for key, config := range platforms.CourseCoCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.CourseCoCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchCourseCo(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch TeeSnap fetches for this metro
	for key, config := range platforms.TeeSnapCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.TeeSnapCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchTeeSnap(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch ForeUp fetches for this metro
	for key, config := range platforms.ForeUpCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.ForeUpCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchForeUp(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Prophet Services disabled — WAF blocks most requests
	// for key, config := range platforms.ProphetCourses {
	// 	if config.Metro != metro.Slug {
	// 		continue
	// 	}
	// 	wg.Add(1)
	// 	go func(n string, c platforms.ProphetCourseConfig) {
	// 		defer wg.Done()
	// 		var results []platforms.DisplayTeeTime
	// 		var err error
	// 		results, err = platforms.FetchProphet(c, date)
	// 		ch <- fetchResult{results: results, err: err, name: n}
	// 	}(key, config)
	// }

	// Launch Purpose Golf fetches for this metro
	for key, config := range platforms.PurposeGolfCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.PurposeGolfCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchPurposeGolf(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch TeeQuest fetches for this metro
	for key, config := range platforms.TeeQuestCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.TeeQuestCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchTeeQuest(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch ResortSuite fetches for this metro
	for key, config := range platforms.ResortSuiteCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.ResortSuiteCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchResortSuite(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch BookTrump fetches for this metro
	for key, config := range platforms.BookTrumpCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.BookTrumpCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchBookTrump(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch TeeOn fetches for this metro
	for key, config := range platforms.TeeOnCourses {
		if config.Metro != metro.Slug {
			continue
		}
		wg.Add(1)
		go func(n string, c platforms.TeeOnCourseConfig) {
			defer wg.Done()
			var results []platforms.DisplayTeeTime
			var err error
			results, err = platforms.FetchTeeOn(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Close channel when all goroutines finish
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results
	var allResults []platforms.DisplayTeeTime
	for result := range ch {
		if result.err != nil {
			fmt.Println("Error fetching", result.name, ":", result.err)
		} else {
			allResults = append(allResults, result.results...)
		}
	}

	// Sort by time
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
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/alerts.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	tmpl.Execute(w, MetroPageData{Metro: metro})
}

func handlePrivacy(w http.ResponseWriter, r *http.Request) {
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/privacy.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func handleTerms(w http.ResponseWriter, r *http.Request) {
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/terms.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func handleGetAlerts(w http.ResponseWriter, r *http.Request) {
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
	if id == "" {
		http.Error(w, "ID required", 400)
		return
	}

	var err error = deleteAlert(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
