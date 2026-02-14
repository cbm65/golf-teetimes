package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Discovery tool for TeeItUp courses in a target region.
//
// Usage: go run discovery/teeitup.go <state> <course name> [course name] ...
//   or: go run discovery/teeitup.go <state> -f <file>
//
// The file should have one course name per line (blank lines and # comments ignored).
//
// Example:
//   go run discovery/teeitup.go GA "Sugar Hill Golf Club" "Bobby Jones Golf Course"
//   go run discovery/teeitup.go AZ -f discovery/phoenix-courses.txt
//
// After finding a facility, it validates by checking 3 dates for actual tee times.
// Only courses with bookable tee times are marked as "confirmed".
// Results are saved to discovery/results/teeitup-{state}-{timestamp}.json

var apiBase = "https://phx-api-be-east-1b.kenna.io"

type Facility struct {
	ID       int       `json:"id"`
	CourseID string    `json:"courseId"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Locality string    `json:"locality"`
	Region   string    `json:"region"`
	Country  string    `json:"country"`
	Location []float64 `json:"location"`
	TimeZone string    `json:"timeZone"`
}

type TeeTimeResponse struct {
	Teetimes []struct {
		Teetime string `json:"teetime"`
	} `json:"teetimes"`
}

type Result struct {
	Input        string    `json:"input"`
	Alias        string    `json:"alias"`
	Status       string    `json:"status"` // "confirmed", "listed_only", "miss", "wrong_state", "error"
	Facility     *Facility `json:"facility,omitempty"`
	Error        string    `json:"error,omitempty"`
	WrongState   string    `json:"wrongState,omitempty"`
	DatesChecked []string  `json:"datesChecked,omitempty"`
	TeeTimes     []int     `json:"teeTimes,omitempty"` // count per date checked
}

func log(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

func toAlias(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, "&", "and")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// probeDates returns the next Wednesday, Saturday, and the Saturday after that.
func probeDates() []string {
	now := time.Now()
	var dates []string

	// Find next Wednesday (weekday = 3)
	d := now
	for d.Weekday() != time.Wednesday {
		d = d.AddDate(0, 0, 1)
	}
	dates = append(dates, d.Format("2006-01-02"))

	// Find next Saturday (weekday = 6)
	d = now
	for d.Weekday() != time.Saturday {
		d = d.AddDate(0, 0, 1)
	}
	dates = append(dates, d.Format("2006-01-02"))

	// Saturday after that
	dates = append(dates, d.AddDate(0, 0, 7).Format("2006-01-02"))

	return dates
}

func probeFacility(alias string) ([]Facility, int, error) {
	url := apiBase + "/facilities"
	log("  HTTP GET %s", url)
	log("  Headers: x-be-alias=%s", alias)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("x-be-alias", alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+alias+".book.teeitup.com")

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	log("  Response: %d (%dms, %d bytes)", resp.StatusCode, elapsed.Milliseconds(), len(body))

	if resp.StatusCode != 200 {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log("  Body: %s", preview)
		return nil, resp.StatusCode, nil
	}

	var facilities []Facility
	if err := json.Unmarshal(body, &facilities); err != nil {
		log("  JSON parse error: %v", err)
		return nil, resp.StatusCode, nil
	}

	log("  Parsed %d facility(ies)", len(facilities))
	for i, f := range facilities {
		log("    [%d] FID:%d  %s  (%s, %s)  tz:%s", i, f.ID, f.Name, f.Locality, f.Region, f.TimeZone)
	}

	return facilities, resp.StatusCode, nil
}

func probeTeeTimes(alias string, facilityID int, date string) (int, error) {
	url := fmt.Sprintf("%s/v2/tee-times?date=%s&facilityIds=%d&dateMax=%s", apiBase, date, facilityID, date)
	log("    Checking tee times: %s (FID:%d)", date, facilityID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-be-alias", alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+alias+".book.teeitup.com")

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	log("    Response: %d (%dms, %d bytes)", resp.StatusCode, elapsed.Milliseconds(), len(body))

	if resp.StatusCode != 200 {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log("    Body: %s", preview)
		return 0, nil
	}

	var data []TeeTimeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		log("    JSON parse error: %v", err)
		return 0, nil
	}

	total := 0
	for _, d := range data {
		total += len(d.Teetimes)
	}

	log("    Found %d tee times for %s", total, date)
	return total, nil
}

func readNamesFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Support "Course Name | City | Notes" format — take first field
		if idx := strings.Index(line, "|"); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			names = append(names, line)
		}
	}
	return names, scanner.Err()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run discovery/teeitup.go <state> <course name> [course name] ...\n")
		fmt.Fprintf(os.Stderr, "   or: go run discovery/teeitup.go <state> -f <file>\n")
		fmt.Fprintf(os.Stderr, "Example: go run discovery/teeitup.go GA \"Sugar Hill Golf Club\" \"Bobby Jones Golf Course\"\n")
		fmt.Fprintf(os.Stderr, "         go run discovery/teeitup.go AZ -f discovery/phoenix-courses.txt\n")
		os.Exit(1)
	}

	state := strings.ToUpper(os.Args[1])
	var names []string
	if os.Args[2] == "-f" {
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Missing file path after -f\n")
			os.Exit(1)
		}
		var err error
		names, err = readNamesFromFile(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	} else {
		names = os.Args[2:]
	}
	startTime := time.Now()
	dates := probeDates()

	log("=== TeeItUp Discovery ===")
	log("Target state: %s", state)
	log("Courses to probe: %d", len(names))
	log("API base: %s", apiBase)
	log("Validation dates: %s, %s, %s", dates[0], dates[1], dates[2])
	log("")

	var results []Result
	var confirmedCount, listedOnlyCount, missCount, errorCount, wrongStateCount int

	for i, name := range names {
		alias := toAlias(name)
		log("[%d/%d] %q -> alias: %q", i+1, len(names), name, alias)

		facilities, statusCode, err := probeFacility(alias)
		if err != nil {
			log("  ERROR: %v", err)
			results = append(results, Result{Input: name, Alias: alias, Status: "error", Error: err.Error()})
			errorCount++
			log("")
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if statusCode != 200 || facilities == nil || len(facilities) == 0 {
			log("  MISS (status=%d, facilities=%d)", statusCode, len(facilities))
			results = append(results, Result{Input: name, Alias: alias, Status: "miss"})
			missCount++
			log("")
			time.Sleep(200 * time.Millisecond)
			continue
		}

		matched := false
		for _, f := range facilities {
			if strings.EqualFold(f.Region, state) {
				log("  FACILITY FOUND — FID:%d  %s (%s, %s)", f.ID, f.Name, f.Locality, f.Region)
				log("  Validating with tee time checks...")

				// Probe 3 dates for actual tee times
				fCopy := f
				var datesChecked []string
				var teeTimes []int
				totalTimes := 0

				for _, date := range dates {
					count, err := probeTeeTimes(alias, f.ID, date)
					if err != nil {
						log("    ERROR checking %s: %v", date, err)
						count = 0
					}
					datesChecked = append(datesChecked, date)
					teeTimes = append(teeTimes, count)
					totalTimes += count
					time.Sleep(200 * time.Millisecond)
				}

				if totalTimes > 0 {
					log("  CONFIRMED — %d total tee times across %d dates", totalTimes, len(dates))
					results = append(results, Result{
						Input: name, Alias: alias, Status: "confirmed",
						Facility: &fCopy, DatesChecked: datesChecked, TeeTimes: teeTimes,
					})
					confirmedCount++
				} else {
					log("  LISTED ONLY — facility exists but 0 tee times across all dates")
					results = append(results, Result{
						Input: name, Alias: alias, Status: "listed_only",
						Facility: &fCopy, DatesChecked: datesChecked, TeeTimes: teeTimes,
					})
					listedOnlyCount++
				}
				matched = true
			}
		}
		if !matched {
			log("  WRONG STATE — got %s, wanted %s", facilities[0].Region, state)
			results = append(results, Result{
				Input: name, Alias: alias, Status: "wrong_state",
				WrongState: facilities[0].Region,
			})
			wrongStateCount++
		}

		log("")
		time.Sleep(200 * time.Millisecond)
	}

	elapsed := time.Since(startTime)

	// Summary
	log("========================================")
	log("=== SUMMARY")
	log("========================================")
	log("Total probed:  %d", len(names))
	log("Confirmed:     %d  (facility + tee times)", confirmedCount)
	log("Listed only:   %d  (facility but no tee times)", listedOnlyCount)
	log("Misses:        %d", missCount)
	log("Wrong state:   %d", wrongStateCount)
	log("Errors:        %d", errorCount)
	log("Elapsed:       %s", elapsed.Round(time.Millisecond))
	log("")

	if confirmedCount > 0 {
		log("=== CONFIRMED (ready to add) ===")
		for _, r := range results {
			if r.Status == "confirmed" {
				total := 0
				for _, t := range r.TeeTimes {
					total += t
				}
				log("  %-40s alias:%-35s FID:%-6d %s (%s, %s)  [%d times across %d dates]",
					r.Input, r.Alias, r.Facility.ID, r.Facility.Name, r.Facility.Locality, r.Facility.Region,
					total, len(r.DatesChecked))
			}
		}
		log("")
	}

	if listedOnlyCount > 0 {
		log("=== LISTED ONLY (facility exists, no tee times — likely not using TeeItUp for booking) ===")
		for _, r := range results {
			if r.Status == "listed_only" {
				log("  %-40s alias:%-35s FID:%-6d %s (%s, %s)",
					r.Input, r.Alias, r.Facility.ID, r.Facility.Name, r.Facility.Locality, r.Facility.Region)
			}
		}
		log("")
	}

	if missCount > 0 {
		log("=== MISSES ===")
		for _, r := range results {
			if r.Status == "miss" {
				log("  %-40s alias: %s", r.Input, r.Alias)
			}
		}
		log("")
	}

	if wrongStateCount > 0 {
		log("=== WRONG STATE ===")
		for _, r := range results {
			if r.Status == "wrong_state" {
				log("  %-40s alias: %-30s got: %s", r.Input, r.Alias, r.WrongState)
			}
		}
		log("")
	}

	if errorCount > 0 {
		log("=== ERRORS ===")
		for _, r := range results {
			if r.Status == "error" {
				log("  %-40s %s", r.Input, r.Error)
			}
		}
		log("")
	}

	// Save results to file
	os.MkdirAll("discovery/results", 0755)
	filename := fmt.Sprintf("discovery/results/teeitup-%s-%s.json",
		strings.ToLower(state), startTime.Format("2006-01-02-150405"))

	output := map[string]any{
		"platform":       "teeitup",
		"state":          state,
		"timestamp":      startTime.Format(time.RFC3339),
		"elapsed":        elapsed.String(),
		"totalInput":     len(names),
		"confirmed":      confirmedCount,
		"listedOnly":     listedOnlyCount,
		"misses":         missCount,
		"wrongState":     wrongStateCount,
		"errors":         errorCount,
		"validationDates": dates,
		"results":        results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log("ERROR saving results: %v", err)
		return
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log("ERROR writing file: %v", err)
		return
	}
	log("Results saved to %s", filename)
}
