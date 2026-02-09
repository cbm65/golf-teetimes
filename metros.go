package main

type Metro struct {
	Name             string
	Slug             string
	State            string
	Tagline          string
	CourseCount      int
	CityCount        int
	MemberSportsKeys []string
	ChronogolfKeys   []string
	CPSGolfKeys      []string
	GolfNowKeys      []string
	TeeItUpKeys      []string
	ClubCaddieKeys   []string
	Quick18Keys      []string
	GolfWithAccessKeys []string
	CourseRevKeys      []string
}

var Metros = map[string]Metro{
	"denver": {
		Name:        "Denver",
		Slug:        "denver",
		State:       "CO",
		Tagline:     "Municipal & Public Courses",
		CourseCount: 34,
		CityCount:   12,
		MemberSportsKeys: []string{"denver", "foxhollow", "foothills", "brokentee"},
		ChronogolfKeys:   []string{"southsuburban", "lonetree", "littleton", "familysports"},
		CPSGolfKeys:      []string{"greenvalleyranch", "indiantree", "emeraldgreens", "fossiltrace", "westminster"},
		GolfNowKeys:      []string{"murphycreek", "springhill", "meadowhills", "aurorahills", "saddlerock", "raccooncreek", "arrowhead"},
		TeeItUpKeys:      []string{"hylandhills", "stoneycreek", "commonground", "buffalorun"},
		ClubCaddieKeys:   []string{"applewood", "thelinks"},
	},
	"phoenix": {
		Name:        "Phoenix",
		Slug:        "phoenix",
		State:       "AZ",
		Tagline:     "Valley of the Sun Public Courses",
		CourseCount: 45,
		CityCount:   11,
		GolfNowKeys: []string{"tpcscottsdale", "tpcscottsdalestadium", "ravengolfclub", "stonecreek", "verrado", "verradofounders", "quintero", "longbow", "superstitionsprings", "ocotillo", "dovevalleyranch", "mccormickranchpine", "mccormickranchpalm"},
		TeeItUpKeys: []string{"dobsonranch", "aguila", "aguila9", "cavecreek", "encanto", "encanto9", "paloverde", "arizonagrand", "cimarron", "granitefallsnorth", "desertsprings", "granitefallssouth", "silverado", "paradisevalley", "legacygolfclub"},
		Quick18Keys: []string{"papago", "grayhawk", "trilogyvistancia", "coyotelakes", "sunridgecanyon", "orangetree"},
		GolfWithAccessKeys: []string{"lookoutmountain", "akchinsoutherndunes", "troonnorthpinnacle", "troonnorthmonument"},
		CourseRevKeys: []string{"wigwamgold", "wigwamblue", "wigwamred", "biltmoreestates", "biltmorelinks"},
	},
}

func GetMetroList() []Metro {
	var list []Metro
	for _, m := range Metros {
		list = append(list, m)
	}
	return list
}
