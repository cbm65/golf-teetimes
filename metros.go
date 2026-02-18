package main

import "golf-teetimes/platforms"

type Metro struct {
	Name        string
	Slug        string
	State       string
	Tagline     string
	CourseCount int
	CityCount   int
}

var Metros = map[string]Metro{
	"denver": {
		Name:    "Denver",
		Slug:    "denver",
		State:   "CO",
		Tagline: "Municipal & Public Courses",
	},
	"phoenix": {
		Name:    "Phoenix",
		Slug:    "phoenix",
		State:   "AZ",
		Tagline: "Valley of the Sun Public Courses",
	},
	"lasvegas": {
		Name:    "Las Vegas",
		Slug:    "lasvegas",
		State:   "NV",
		Tagline: "Desert Golf Year-Round",
	},
	"atlanta": {
		Name:    "Atlanta",
		Slug:    "atlanta",
		State:   "GA",
		Tagline: "Public Courses Across Metro Atlanta",
	},
	"dallas": {
		Name:    "DFW",
		Slug:    "dallas",
		State:   "TX",
		Tagline: "Public Courses Across the Dallas-Fort Worth Metroplex",
	},
	"neworleans": {
		Name:    "New Orleans",
		Slug:    "neworleans",
		State:   "LA",
		Tagline: "Public Courses Across Metro New Orleans",
	},
	"nashville": {
		Name:    "Nashville",
		Slug:    "nashville",
		State:   "TN",
		Tagline: "Public Courses Across Middle Tennessee",
	},
	"miami": {
		Name:    "South Florida",
		Slug:    "miami",
		State:   "FL",
		Tagline: "Public Courses from Miami to Fort Lauderdale",
	},
	"sanfrancisco": {
		Name:    "Bay Area",
		Slug:    "sanfrancisco",
		State:   "CA",
		Tagline: "Public Courses from San Francisco to San Jose",
	},
	"albuquerque": {
		Name:    "Albuquerque & Santa Fe",
		Slug:    "albuquerque",
		State:   "NM",
		Tagline: "High Desert Golf Along the Rio Grande",
	},
	"oklahomacity": {
		Name:    "Oklahoma City",
		Slug:    "oklahomacity",
		State:   "OK",
		Tagline: "Public Courses Across Metro OKC",
	},
	"losangeles": {
		Name:    "LA & Orange County",
		Slug:    "losangeles",
		State:   "CA",
		Tagline: "Public Courses Across Los Angeles and Orange County",
	},
	"charlotte": {
		Name:    "Greater Charlotte",
		Slug:    "charlotte",
		State:   "NC",
		Tagline: "Public Courses Across the Carolinas' Queen City",
	},
	"sandiego": {
		Name:    "San Diego",
		Slug:    "sandiego",
		State:   "CA",
		Tagline: "Year-Round Golf Across San Diego County",
	},
	"austin": {
		Name:    "Austin",
		Slug:    "austin",
		State:   "TX",
		Tagline: "Public Courses Across the Texas Hill Country",
	},
	"houston": {
		Name:    "Houston",
		Slug:    "houston",
		State:   "TX",
		Tagline: "Public Courses Across Greater Houston",
	},
	"tampa": {
		Name:    "Tampa Bay",
		Slug:    "tampa",
		State:   "FL",
		Tagline: "Public Courses Across the Tampa Bay Area",
	},
	"orlando": {
		Name:    "Orlando",
		Slug:    "orlando",
		State:   "FL",
		Tagline: "Championship Golf in the Heart of Central Florida",
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
		list = append(list, m)
	}
	return list
}
