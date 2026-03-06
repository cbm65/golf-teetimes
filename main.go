package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	http.HandleFunc("/privacy", handlePrivacy)
	http.HandleFunc("/terms", handleTerms)
	http.HandleFunc("/api/alerts", handleGetAlerts)
	http.HandleFunc("/api/alerts/create", handleCreateAlert)
	http.HandleFunc("/api/alerts/delete", handleDeleteAlert)
	http.HandleFunc("/optin", handleOptIn)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", handleRouting)

	go startAlertChecker()

	srv := &http.Server{Addr: ":8080"}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	fmt.Println("Server running at http://localhost:8080")
	var err error = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Println("Server failed:", err)
	}
}

func handleRouting(w http.ResponseWriter, r *http.Request) {
	var path string = strings.Trim(r.URL.Path, "/")

	if path == "" {
		handleLanding(w, r)
		return
	}

	if strings.HasPrefix(path, "railway-verify=") {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(path))
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
