package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"time"
)

func handleTeeTimes(w http.ResponseWriter, r *http.Request) {
	var date string = r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	var allResults []DisplayTeeTime
	var err error

	// Fetch all MemberSports courses
	for name, config := range MemberSportsCourses {
		var results []DisplayTeeTime
		results, err = fetchMemberSports(config, date)
		if err != nil {
			fmt.Println("Error fetching", name, ":", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	// Fetch all Chronogolf courses
	for name, config := range ChronogolfCourses {
		var results []DisplayTeeTime
		results, err = fetchChronogolf(config, date)
		if err != nil {
			fmt.Println("Error fetching", name, ":", err)
		} else {
			allResults = append(allResults, results...)
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

func handleHome(w http.ResponseWriter, r *http.Request) {
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/home.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var today string = time.Now().Format("2006-01-02")
	tmpl.Execute(w, today)
}

func handleAlertsPage(w http.ResponseWriter, r *http.Request) {
	var tmpl *template.Template
	var err error
	tmpl, err = template.ParseFiles("templates/alerts.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var today string = time.Now().Format("2006-01-02")
	tmpl.Execute(w, today)
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
