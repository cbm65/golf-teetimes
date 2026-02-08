package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const AlertsFile = "data/alerts.json"

func parseTimeToMinutes(timeStr string) int {
	var parts []string = strings.Split(timeStr, " ")
	if len(parts) != 2 {
		return 0
	}
	var period string = parts[1]
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

func loadAlerts() ([]Alert, error) {
	var data []byte
	var err error
	data, err = os.ReadFile(AlertsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Alert{}, nil
		}
		return nil, err
	}

	var alerts []Alert
	err = json.Unmarshal(data, &alerts)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func saveAlerts(alerts []Alert) error {
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

func addAlert(phone string, course string, date string, startTime string, endTime string) (Alert, error) {
	// Validate start time is before end time
	var startMins int = parseTimeToMinutes(startTime)
	var endMins int = parseTimeToMinutes(endTime)
	if startMins >= endMins {
		return Alert{}, errors.New("Start time must be before end time.")
	}

	// Check for duplicate alert (same phone + course + date)
	var alerts []Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		return Alert{}, err
	}

	for _, existing := range alerts {
		if existing.Phone == phone && existing.Course == course && existing.Date == date && existing.Active {
			return Alert{}, errors.New("You already have an alert set for " + course + " on " + date + ". Delete it first to create a new one.")
		}
	}

	// Check if a tee time already exists in this window
	var teeTimes []DisplayTeeTime
	teeTimes, err = fetchDenver(date)
	if err == nil {
		for _, tt := range teeTimes {
			var baseCourse string = tt.Course
			if strings.HasPrefix(baseCourse, "Kennedy") {
				baseCourse = "Kennedy"
			}
			baseCourse = strings.Replace(baseCourse, " Back Nine", "", 1)

			if baseCourse == course && tt.Openings > 0 {
				var ttMins int = parseTimeToMinutes(tt.Time)
				if ttMins >= startMins && ttMins <= endMins {
					return Alert{}, errors.New("There's already a tee time available at " + course + " at " + tt.Time + " — go book it!")
				}
			}
		}
	}

	var alert Alert = Alert{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Phone:     phone,
		Course:    course,
		Date:      date,
		StartTime: startTime,
		EndTime:   endTime,
		Active:    true,
		CreatedAt: time.Now().Format("2006-01-02 3:04 PM"),
	}

	alerts = append(alerts, alert)

	err = saveAlerts(alerts)
	if err != nil {
		return Alert{}, err
	}

	return alert, nil
}

func deleteAlert(id string) error {
	var alerts []Alert
	var err error
	alerts, err = loadAlerts()
	if err != nil {
		return err
	}

	var updated []Alert
	for _, alert := range alerts {
		if alert.ID != id {
			updated = append(updated, alert)
		}
	}

	return saveAlerts(updated)
}

func startAlertChecker() {
	fmt.Println("Alert checker started — checking every 1 minute")

	for {
		var alerts []Alert
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

		for _, alert := range alerts {
			if !alert.Active {
				continue
			}

			fmt.Println("")
			fmt.Println("  Checking:", alert.Course, "|", alert.Date, "|", alert.StartTime, "–", alert.EndTime, "|", alert.Phone)

			var teeTimes []DisplayTeeTime
			teeTimes, err = fetchDenver(alert.Date)
			if err != nil {
				fmt.Println("    [ERROR] Fetching tee times:", err)
				continue
			}

			fmt.Println("    Fetched", len(teeTimes), "tee times for", alert.Date)

			var startMins int = parseTimeToMinutes(alert.StartTime)
			var endMins int = parseTimeToMinutes(alert.EndTime)
			var matchCount int = 0

			for _, tt := range teeTimes {
				var baseCourse string = tt.Course
				if strings.HasPrefix(baseCourse, "Kennedy") {
					baseCourse = "Kennedy"
				}
				baseCourse = strings.Replace(baseCourse, " Back Nine", "", 1)

				if baseCourse != alert.Course {
					continue
				}

				if tt.Openings <= 0 {
					fmt.Println("    ✗", tt.Time, tt.Course, "— no openings")
					continue
				}

				var ttMins int = parseTimeToMinutes(tt.Time)
				if ttMins < startMins || ttMins > endMins {
					fmt.Println("    ✗", tt.Time, tt.Course, "— outside time window")
					continue
				}

				matchCount++
				fmt.Println("    ✓ MATCH!", tt.Time, tt.Course, "—", tt.Openings, "openings, $", tt.Price)
				// TODO: send SMS here
			}

			if matchCount == 0 {
				fmt.Println("    No matches found")
			} else {
				fmt.Println("   ", matchCount, "match(es) found!")
			}
		}

		fmt.Println("")
		fmt.Println("  Next check in 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}
