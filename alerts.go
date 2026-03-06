package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"golf-teetimes/platforms"
)

const AlertsFile = "alerts.json"

var alertFileMu sync.Mutex

func sendSMS(to string, message string) error {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	fromNumber := os.Getenv("TWILIO_FROM_NUMBER")
	if accountSid == "" || authToken == "" || fromNumber == "" {
		return fmt.Errorf("TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, or TWILIO_FROM_NUMBER not set")
	}

	data := url.Values{}
	data.Set("From", fromNumber)
	data.Set("To", to)
	data.Set("Body", message)
	apiURL := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(accountSid, authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func metroForCourse(course string) string {
	if c, ok := platforms.FindCourse(course); ok {
		return c.Metro
	}
	return ""
}

func fetchForCourse(course string, date string) ([]platforms.DisplayTeeTime, error) {
	if c, ok := platforms.FindCourse(course); ok {
		return c.Fetch(date)
	}
	// Default to Denver
	return platforms.FetchMemberSports(platforms.MemberSportsCourses["denver"], date)
}

func bookingURLForCourse(course string) string {
	if c, ok := platforms.FindCourse(course); ok {
		return c.BookingURL
	}
	return platforms.MemberSportsCourses["denver"].BookingURL
}

func parseTimeToMinutes(timeStr string) int {
	var parts []string = strings.Split(timeStr, " ")
	if len(parts) != 2 {
		return 0
	}
	var period string = strings.ToUpper(parts[1])
	var timeParts []string = strings.Split(parts[0], ":")
	if len(timeParts) != 2 {
		return 0
	}

	var hours int
	var mins int
	fmt.Sscanf(timeParts[0], "%d", &hours)
	fmt.Sscanf(timeParts[1], "%d", &mins)

	if period == "PM" && hours != 12 {
		hours = hours + 12
	}
	if period == "AM" && hours == 12 {
		hours = 0
	}

	return hours*60 + mins
}

func loadAlerts() ([]platforms.Alert, error) {
	var data []byte
	var err error
	data, err = os.ReadFile(AlertsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []platforms.Alert{}, nil
		}
		return nil, err
	}

	var alerts []platforms.Alert
	err = json.Unmarshal(data, &alerts)
	if err != nil {
		return nil, err
	}

	// Prune alerts whose date has passed
	var today string = time.Now().Format("2006-01-02")
	var pruned bool
	var current []platforms.Alert
	for _, a := range alerts {
		if a.Date < today {
			pruned = true
			continue
		}
		current = append(current, a)
	}
	if pruned {
		if current == nil {
			current = []platforms.Alert{}
		}
		saveAlerts(current)
		return current, nil
	}

	return alerts, nil
}

func saveAlerts(alerts []platforms.Alert) error {
	var data []byte
	var err error
	data, err = json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return err
	}

	err = os.MkdirAll("data", 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(AlertsFile, data, 0644)
}

func getBaseCourse(name string) string {
	return platforms.GetBaseCourse(name)
}

func addAlert(phone string, course string, date string, startTime string, endTime string, minPlayers int, holes string) (platforms.Alert, error) {
	// Validate start time is before end time
	var startMins int = parseTimeToMinutes(startTime)
	var endMins int = parseTimeToMinutes(endTime)
	if startMins >= endMins {
		return platforms.Alert{}, errors.New("Start time must be before end time.")
	}

	// Check if a tee time already exists in this window (before taking lock, read-only external fetch)
	var teeTimes []platforms.DisplayTeeTime
	teeTimes, _ = fetchForCourse(course, date)
	for _, tt := range teeTimes {
		var baseCourse string = getBaseCourse(tt.Course)

		if baseCourse == course && tt.Openings > 0 {
			if minPlayers > 0 && tt.Openings < minPlayers {
				continue
			}
			if holes != "" && holes != "0" && tt.Holes != "" && tt.Holes != holes {
				continue
			}
			var ttMins int = parseTimeToMinutes(tt.Time)
			if ttMins >= startMins && ttMins <= endMins {
				return platforms.Alert{}, errors.New("There's already a tee time available at " + course + " at " + tt.Time + " — go book it!")
			}
		}
	}

	alertFileMu.Lock()
	defer alertFileMu.Unlock()

	// Check for duplicate alert (same phone + course + date)
	var alerts []platforms.Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		return platforms.Alert{}, err
	}

	for _, existing := range alerts {
		if existing.Phone == phone && existing.Course == course && existing.Date == date && existing.Active {
			return platforms.Alert{}, errors.New("You already have an alert set for " + course + " on " + date + ". Delete it first to create a new one.")
		}
	}

	var alert platforms.Alert = platforms.Alert{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		Phone:      phone,
		Course:     course,
		Date:       date,
		StartTime:  startTime,
		EndTime:    endTime,
		MinPlayers: minPlayers,
		Holes:      holes,
		Active:     true,
		CreatedAt:  time.Now().Format("2006-01-02 3:04 PM"),
		ConsentAt:  time.Now().Format("2006-01-02 3:04:05 PM MST"),
	}

	alerts = append(alerts, alert)

	err = saveAlerts(alerts)
	if err != nil {
		return platforms.Alert{}, err
	}

	return alert, nil
}

func deleteAlertByOwner(id string, phone string) error {
	alertFileMu.Lock()
	defer alertFileMu.Unlock()

	var alerts []platforms.Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		return err
	}

	var found bool
	var updated []platforms.Alert
	for _, alert := range alerts {
		if alert.ID == id {
			if alert.Phone != phone {
				return errors.New("Not authorized to delete this alert")
			}
			found = true
			continue
		}
		updated = append(updated, alert)
	}

	if !found {
		return errors.New("Alert not found")
	}

	return saveAlerts(updated)
}

