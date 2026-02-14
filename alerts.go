package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"golf-teetimes/platforms"
)

const AlertsFile = "alerts.json"

func sendSMS(to string, message string) error {
	apiKey := os.Getenv("TELNYX_API_KEY")
	fromNumber := os.Getenv("TELNYX_FROM_NUMBER")
	if apiKey == "" || fromNumber == "" {
		return fmt.Errorf("TELNYX_API_KEY or TELNYX_FROM_NUMBER not set")
	}

	payload, _ := json.Marshal(map[string]string{
		"from": fromNumber,
		"to":   to,
		"text": message,
	})

	req, err := http.NewRequest("POST", "https://api.telnyx.com/v2/messages", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telnyx API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func findChronogolfConfig(course string) (platforms.ChronogolfCourseConfig, bool) {
	for _, config := range platforms.ChronogolfCourses {
		for _, displayName := range config.Names {
			if displayName == course {
				return config, true
			}
		}
	}
	return platforms.ChronogolfCourseConfig{}, false
}

func findMemberSportsConfig(course string) (platforms.MemberSportsCourseConfig, bool) {
	for _, config := range platforms.MemberSportsCourses {
		for _, known := range config.KnownCourses {
			if known == course {
				return config, true
			}
		}
	}
	return platforms.MemberSportsCourseConfig{}, false
}

func findCPSGolfConfig(course string) (platforms.CPSGolfCourseConfig, bool) {
	for _, config := range platforms.CPSGolfCourses {
		for _, displayName := range config.Names {
			if displayName == course {
				return config, true
			}
		}
	}
	return platforms.CPSGolfCourseConfig{}, false
}

func findClubCaddieConfig(course string) (platforms.ClubCaddieCourseConfig, bool) {
	for _, config := range platforms.ClubCaddieCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.ClubCaddieCourseConfig{}, false
}

func findTeeItUpConfig(course string) (platforms.TeeItUpCourseConfig, bool) {
	for _, config := range platforms.TeeItUpCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.TeeItUpCourseConfig{}, false
}

func findGolfNowConfig(course string) (platforms.GolfNowCourseConfig, bool) {
	for _, config := range platforms.GolfNowCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.GolfNowCourseConfig{}, false
}

func findQuick18Config(course string) (platforms.Quick18CourseConfig, bool) {
	for _, config := range platforms.Quick18Courses {
		if config.DisplayName == course {
			return config, true
		}
		if config.NamePrefix != "" && strings.HasPrefix(course, config.NamePrefix) {
			return config, true
		}
	}
	return platforms.Quick18CourseConfig{}, false
}

func findGolfWithAccessConfig(course string) (platforms.GolfWithAccessCourseConfig, bool) {
	for _, config := range platforms.GolfWithAccessCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.GolfWithAccessCourseConfig{}, false
}

func findCourseRevConfig(course string) (platforms.CourseRevCourseConfig, bool) {
	for _, config := range platforms.CourseRevCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.CourseRevCourseConfig{}, false
}

func findRGuestConfig(course string) (platforms.RGuestCourseConfig, bool) {
	for _, config := range platforms.RGuestCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.RGuestCourseConfig{}, false
}

func findCourseCoConfig(course string) (platforms.CourseCoCourseConfig, bool) {
	for _, config := range platforms.CourseCoCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.CourseCoCourseConfig{}, false
}

func findTeeSnapConfig(course string) (platforms.TeeSnapCourseConfig, bool) {
	for _, config := range platforms.TeeSnapCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.TeeSnapCourseConfig{}, false
}

func findForeUpConfig(course string) (platforms.ForeUpCourseConfig, bool) {
	for _, config := range platforms.ForeUpCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.ForeUpCourseConfig{}, false
}

func fetchForCourse(course string, date string) ([]platforms.DisplayTeeTime, error) {
	var cgConfig platforms.ChronogolfCourseConfig
	var cgFound bool
	cgConfig, cgFound = findChronogolfConfig(course)
	if cgFound {
		return platforms.FetchChronogolf(cgConfig, date)
	}

	var msConfig platforms.MemberSportsCourseConfig
	var msFound bool
	msConfig, msFound = findMemberSportsConfig(course)
	if msFound {
		return platforms.FetchMemberSports(msConfig, date)
	}

	var cpsConfig platforms.CPSGolfCourseConfig
	var cpsFound bool
	cpsConfig, cpsFound = findCPSGolfConfig(course)
	if cpsFound {
		return platforms.FetchCPSGolf(cpsConfig, date)
	}

	var gnConfig platforms.GolfNowCourseConfig
	var gnFound bool
	gnConfig, gnFound = findGolfNowConfig(course)
	if gnFound {
		return platforms.FetchGolfNow(gnConfig, date)
	}

	var tiuConfig platforms.TeeItUpCourseConfig
	var tiuFound bool
	tiuConfig, tiuFound = findTeeItUpConfig(course)
	if tiuFound {
		return platforms.FetchTeeItUp(tiuConfig, date)
	}

	var ccConfig platforms.ClubCaddieCourseConfig
	var ccFound bool
	ccConfig, ccFound = findClubCaddieConfig(course)
	if ccFound {
		return platforms.FetchClubCaddie(ccConfig, date)
	}

	var q18Config platforms.Quick18CourseConfig
	var q18Found bool
	q18Config, q18Found = findQuick18Config(course)
	if q18Found {
		return platforms.FetchQuick18(q18Config, date)
	}

	var gwaConfig platforms.GolfWithAccessCourseConfig
	var gwaFound bool
	gwaConfig, gwaFound = findGolfWithAccessConfig(course)
	if gwaFound {
		return platforms.FetchGolfWithAccess(gwaConfig, date)
	}

	var crConfig platforms.CourseRevCourseConfig
	var crFound bool
	crConfig, crFound = findCourseRevConfig(course)
	if crFound {
		return platforms.FetchCourseRev(crConfig, date)
	}

	var rgConfig platforms.RGuestCourseConfig
	var rgFound bool
	rgConfig, rgFound = findRGuestConfig(course)
	if rgFound {
		return platforms.FetchRGuest(rgConfig, date)
	}

	var coConfig platforms.CourseCoCourseConfig
	var coFound bool
	coConfig, coFound = findCourseCoConfig(course)
	if coFound {
		return platforms.FetchCourseCo(coConfig, date)
	}

	var tsConfig platforms.TeeSnapCourseConfig
	var tsFound bool
	tsConfig, tsFound = findTeeSnapConfig(course)
	if tsFound {
		return platforms.FetchTeeSnap(tsConfig, date)
	}

	var fuConfig platforms.ForeUpCourseConfig
	var fuFound bool
	fuConfig, fuFound = findForeUpConfig(course)
	if fuFound {
		return platforms.FetchForeUp(fuConfig, date)
	}

	// Default to Denver
	return platforms.FetchMemberSports(platforms.MemberSportsCourses["denver"], date)
}

func bookingURLForCourse(course string) string {
	var cgConfig platforms.ChronogolfCourseConfig
	var cgFound bool
	cgConfig, cgFound = findChronogolfConfig(course)
	if cgFound {
		return cgConfig.BookingURL
	}

	var msConfig platforms.MemberSportsCourseConfig
	var msFound bool
	msConfig, msFound = findMemberSportsConfig(course)
	if msFound {
		return msConfig.BookingURL
	}

	var cpsConfig platforms.CPSGolfCourseConfig
	var cpsFound bool
	cpsConfig, cpsFound = findCPSGolfConfig(course)
	if cpsFound {
		return cpsConfig.BookingURL
	}

	var gnConfig platforms.GolfNowCourseConfig
	var gnFound bool
	gnConfig, gnFound = findGolfNowConfig(course)
	if gnFound {
		return gnConfig.BookingURL
	}

	var tiuConfig platforms.TeeItUpCourseConfig
	var tiuFound bool
	tiuConfig, tiuFound = findTeeItUpConfig(course)
	if tiuFound {
		return "https://" + tiuConfig.Alias + ".book.teeitup.com/teetimes"
	}

	var ccConfig platforms.ClubCaddieCourseConfig
	var ccFound bool
	ccConfig, ccFound = findClubCaddieConfig(course)
	if ccFound {
		return ccConfig.BookingURL
	}

	var q18Config platforms.Quick18CourseConfig
	var q18Found bool
	q18Config, q18Found = findQuick18Config(course)
	if q18Found {
		return q18Config.BookingURL
	}

	var gwaConfig platforms.GolfWithAccessCourseConfig
	var gwaFound bool
	gwaConfig, gwaFound = findGolfWithAccessConfig(course)
	if gwaFound {
		return gwaConfig.BookingURL
	}

	var crConfig platforms.CourseRevCourseConfig
	var crFound bool
	crConfig, crFound = findCourseRevConfig(course)
	if crFound {
		return crConfig.BookingURL
	}

	var rgConfig platforms.RGuestCourseConfig
	var rgFound bool
	rgConfig, rgFound = findRGuestConfig(course)
	if rgFound {
		return rgConfig.BookingURL
	}

	var coConfig platforms.CourseCoCourseConfig
	var coFound bool
	coConfig, coFound = findCourseCoConfig(course)
	if coFound {
		return coConfig.BookingURL
	}

	var tsConfig platforms.TeeSnapCourseConfig
	var tsFound bool
	tsConfig, tsFound = findTeeSnapConfig(course)
	if tsFound {
		return tsConfig.BookingURL
	}

	var fuConfig platforms.ForeUpCourseConfig
	var fuFound bool
	fuConfig, fuFound = findForeUpConfig(course)
	if fuFound {
		return fuConfig.BookingURL
	}

	return platforms.MemberSportsCourses["denver"].BookingURL
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
	if strings.HasPrefix(name, "Kennedy") {
		return "Kennedy"
	}
	if strings.HasPrefix(name, "Fox Hollow") {
		return "Fox Hollow"
	}
	if strings.HasPrefix(name, "Homestead") {
		return "Homestead"
	}
	if strings.HasPrefix(name, "Harvard Gulch") {
		return "Harvard Gulch"
	}
	if strings.HasPrefix(name, "South Suburban") {
		return "South Suburban"
	}
	if strings.HasPrefix(name, "Foothills") {
		return "Foothills"
	}
	if strings.HasPrefix(name, "Meadows") {
		return "Meadows"
	}
	if strings.HasPrefix(name, "Broken Tee") {
		return "Broken Tee"
	}
	if strings.HasPrefix(name, "Fossil Trace") {
		return "Fossil Trace"
	}
	if strings.HasPrefix(name, "McCormick Ranch") {
		return "McCormick Ranch"
	}
	if strings.HasPrefix(name, "TPC Scottsdale") {
		return "TPC Scottsdale"
	}
	if strings.HasPrefix(name, "Verrado") {
		return "Verrado"
	}
	if strings.HasPrefix(name, "Grayhawk") {
		return "Grayhawk"
	}
	if strings.HasPrefix(name, "Coyote Lakes") {
		return "Coyote Lakes"
	}
	if strings.HasPrefix(name, "Granite Falls") {
		return "Granite Falls"
	}
	if strings.HasPrefix(name, "Wigwam") {
		return "Wigwam"
	}
	if strings.HasPrefix(name, "Troon North") {
		return "Troon North"
	}
	if strings.HasPrefix(name, "Aguila") {
		return "Aguila"
	}
	if strings.HasPrefix(name, "Encanto") {
		return "Encanto"
	}
	if strings.HasPrefix(name, "AZ Biltmore") {
		return "AZ Biltmore"
	}
	if strings.HasPrefix(name, "We-Ko-Pa") {
		return "We-Ko-Pa"
	}
	if strings.HasPrefix(name, "Talking Stick") {
		return "Talking Stick"
	}
	if strings.HasPrefix(name, "Whirlwind") {
		return "Whirlwind"
	}
	if strings.HasPrefix(name, "Wildfire") {
		return "Wildfire"
	}
	if strings.HasPrefix(name, "Camelback") {
		return "Camelback"
	}
	if strings.HasPrefix(name, "Gold Canyon") {
		return "Gold Canyon"
	}
	if strings.HasPrefix(name, "Bear Creek") {
		return "Bear Creek"
	}
	if strings.HasPrefix(name, "Riverdale") {
		return "Riverdale"
	}
	return strings.Replace(name, " Back Nine", "", 1)
}

func addAlert(phone string, course string, date string, startTime string, endTime string) (platforms.Alert, error) {
	// Validate start time is before end time
	var startMins int = parseTimeToMinutes(startTime)
	var endMins int = parseTimeToMinutes(endTime)
	if startMins >= endMins {
		return platforms.Alert{}, errors.New("Start time must be before end time.")
	}

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

	// Check if a tee time already exists in this window
	var teeTimes []platforms.DisplayTeeTime
	teeTimes, err = fetchForCourse(course, date)
	if err == nil {
		for _, tt := range teeTimes {
			var baseCourse string = getBaseCourse(tt.Course)

			if baseCourse == course && tt.Openings > 0 {
				var ttMins int = parseTimeToMinutes(tt.Time)
				if ttMins >= startMins && ttMins <= endMins {
					return platforms.Alert{}, errors.New("There's already a tee time available at " + course + " at " + tt.Time + " — go book it!")
				}
			}
		}
	}

	var alert platforms.Alert = platforms.Alert{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Phone:     phone,
		Course:    course,
		Date:      date,
		StartTime: startTime,
		EndTime:   endTime,
		Active:    true,
		CreatedAt: time.Now().Format("2006-01-02 3:04 PM"),
		ConsentAt: time.Now().Format("2006-01-02 3:04:05 PM MST"),
	}

	alerts = append(alerts, alert)

	err = saveAlerts(alerts)
	if err != nil {
		return platforms.Alert{}, err
	}

	return alert, nil
}

func deleteAlert(id string) error {
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

		for i, alert := range alerts {
			if !alert.Active {
				continue
			}

			fmt.Println("")
			fmt.Println("  Checking:", alert.Course, "|", alert.Date, "|", alert.StartTime, "–", alert.EndTime, "|", alert.Phone)

			var teeTimes []platforms.DisplayTeeTime
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
				fmt.Println("    Sending SMS to", alert.Phone)
				var smsErr error = sendSMS(alert.Phone, msg)
				if smsErr != nil {
					fmt.Println("    [ERROR] SMS failed:", smsErr)
				} else {
					fmt.Println("    ✓ SMS sent successfully")
					// Deactivate alert so we don't send again
					alert.Active = false
					alerts[i] = alert
					saveAlerts(alerts)
				}
			}
		}

		fmt.Println("")
		fmt.Println("  Next check in 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}
