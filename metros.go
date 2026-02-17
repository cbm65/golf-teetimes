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
		Name:    "Dallas",
		Slug:    "dallas",
		State:   "TX",
		Tagline: "Public Courses Across DFW",
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
		Name:    "Miami",
		Slug:    "miami",
		State:   "FL",
		Tagline: "Public Courses Across South Florida",
	},
	"sanfrancisco": {
		Name:    "San Francisco",
		Slug:    "sanfrancisco",
		State:   "CA",
		Tagline: "Public Courses Across the Bay Area",
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
	"montgomery": {
		Name:    "Montgomery",
		Slug:    "montgomery",
		State:   "AL",
		Tagline: "Public Courses Across the River Region",
	},
	"losangeles": {
		Name:    "Los Angeles",
		Slug:    "losangeles",
		State:   "CA",
		Tagline: "Public Courses Across LA County",
	},
	"charlotte": {
		Name:    "Charlotte",
		Slug:    "charlotte",
		State:   "NC",
		Tagline: "Public Courses Across the Queen City",
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