func deleteAlert(id string) error {
	alertFileMu.Lock()
	defer alertFileMu.Unlock()

	var alerts []platforms.Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		return err
	}

	var updated []platforms.Alert
	for _, alert := range alerts {
		if alert.ID != id {
			updated = append(updated, alert)
		}
	}

	return saveAlerts(updated)
}

type MatchedTeeTime struct {
	Time     string
	Openings int
	Price    float64
	Holes    string
}

func buildAlertMessage(course string, date string, matches []MatchedTeeTime) string {
	var bookURL string = bookingURLForCourse(course)
	var msg string = "⛳ Tee time alert! " + course + " on " + date + ":\n"

	for _, m := range matches {
		msg += fmt.Sprintf("%s (%d openings, %s holes) - $%.0f\n", m.Time, m.Openings, m.Holes, m.Price)
	}

	msg += "\nBook now: " + bookURL
	msg += "\nReply STOP to unsubscribe"

	return msg
}

func startAlertChecker() {
	fmt.Println("platforms.Alert checker started — checking every 1 minute")

	for {
		var alerts []platforms.Alert
		var err error
		alerts, err = loadAlerts()
		if err != nil {
			fmt.Println("  [ERROR] Loading alerts:", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		var activeCount int = 0
		for _, alert := range alerts {
			if alert.Active {
				activeCount++
			}
		}

		fmt.Println("")
		fmt.Println("──────────────────────────────────────")
		fmt.Println("  Checking alerts at", time.Now().Format("3:04:05 PM"))
		fmt.Println("  Active alerts:", activeCount)
		fmt.Println("──────────────────────────────────────")

		if activeCount == 0 {
			fmt.Println("  No active alerts — sleeping")
			time.Sleep(1 * time.Minute)
			continue
		}

		// Group active alerts by metro:date for batched fetching
		type alertRef struct {
			index int
			alert platforms.Alert
			metro string
		}
		groups := make(map[string][]alertRef) // key = metro:date

		for i, alert := range alerts {
			if !alert.Active {
				continue
			}
			metro := metroForCourse(alert.Course)
			if metro == "" {
				// Unknown course — skip
				fmt.Println("  [WARN] No metro found for course:", alert.Course)
				continue
			}
			key := metro + ":" + alert.Date
			groups[key] = append(groups[key], alertRef{index: i, alert: alert, metro: metro})
		}

		var dirty bool // track if any alerts were deactivated

		for groupKey, refs := range groups {
			// Parse metro and date from group key
			var metro string
			var date string
			for j := range groupKey {
				if groupKey[j] == ':' {
					metro = groupKey[:j]
					date = groupKey[j+1:]
					break
				}
			}

			metroObj, metroExists := Metros[metro]
			if !metroExists {
				fmt.Println("  [WARN] Unknown metro slug:", metro)
				continue
			}

			// One fetch for all alerts in this metro+date group
			teeTimes := fetchMetroTeeTimes(metroObj, date)
			fmt.Println("")
			fmt.Printf("  Fetched %d tee times for %s on %s (%d alerts)\n", len(teeTimes), metro, date, len(refs))

			for _, ref := range refs {
				alert := ref.alert
				fmt.Println("")
				fmt.Println("  Checking:", alert.Course, "|", alert.Date, "|", alert.StartTime, "–", alert.EndTime)

				var startMins int = parseTimeToMinutes(alert.StartTime)
				var endMins int = parseTimeToMinutes(alert.EndTime)
				var matches []MatchedTeeTime

				for _, tt := range teeTimes {
					var baseCourse string = getBaseCourse(tt.Course)

					if baseCourse != alert.Course {
						continue
					}

					if tt.Openings <= 0 {
						fmt.Println("    ✗", tt.Time, tt.Course, "— no openings")
						continue
					}

					if alert.MinPlayers > 0 && tt.Openings < alert.MinPlayers {
						fmt.Println("    ✗", tt.Time, tt.Course, "— only", tt.Openings, "openings, need", alert.MinPlayers)
						continue
					}

					if alert.Holes != "" && alert.Holes != "0" && tt.Holes != "" && tt.Holes != alert.Holes {
						fmt.Println("    ✗", tt.Time, tt.Course, "—", tt.Holes, "holes, want", alert.Holes)
						continue
					}

					var ttMins int = parseTimeToMinutes(tt.Time)
					if ttMins < startMins || ttMins > endMins {
						fmt.Println("    ✗", tt.Time, tt.Course, "— outside time window")
						continue
					}

					fmt.Println("    ✓ MATCH!", tt.Time, tt.Course, "—", tt.Openings, "openings, $", tt.Price)
					matches = append(matches, MatchedTeeTime{
						Time:     tt.Time,
						Openings: tt.Openings,
						Price:    tt.Price,
						Holes:    tt.Holes,
					})
				}

				if len(matches) == 0 {
					fmt.Println("    No matches found")
				} else {
					var msg string = buildAlertMessage(alert.Course, alert.Date, matches)
					fmt.Println("   ", len(matches), "match(es) found!")
					fmt.Println("    Sending SMS for alert", alert.ID)
					var smsErr error = sendSMS(alert.Phone, msg)
					if smsErr != nil {
						fmt.Println("    [ERROR] SMS failed:", smsErr)
					} else {
						fmt.Println("    ✓ SMS sent successfully")
						alerts[ref.index].Active = false
						dirty = true
					}
				}
			}
		}

		if dirty {
			alertFileMu.Lock()
			saveAlerts(alerts)
			alertFileMu.Unlock()
		}

		fmt.Println("")
		fmt.Println("  Next check in 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}
