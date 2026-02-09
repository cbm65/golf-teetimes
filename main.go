package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/privacy", handlePrivacy)
	http.HandleFunc("/terms", handleTerms)
	http.HandleFunc("/api/alerts", handleGetAlerts)
	http.HandleFunc("/api/alerts/create", handleCreateAlert)
	http.HandleFunc("/api/alerts/delete", handleDeleteAlert)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", handleRouting)

	go startAlertChecker()

	fmt.Println("Server running at http://localhost:8080")
	var err error = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server failed:", err)
	}
}

func handleRouting(w http.ResponseWriter, r *http.Request) {
	var path string = strings.Trim(r.URL.Path, "/")

	if path == "" {
		handleLanding(w, r)
		return
	}

	var parts []string = strings.SplitN(path, "/", 2)
	var metro Metro
	var exists bool
	metro, exists = Metros[parts[0]]
	if !exists {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 {
		handleMetroHome(w, r, metro)
	} else if parts[1] == "teetimes" {
		handleMetroTeeTimes(w, r, metro)
	} else if parts[1] == "alerts" {
		handleMetroAlerts(w, r, metro)
	} else {
		http.NotFound(w, r)
	}
}
