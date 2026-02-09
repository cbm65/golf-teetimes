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

func findCPSGolfConfig(course string) (CPSGolfCourseConfig, bool) {
	for _, config := range CPSGolfCourses {
		for _, displayName := range config.Names {
			if displayName == course {
				return config, true
			}
		}
	}
	return CPSGolfCourseConfig{}, false
}

func findClubCaddieConfig(course string) (ClubCaddieCourseConfig, bool) {
	for _, config := range ClubCaddieCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return ClubCaddieCourseConfig{}, false
}

func findTeeItUpConfig(course string) (TeeItUpCourseConfig, bool) {
	for _, config := range TeeItUpCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return TeeItUpCourseConfig{}, false
}

func findGolfNowConfig(course string) (GolfNowCourseConfig, bool) {
	for _, config := range GolfNowCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return GolfNowCourseConfig{}, false
}

func findQuick18Config(course string) (Quick18CourseConfig, bool) {
	for _, config := range Quick18Courses {
		if config.DisplayName == course {
			return config, true
		}
		if config.NamePrefix != "" && strings.HasPrefix(course, config.NamePrefix) {
			return config, true
		}
	}
	return Quick18CourseConfig{}, false
}

func findGolfWithAccessConfig(course string) (GolfWithAccessCourseConfig, bool) {
	for _, config := range GolfWithAccessCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return GolfWithAccessCourseConfig{}, false
}

func findCourseRevConfig(course string) (CourseRevCourseConfig, bool) {
	for _, config := range CourseRevCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return CourseRevCourseConfig{}, false
}

func findRGuestConfig(course string) (RGuestCourseConfig, bool) {
	for _, config := range RGuestCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return RGuestCourseConfig{}, false
}

func findCourseCoConfig(course string) (CourseCoCourseConfig, bool) {
	for _, config := range CourseCoCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return CourseCoCourseConfig{}, false
}

func findTeeSnapConfig(course string) (TeeSnapCourseConfig, bool) {
	for _, config := range TeeSnapCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return TeeSnapCourseConfig{}, false
}

func findForeUpConfig(course string) (ForeUpCourseConfig, bool) {
	for _, config := range ForeUpCourses {
		if config.DisplayName == course {
			return config, true
		}
	}
	return ForeUpCourseConfig{}, false
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

	var cpsConfig CPSGolfCourseConfig
	var cpsFound bool
	cpsConfig, cpsFound = findCPSGolfConfig(course)
	if cpsFound {
		return fetchCPSGolf(cpsConfig, date)
	}

	var gnConfig GolfNowCourseConfig
	var gnFound bool
	gnConfig, gnFound = findGolfNowConfig(course)
	if gnFound {
		return fetchGolfNow(gnConfig, date)
	}

	var tiuConfig TeeItUpCourseConfig
	var tiuFound bool
	tiuConfig, tiuFound = findTeeItUpConfig(course)
	if tiuFound {
		return fetchTeeItUp(tiuConfig, date)
	}

	var ccConfig ClubCaddieCourseConfig
	var ccFound bool
	ccConfig, ccFound = findClubCaddieConfig(course)
	if ccFound {
		return fetchClubCaddie(ccConfig, date)
	}

	var q18Config Quick18CourseConfig
	var q18Found bool
	q18Config, q18Found = findQuick18Config(course)
	if q18Found {
		return fetchQuick18(q18Config, date)
	}

	var gwaConfig GolfWithAccessCourseConfig
	var gwaFound bool
	gwaConfig, gwaFound = findGolfWithAccessConfig(course)
	if gwaFound {
		return fetchGolfWithAccess(gwaConfig, date)
	}

	var crConfig CourseRevCourseConfig
	var crFound bool
	crConfig, crFound = findCourseRevConfig(course)
	if crFound {
		return fetchCourseRev(crConfig, date)
	}

	var rgConfig RGuestCourseConfig
	var rgFound bool
	rgConfig, rgFound = findRGuestConfig(course)
	if rgFound {
		return fetchRGuest(rgConfig, date)
	}

	var coConfig CourseCoCourseConfig
	var coFound bool
	coConfig, coFound = findCourseCoConfig(course)
	if coFound {
		return fetchCourseCo(coConfig, date)
	}

	var tsConfig TeeSnapCourseConfig
	var tsFound bool
	tsConfig, tsFound = findTeeSnapConfig(course)
	if tsFound {
		return fetchTeeSnap(tsConfig, date)
	}

	var fuConfig ForeUpCourseConfig
	var fuFound bool
	fuConfig, fuFound = findForeUpConfig(course)
	if fuFound {
		return fetchForeUp(fuConfig, date)
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

	var cpsConfig CPSGolfCourseConfig
	var cpsFound bool
	cpsConfig, cpsFound = findCPSGolfConfig(course)
	if cpsFound {
		return cpsConfig.BookingURL
	}

	var gnConfig GolfNowCourseConfig
	var gnFound bool
	gnConfig, gnFound = findGolfNowConfig(course)
	if gnFound {
		return gnConfig.BookingURL
	}

	var tiuConfig TeeItUpCourseConfig
	var tiuFound bool
	tiuConfig, tiuFound = findTeeItUpConfig(course)
	if tiuFound {
		return tiuConfig.BookingURL
	}

	var ccConfig ClubCaddieCourseConfig
	var ccFound bool
	ccConfig, ccFound = findClubCaddieConfig(course)
	if ccFound {
		return ccConfig.BookingURL
	}

	var q18Config Quick18CourseConfig
	var q18Found bool
	q18Config, q18Found = findQuick18Config(course)
	if q18Found {
		return q18Config.BookingURL
	}

	var gwaConfig GolfWithAccessCourseConfig
	var gwaFound bool
	gwaConfig, gwaFound = findGolfWithAccessConfig(course)
	if gwaFound {
		return gwaConfig.BookingURL
	}

	var crConfig CourseRevCourseConfig
	var crFound bool
	crConfig, crFound = findCourseRevConfig(course)
	if crFound {
		return crConfig.BookingURL
	}

	var rgConfig RGuestCourseConfig
	var rgFound bool
	rgConfig, rgFound = findRGuestConfig(course)
	if rgFound {
		return rgConfig.BookingURL
	}

	var coConfig CourseCoCourseConfig
	var coFound bool
	coConfig, coFound = findCourseCoConfig(course)
	if coFound {
		return coConfig.BookingURL
	}

	var tsConfig TeeSnapCourseConfig
	var tsFound bool
	tsConfig, tsFound = findTeeSnapConfig(course)
	if tsFound {
		return tsConfig.BookingURL
	}

	var fuConfig ForeUpCourseConfig
	var fuFound bool
	fuConfig, fuFound = findForeUpConfig(course)
	if fuFound {
		return fuConfig.BookingURL
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
