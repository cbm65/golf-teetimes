package main

import "golf-teetimes/platforms"

type Metro struct {
	Name        string
	Slug        string
	State       string
	Tagline     string
	CourseCount int
	CityCount   int
	Lat         float64
	Lng         float64
}

var Metros = map[string]Metro{
	"denver": {
		Name: "Denver", Slug: "denver", State: "CO",
		Tagline: "Municipal & Public Courses",
		Lat: 39.74, Lng: -104.99,
	},
	"phoenix": {
		Name: "Phoenix", Slug: "phoenix", State: "AZ",
		Tagline: "Valley of the Sun Public Courses",
		Lat: 33.45, Lng: -112.07,
	},
	"lasvegas": {
		Name: "Las Vegas", Slug: "lasvegas", State: "NV",
		Tagline: "Desert Golf Year-Round",
		Lat: 36.17, Lng: -115.14,
	},
	"atlanta": {
		Name: "Atlanta", Slug: "atlanta", State: "GA",
		Tagline: "Public Courses Across Metro Atlanta",
		Lat: 33.75, Lng: -84.39,
	},
	"dallas": {
		Name: "DFW", Slug: "dallas", State: "TX",
		Tagline: "Public Courses Across the Dallas-Fort Worth Metroplex",
		Lat: 32.78, Lng: -96.80,
	},
	"neworleans": {
		Name: "New Orleans", Slug: "neworleans", State: "LA",
		Tagline: "Public Courses Across Metro New Orleans",
		Lat: 29.95, Lng: -90.07,
	},
	"nashville": {
		Name: "Nashville", Slug: "nashville", State: "TN",
		Tagline: "Public Courses Across Middle Tennessee",
		Lat: 36.16, Lng: -86.78,
	},
	"miami": {
		Name: "South Florida", Slug: "miami", State: "FL",
		Tagline: "Public Courses from Miami to Fort Lauderdale",
		Lat: 26.00, Lng: -80.60,
	},
	"sanfrancisco": {
		Name: "Bay Area", Slug: "sanfrancisco", State: "CA",
		Tagline: "Public Courses from San Francisco to San Jose",
		Lat: 37.80, Lng: -121.90,
	},
	"albuquerque": {
		Name: "Albuquerque & Santa Fe", Slug: "albuquerque", State: "NM",
		Tagline: "High Desert Golf Along the Rio Grande",
		Lat: 35.08, Lng: -106.65,
	},
	"oklahomacity": {
		Name: "Oklahoma City", Slug: "oklahomacity", State: "OK",
		Tagline: "Public Courses Across Metro OKC",
		Lat: 35.47, Lng: -97.52,
	},
	"losangeles": {
		Name: "LA & Orange County", Slug: "losangeles", State: "CA",
		Tagline: "Public Courses Across Los Angeles and Orange County",
		Lat: 34.05, Lng: -118.24,
	},
	"charlotte": {
		Name: "Greater Charlotte", Slug: "charlotte", State: "NC",
		Tagline: "Public Courses Across the Carolinas' Queen City",
		Lat: 35.23, Lng: -80.84,
	},
	"sandiego": {
		Name: "San Diego", Slug: "sandiego", State: "CA",
		Tagline: "Year-Round Golf Across San Diego County",
		Lat: 32.90, Lng: -116.80,
	},
	"austin": {
		Name: "Austin", Slug: "austin", State: "TX",
		Tagline: "Public Courses Across the Texas Hill Country",
		Lat: 30.27, Lng: -97.74,
	},
	"houston": {
		Name: "Houston", Slug: "houston", State: "TX",
		Tagline: "Public Courses Across Greater Houston",
		Lat: 29.76, Lng: -95.37,
	},
	"tampa": {
		Name: "Tampa Bay", Slug: "tampa", State: "FL",
		Tagline: "Public Courses Across the Tampa Bay Area",
		Lat: 28.10, Lng: -82.10,
	},
	"tucson": {
		Name: "Tucson", Slug: "tucson", State: "AZ",
		Tagline: "Desert Golf in Southern Arizona",
		Lat: 32.22, Lng: -110.97,
	},
	"orlando": {
		Name: "Orlando", Slug: "orlando", State: "FL",
		Tagline: "Championship Golf in the Heart of Central Florida",
		Lat: 28.54, Lng: -81.38,
	},
	"myrtle-beach": {
		Name: "Myrtle Beach", Slug: "myrtle-beach", State: "SC",
		Tagline: "The Golf Capital of the World",
		Lat: 33.69, Lng: -78.89,
	},
	"palmsprings": {
		Name: "Palm Springs", Slug: "palmsprings", State: "CA",
		Tagline: "Desert Golf in the Coachella Valley",
		Lat: 33.83, Lng: -116.55,
	},
	"jacksonville": {
		Name: "Jacksonville", Slug: "jacksonville", State: "FL",
		Tagline: "Public Courses Across Northeast Florida",
		Lat: 30.33, Lng: -81.66,
	},
	"sanantonio": {
		Name: "San Antonio", Slug: "sanantonio", State: "TX",
		Tagline: "Public Courses Across the Alamo City",
		Lat: 29.42, Lng: -98.49,
	},
	"sacramento": {
		Name: "Sacramento", Slug: "sacramento", State: "CA",
		Tagline: "Public Courses Across the Sacramento Valley",
		Lat: 38.58, Lng: -121.49,
	},
	"charleston": {
		Name: "Charleston", Slug: "charleston", State: "SC",
		Tagline: "Lowcountry Public Golf from Kiawah to Summerville",
		Lat: 32.78, Lng: -79.93,
	},
	"hiltonhead": {
		Name: "Hilton Head", Slug: "hiltonhead", State: "SC",
		Tagline: "Lowcountry Golf from Hilton Head to Beaufort",
		Lat: 32.22, Lng: -80.75,
	},
}

func init() {
	var stats map[string][2]int = platforms.MetroStats()
	for slug, m := range Metros {
		if s, ok := stats[slug]; ok {
			m.CourseCount = s[0]
			m.CityCount = s[1]
			Metros[slug] = m
		}
	}
}

func GetMetroList() []Metro {
	var list []Metro
	for _, m := range Metros {
		if m.CourseCount == 0 {
			continue
		}
		list = append(list, m)
	}
	return list
}
