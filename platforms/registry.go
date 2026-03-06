package platforms

import "strings"

// CourseEntry is a single bookable course in the global registry.
type CourseEntry struct {
	Key        string
	Metro      string
	City       string
	Fetch      func(date string) ([]DisplayTeeTime, error)
	Match      func(name string) bool
	BookingURL string
	Enabled    bool // false = skip in metro tee-time fetches (e.g. Prophet/WAF-blocked)
}

// Registry holds every course across all platforms, populated during init().
var Registry []CourseEntry

// FindCourse returns the first entry whose Match function accepts the name.
// Searches all entries regardless of Enabled flag.
func FindCourse(name string) (*CourseEntry, bool) {
	for i := range Registry {
		if Registry[i].Match(name) {
			return &Registry[i], true
		}
	}
	return nil, false
}

// GetBaseCourse strips sub-course suffixes like " - Links" from a course name.
func GetBaseCourse(name string) string {
	var idx int = strings.Index(name, " - ")
	if idx > 0 {
		return name[:idx]
	}
	return name
}
