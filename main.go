package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/teetimes", handleTeeTimes)
	http.HandleFunc("/alerts", handleAlertsPage)
	http.HandleFunc("/api/alerts", handleGetAlerts)
	http.HandleFunc("/api/alerts/create", handleCreateAlert)
	http.HandleFunc("/api/alerts/delete", handleDeleteAlert)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	go startAlertChecker()

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
