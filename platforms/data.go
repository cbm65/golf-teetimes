package platforms

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/*.json
var dataFS embed.FS

func loadJSON[T any](filename string) []T {
	var data []byte
	var err error
	data, err = dataFS.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("failed to read %s: %v", filename, err))
	}
	var list []T
	err = json.Unmarshal(data, &list)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %s: %v", filename, err))
	}
	return list
}

func init() {
	for _, c := range loadJSON[TeeItUpCourseConfig]("data/teeitup.json") {
		TeeItUpCourses[c.Key] = c
	}
	for _, c := range loadJSON[GolfNowCourseConfig]("data/golfnow.json") {
		GolfNowCourses[c.Key] = c
	}
	for _, c := range loadJSON[ChronogolfCourseConfig]("data/chronogolf.json") {
		ChronogolfCourses[c.Key] = c
	}
	for _, c := range loadJSON[ForeUpCourseConfig]("data/foreup.json") {
		ForeUpCourses[c.Key] = c
	}
	for _, c := range loadJSON[CPSGolfCourseConfig]("data/cpsgolf.json") {
		CPSGolfCourses[c.Key] = c
	}
	for _, c := range loadJSON[MemberSportsCourseConfig]("data/membersports.json") {
		MemberSportsCourses[c.Key] = c
	}
	for _, c := range loadJSON[ClubCaddieCourseConfig]("data/clubcaddie.json") {
		ClubCaddieCourses[c.Key] = c
	}
	for _, c := range loadJSON[Quick18CourseConfig]("data/quick18.json") {
		Quick18Courses[c.Key] = c
	}
	for _, c := range loadJSON[CourseRevCourseConfig]("data/courserev.json") {
		CourseRevCourses[c.Key] = c
	}
	for _, c := range loadJSON[RGuestCourseConfig]("data/rguest.json") {
		RGuestCourses[c.Key] = c
	}
	for _, c := range loadJSON[CourseCoCourseConfig]("data/courseco.json") {
		CourseCoCourses[c.Key] = c
	}
	for _, c := range loadJSON[TeeSnapCourseConfig]("data/teesnap.json") {
		TeeSnapCourses[c.Key] = c
	}
	for _, c := range loadJSON[ProphetCourseConfig]("data/prophet.json") {
		ProphetCourses[c.Key] = c
	}
	for _, c := range loadJSON[PurposeGolfCourseConfig]("data/purposegolf.json") {
		PurposeGolfCourses[c.Key] = c
	}
	for _, c := range loadJSON[TeeQuestCourseConfig]("data/teequest.json") {
		TeeQuestCourses[c.Key] = c
	}
	for _, c := range loadJSON[ResortSuiteCourseConfig]("data/resortsuite.json") {
		ResortSuiteCourses[c.Key] = c
	}
	for _, c := range loadJSON[BookTrumpCourseConfig]("data/booktrump.json") {
		BookTrumpCourses[c.Key] = c
	}
	for _, c := range loadJSON[TeeOnCourseConfig]("data/teeon.json") {
		TeeOnCourses[c.Key] = c
	}
	for _, c := range loadJSON[GolfBackCourseConfig]("data/golfback.json") {
		GolfBackCourses[c.Key] = c
	}
	for _, c := range loadJSON[WGMCourseConfig]("data/wgm.json") {
		WGMCourses[c.Key] = c
	}
	for _, c := range loadJSON[TeeTimeCentralCourseConfig]("data/teetimecentral.json") {
		TeeTimeCentralCourses[c.Key] = c
	}
	for _, c := range loadJSON[LetsGoGolfCourseConfig]("data/letsgogolf.json") {
		LetsGoGolfCourses[c.Key] = c
	}
	for _, c := range loadJSON[GuestDeskSiteConfig]("data/guestdesk.json") {
		GuestDeskCourses[c.Key] = c
	}
	for _, c := range loadJSON[TenForeCourseConfig]("data/tenfore.json") {
		TenForeCourses[c.Key] = c
	}

	// --- Populate global Registry ---
	reg := func(key, metro, city, displayName, bookingURL string, enabled bool, fetch func(string) ([]DisplayTeeTime, error)) {
		Registry = append(Registry, CourseEntry{
			Key: key, Metro: metro, City: city,
			Match:      func(name string) bool { return displayName == name },
			BookingURL: bookingURL, Enabled: enabled, Fetch: fetch,
		})
	}

	// Simple platforms (DisplayName matching)
	for _, c := range BookTrumpCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchBookTrump(c, d) })
	}
	for _, c := range ClubCaddieCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchClubCaddie(c, d) })
	}
	for _, c := range CourseCoCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchCourseCo(c, d) })
	}
	for _, c := range CourseRevCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchCourseRev(c, d) })
	}
	for _, c := range ForeUpCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchForeUp(c, d) })
	}
	for _, c := range GolfBackCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchGolfBack(c, d) })
	}
	for _, c := range WGMCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchWGM(c, d) })
	}
	for _, c := range GolfNowCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchGolfNow(c, d) })
	}
	for _, c := range LetsGoGolfCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchLetsGoGolf(c, d) })
	}
	for _, c := range PurposeGolfCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchPurposeGolf(c, d) })
	}
	for _, c := range RGuestCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchRGuest(c, d) })
	}
	for _, c := range ResortSuiteCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchResortSuite(c, d) })
	}
	for _, c := range TeeOnCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchTeeOn(c, d) })
	}
	for _, c := range TeeQuestCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchTeeQuest(c, d) })
	}
	for _, c := range TeeSnapCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchTeeSnap(c, d) })
	}
	for _, c := range TeeTimeCentralCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchTeeTimeCentral(c, d) })
	}
	for _, c := range TenForeCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, true, func(d string) ([]DisplayTeeTime, error) { return FetchTenFore(c, d) })
	}

	// Prophet — disabled (WAF blocks most requests)
	for _, c := range ProphetCourses {
		reg(c.Key, c.Metro, c.City, c.DisplayName, c.BookingURL, false, func(d string) ([]DisplayTeeTime, error) { return FetchProphet(c, d) })
	}

	// Platforms with Names-map matching (Chronogolf, CPSGolf, MemberSports)
	for _, c := range ChronogolfCourses {
		c := c
		Registry = append(Registry, CourseEntry{
			Key: c.Key, Metro: c.Metro, City: c.City, BookingURL: c.BookingURL, Enabled: true,
			Fetch: func(d string) ([]DisplayTeeTime, error) { return FetchChronogolf(c, d) },
			Match: func(name string) bool {
				for _, dn := range c.Names {
					if dn == name {
						return true
					}
				}
				return false
			},
		})
	}
	for _, c := range CPSGolfCourses {
		c := c
		Registry = append(Registry, CourseEntry{
			Key: c.Key, Metro: c.Metro, City: c.City, BookingURL: c.BookingURL, Enabled: true,
			Fetch: func(d string) ([]DisplayTeeTime, error) { return FetchCPSGolf(c, d) },
			Match: func(name string) bool {
				for _, dn := range c.Names {
					if dn == name {
						return true
					}
				}
				return false
			},
		})
	}
	for _, c := range MemberSportsCourses {
		c := c
		Registry = append(Registry, CourseEntry{
			Key: c.Key, Metro: c.Metro, City: c.City, BookingURL: c.BookingURL, Enabled: true,
			Fetch: func(d string) ([]DisplayTeeTime, error) { return FetchMemberSports(c, d) },
			Match: func(name string) bool {
				for _, kn := range c.KnownCourses {
					if kn == name {
						return true
					}
				}
				return false
			},
		})
	}

	// TeeItUp — DisplayName + Names values + base-course matching, custom booking URL
	for _, c := range TeeItUpCourses {
		c := c
		Registry = append(Registry, CourseEntry{
			Key: c.Key, Metro: c.Metro, City: c.City, Enabled: true,
			BookingURL: "https://" + c.Alias + ".book.teeitup.com/teetimes",
			Fetch:      func(d string) ([]DisplayTeeTime, error) { return FetchTeeItUp(c, d) },
			Match: func(name string) bool {
				if c.DisplayName == name {
					return true
				}
				for _, n := range c.Names {
					if n == name || GetBaseCourse(n) == name {
						return true
					}
				}
				return false
			},
		})
	}

	// Quick18 — DisplayName + NamePrefix matching
	for _, c := range Quick18Courses {
		c := c
		Registry = append(Registry, CourseEntry{
			Key: c.Key, Metro: c.Metro, City: c.City, BookingURL: c.BookingURL, Enabled: true,
			Fetch: func(d string) ([]DisplayTeeTime, error) { return FetchQuick18(c, d) },
			Match: func(name string) bool {
				if c.DisplayName == name {
					return true
				}
				return c.NamePrefix != "" && strings.HasPrefix(name, c.NamePrefix)
			},
		})
	}

	// GuestDesk — site with nested courses
	for _, s := range GuestDeskCourses {
		s := s
		city := ""
		if len(s.Courses) > 0 {
			city = s.Courses[0].City
		}
		Registry = append(Registry, CourseEntry{
			Key: s.Key, Metro: s.Metro, City: city, BookingURL: s.BookingURL, Enabled: true,
			Fetch: func(d string) ([]DisplayTeeTime, error) { return FetchGuestDesk(s, d) },
			Match: func(name string) bool {
				for _, gc := range s.Courses {
					if gc.DisplayName == name {
						return true
					}
				}
				return false
			},
		})
	}
}

type metroStat struct {
	Courses int
	Cities  map[string]bool
}

// MetroStats returns course count and unique city count per metro slug.
func MetroStats() map[string][2]int {
	var stats = map[string]*metroStat{}
	ensure := func(metro, city string) {
		if metro == "" {
			return
		}
		if stats[metro] == nil {
			stats[metro] = &metroStat{Cities: map[string]bool{}}
		}
		stats[metro].Courses++
		stats[metro].Cities[city] = true
	}
	for _, c := range MemberSportsCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range ChronogolfCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range CPSGolfCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range GolfNowCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range TeeItUpCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range ClubCaddieCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range Quick18Courses {
		ensure(c.Metro, c.City)
	}
	for _, c := range CourseRevCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range RGuestCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range CourseCoCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range TeeSnapCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range ForeUpCourses {
		ensure(c.Metro, c.City)
	}
	// Prophet disabled — AWS WAF blocks most requests
	// for _, c := range ProphetCourses {
	// 	ensure(c.Metro, c.City)
	// }
	for _, c := range PurposeGolfCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range TeeQuestCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range ResortSuiteCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range BookTrumpCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range TeeOnCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range GolfBackCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range WGMCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range TenForeCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range TeeTimeCentralCourses {
		ensure(c.Metro, c.City)
	}
	for _, c := range LetsGoGolfCourses {
		ensure(c.Metro, c.City)
	}
	for _, s := range GuestDeskCourses {
		for _, c := range s.Courses {
			ensure(s.Metro, c.City)
		}
	}

	var result = map[string][2]int{}
	for slug, s := range stats {
		result[slug] = [2]int{s.Courses, len(s.Cities)}
	}
	return result
}
