package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"sync"
	"time"
)

type fetchResult struct {
	results []DisplayTeeTime
	err     error
	name    string
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

	var ch chan fetchResult = make(chan fetchResult)
	var wg sync.WaitGroup

	// Launch MemberSports fetches for this metro
	for _, key := range metro.MemberSportsKeys {
		var config MemberSportsCourseConfig
		var exists bool
		config, exists = MemberSportsCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c MemberSportsCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchMemberSports(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch Chronogolf fetches for this metro
	for _, key := range metro.ChronogolfKeys {
		var config ChronogolfCourseConfig
		var exists bool
		config, exists = ChronogolfCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c ChronogolfCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchChronogolf(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch CPS Golf fetches for this metro
	for _, key := range metro.CPSGolfKeys {
		var config CPSGolfCourseConfig
		var exists bool
		config, exists = CPSGolfCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c CPSGolfCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchCPSGolf(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch GolfNow fetches for this metro
	for _, key := range metro.GolfNowKeys {
		var config GolfNowCourseConfig
		var exists bool
		config, exists = GolfNowCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c GolfNowCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchGolfNow(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch TeeItUp fetches for this metro
	for _, key := range metro.TeeItUpKeys {
		var config TeeItUpCourseConfig
		var exists bool
		config, exists = TeeItUpCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c TeeItUpCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchTeeItUp(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch ClubCaddie fetches for this metro
	for _, key := range metro.ClubCaddieKeys {
		var config ClubCaddieCourseConfig
		var exists bool
		config, exists = ClubCaddieCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c ClubCaddieCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchClubCaddie(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch Quick18 fetches for this metro
	for _, key := range metro.Quick18Keys {
		var config Quick18CourseConfig
		var exists bool
		config, exists = Quick18Courses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c Quick18CourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchQuick18(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch GolfWithAccess fetches for this metro
	for _, key := range metro.GolfWithAccessKeys {
		var config GolfWithAccessCourseConfig
		var exists bool
		config, exists = GolfWithAccessCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c GolfWithAccessCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchGolfWithAccess(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch CourseRev fetches for this metro
	for _, key := range metro.CourseRevKeys {
		var config CourseRevCourseConfig
		var exists bool
		config, exists = CourseRevCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c CourseRevCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchCourseRev(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch RGuest fetches for this metro
	for _, key := range metro.RGuestKeys {
		var config RGuestCourseConfig
		var exists bool
		config, exists = RGuestCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c RGuestCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchRGuest(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch CourseCo fetches for this metro
	for _, key := range metro.CourseCoKeys {
		var config CourseCoCourseConfig
		var exists bool
		config, exists = CourseCoCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c CourseCoCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchCourseCo(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Launch TeeSnap fetches for this metro
	for _, key := range metro.TeeSnapKeys {
		var config TeeSnapCourseConfig
		var exists bool
		config, exists = TeeSnapCourses[key]
		if !exists {
			continue
		}
		wg.Add(1)
		go func(n string, c TeeSnapCourseConfig) {
			defer wg.Done()
			var results []DisplayTeeTime
			var err error
			results, err = fetchTeeSnap(c, date)
			ch <- fetchResult{results: results, err: err, name: n}
		}(key, config)
	}

	// Close channel when all goroutines finish
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results
	var allResults []DisplayTeeTime
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allResults)
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
	var alerts []Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var incoming Alert
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

	var alert Alert
	alert, err = addAlert(incoming.Phone, incoming.Course, incoming.Date, incoming.StartTime, incoming.EndTime)
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
