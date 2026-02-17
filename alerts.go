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
	"time"
	"golf-teetimes/platforms"
)

const AlertsFile = "alerts.json"

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

func findProphetConfig(course string) (platforms.ProphetCourseConfig, bool) {
	for _, config := range platforms.ProphetCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.ProphetCourseConfig{}, false
}

func findPurposeGolfConfig(course string) (platforms.PurposeGolfCourseConfig, bool) {
	for _, config := range platforms.PurposeGolfCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.PurposeGolfCourseConfig{}, false
}

func findTeeQuestConfig(course string) (platforms.TeeQuestCourseConfig, bool) {
	for _, config := range platforms.TeeQuestCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.TeeQuestCourseConfig{}, false
}

func findResortSuiteConfig(course string) (platforms.ResortSuiteCourseConfig, bool) {
	for _, config := range platforms.ResortSuiteCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.ResortSuiteCourseConfig{}, false
}

func findBookTrumpConfig(course string) (platforms.BookTrumpCourseConfig, bool) {
	for _, config := range platforms.BookTrumpCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return platforms.BookTrumpCourseConfig{}, false
}

func metroForCourse(course string) string {
	if c, ok := findChronogolfConfig(course); ok {
		return c.Metro
	}
	if c, ok := findMemberSportsConfig(course); ok {
		return c.Metro
	}
	if c, ok := findCPSGolfConfig(course); ok {
		return c.Metro
	}
	if c, ok := findGolfNowConfig(course); ok {
		return c.Metro
	}
	if c, ok := findTeeItUpConfig(course); ok {
		return c.Metro
	}
	if c, ok := findClubCaddieConfig(course); ok {
		return c.Metro
	}
	if c, ok := findQuick18Config(course); ok {
		return c.Metro
	}
	if c, ok := findGolfWithAccessConfig(course); ok {
		return c.Metro
	}
	if c, ok := findCourseRevConfig(course); ok {
		return c.Metro
	}
	if c, ok := findRGuestConfig(course); ok {
		return c.Metro
	}
	if c, ok := findCourseCoConfig(course); ok {
		return c.Metro
	}
	if c, ok := findTeeSnapConfig(course); ok {
		return c.Metro
	}
	if c, ok := findForeUpConfig(course); ok {
		return c.Metro
	}
	if c, ok := findProphetConfig(course); ok {
		return c.Metro
	}
	if c, ok := findPurposeGolfConfig(course); ok {
		return c.Metro
	}
	if c, ok := findTeeQuestConfig(course); ok {
		return c.Metro
	}
	if c, ok := findResortSuiteConfig(course); ok {
		return c.Metro
	}
	if c, ok := findBookTrumpConfig(course); ok {
		return c.Metro
	}
	return ""
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

	var prConfig platforms.ProphetCourseConfig
	var prFound bool
	prConfig, prFound = findProphetConfig(course)
	if prFound {
		return platforms.FetchProphet(prConfig, date)
	}

	var pgConfig platforms.PurposeGolfCourseConfig
	var pgFound bool
	pgConfig, pgFound = findPurposeGolfConfig(course)
	if pgFound {
		return platforms.FetchPurposeGolf(pgConfig, date)
	}

	var tqConfig platforms.TeeQuestCourseConfig
	var tqFound bool
	tqConfig, tqFound = findTeeQuestConfig(course)
	if tqFound {
		return platforms.FetchTeeQuest(tqConfig, date)
	}

	var rsConfig platforms.ResortSuiteCourseConfig
	var rsFound bool
	rsConfig, rsFound = findResortSuiteConfig(course)
	if rsFound {
		return platforms.FetchResortSuite(rsConfig, date)
	}

	var btConfig platforms.BookTrumpCourseConfig
	var btFound bool
	btConfig, btFound = findBookTrumpConfig(course)
	if btFound {
		return platforms.FetchBookTrump(btConfig, date)
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

	var prConfig platforms.ProphetCourseConfig
	var prFound bool
	prConfig, prFound = findProphetConfig(course)
	if prFound {
		return prConfig.BookingURL
	}

	var pgConfig platforms.PurposeGolfCourseConfig
	var pgFound bool
	pgConfig, pgFound = findPurposeGolfConfig(course)
	if pgFound {
		return pgConfig.BookingURL
	}

	var tqConfig platforms.TeeQuestCourseConfig
	var tqFound bool
	tqConfig, tqFound = findTeeQuestConfig(course)
	if tqFound {
		return tqConfig.BookingURL
	}

	var rsConfig platforms.ResortSuiteCourseConfig
	var rsFound bool
	rsConfig, rsFound = findResortSuiteConfig(course)
	if rsFound {
		return rsConfig.BookingURL
	}

	var btConfig platforms.BookTrumpCourseConfig
	var btFound bool
	btConfig, btFound = findBookTrumpConfig(course)
	if btFound {
		return btConfig.BookingURL
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
	var idx int = strings.Index(name, " - ")
	if idx > 0 {
		return name[:idx]
	}
	return name
}

func addAlert(phone string, course string, date string, startTime string, endTime string, minPlayers int, holes string) (platforms.Alert, error) {
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
				fmt.Println("  Checking:", alert.Course, "|", alert.Date, "|", alert.StartTime, "–", alert.EndTime, "|", alert.Phone)

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
					fmt.Println("    Sending SMS to", alert.Phone)
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
			saveAlerts(alerts)
		}

		fmt.Println("")
		fmt.Println("  Next check in 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}
