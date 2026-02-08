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

func findChronogolfConfig(course string) (ChronogolfCourseConfig, bool) {
	for _, config := range ChronogolfCourses {
		for _, displayName := range config.Names {
			if displayName == course {
				return config, true
			}
		}
	}
	return ChronogolfCourseConfig{}, false
}

func findMemberSportsConfig(course string) (MemberSportsCourseConfig, bool) {
	for _, config := range MemberSportsCourses {
		for _, known := range config.KnownCourses {
			if known == course {
				return config, true
			}
		}
	}
	return MemberSportsCourseConfig{}, false
}

func fetchForCourse(course string, date string) ([]DisplayTeeTime, error) {
	var cgConfig ChronogolfCourseConfig
	var cgFound bool
	cgConfig, cgFound = findChronogolfConfig(course)
	if cgFound {
		return fetchChronogolf(cgConfig, date)
	}

	var msConfig MemberSportsCourseConfig
	var msFound bool
	msConfig, msFound = findMemberSportsConfig(course)
	if msFound {
		return fetchMemberSports(msConfig, date)
	}

	// Default to Denver
	return fetchMemberSports(MemberSportsCourses["denver"], date)
}

func bookingURLForCourse(course string) string {
	var cgConfig ChronogolfCourseConfig
	var cgFound bool
	cgConfig, cgFound = findChronogolfConfig(course)
	if cgFound {
		return cgConfig.BookingURL
	}

	var msConfig MemberSportsCourseConfig
	var msFound bool
	msConfig, msFound = findMemberSportsConfig(course)
	if msFound {
		return msConfig.BookingURL
	}

	return MemberSportsCourses["denver"].BookingURL
}

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

func getBaseCourse(name string) string {
	if strings.HasPrefix(name, "Kennedy") {
		return "Kennedy"
	}
	if strings.HasPrefix(name, "Fox Hollow") {
		return "Fox Hollow"
	}
	if strings.HasPrefix(name, "Homestead") {
		return "Homestead"
	}
	return strings.Replace(name, " Back Nine", "", 1)
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
	teeTimes, err = fetchForCourse(course, date)
	if err == nil {
		for _, tt := range teeTimes {
			var baseCourse string = getBaseCourse(tt.Course)

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

type MatchedTeeTime struct {
	Time     string
	Openings int
	Price    float64
}

func buildAlertMessage(course string, date string, matches []MatchedTeeTime) string {
	var bookURL string = bookingURLForCourse(course)
	var msg string = "⛳ Tee time alert! " + course + " on " + date + ":\n"

	for _, m := range matches {
		msg += fmt.Sprintf("%s (%d openings) - $%.0f\n", m.Time, m.Openings, m.Price)
	}

	msg += "\nBook now: " + bookURL
	msg += "\nReply STOP to unsubscribe"

	return msg
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
			teeTimes, err = fetchForCourse(alert.Course, alert.Date)
			if err != nil {
				fmt.Println("    [ERROR] Fetching tee times:", err)
				continue
			}

			fmt.Println("    Fetched", len(teeTimes), "tee times for", alert.Date)

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
				})
			}

			if len(matches) == 0 {
				fmt.Println("    No matches found")
			} else {
				var msg string = buildAlertMessage(alert.Course, alert.Date, matches)
				fmt.Println("   ", len(matches), "match(es) found!")
				fmt.Println("    SMS would be:")
				fmt.Println("    ────────────")
				fmt.Println("   ", msg)
				fmt.Println("    ────────────")
				// TODO: send SMS via Twilio here
			}
		}

		fmt.Println("")
		fmt.Println("  Next check in 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}
