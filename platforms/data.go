package platforms

import (
	"embed"
	"encoding/json"
	"fmt"
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
	for _, c := range loadJSON[GolfWithAccessCourseConfig]("data/golfwithaccess.json") {
		GolfWithAccessCourses[c.Key] = c
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
	for _, c := range GolfWithAccessCourses {
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

	var result = map[string][2]int{}
	for slug, s := range stats {
		result[slug] = [2]int{s.Courses, len(s.Cities)}
	}
	return result
}
