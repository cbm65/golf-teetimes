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
	RGuestKeys         []string
	CourseCoKeys       []string
	TeeSnapKeys        []string
	ForeUpKeys         []string
}

var Metros = map[string]Metro{
	"denver": {
		Name:        "Denver",
		Slug:        "denver",
		State:       "CO",
		Tagline:     "Municipal & Public Courses",
		CourseCount: 51,
		CityCount:   23,
		MemberSportsKeys: []string{"denver", "foxhollow", "foothills", "brokentee", "coalcreek"},
		ChronogolfKeys:   []string{"southsuburban", "lonetree", "littleton", "familysports", "highlandmeadows", "broadlands"},
		CPSGolfKeys:      []string{"greenvalleyranch", "indiantree", "emeraldgreens", "fossiltrace", "westminster", "flatirons", "indianpeaks"},
		GolfNowKeys:      []string{"murphycreek", "springhill", "meadowhills", "aurorahills", "saddlerock", "raccooncreek", "arrowhead", "beardance", "ridgecastlepines", "heatherridge", "heritageeaglebend"},
		TeeItUpKeys:      []string{"hylandhills", "stoneycreek", "commonground", "buffalorun", "riverdaledunes", "riverdaleknolls", "coloradonational", "plumcreek", "omniinterlocken"},
		ClubCaddieKeys:   []string{"applewood", "thelinks"},
		ForeUpKeys:       []string{"toddcreek"},
		Quick18Keys:      []string{"thorncreek"},
	},
	"phoenix": {
		Name:        "Phoenix",
		Slug:        "phoenix",
		State:       "AZ",
		Tagline:     "Valley of the Sun Public Courses",
		CourseCount: 88,
		CityCount:   21,
		GolfNowKeys: []string{"tpcscottsdale", "tpcscottsdalestadium", "ravengolfclub", "stonecreek", "verrado", "verradofounders", "quintero", "longbow", "superstitionsprings", "ocotillo", "dovevalleyranch", "mccormickranchpine", "mccormickranchpalm", "talkingstickoodham", "talkingstickpiipaash", "whirlwinddevilsclaw", "whirlwindcattail", "westernskies", "kokopelli", "boulders", "continental", "rollinghills", "tokasticks", "sanmarcos", "palmvalley", "bearcreekbear", "bearcreekcub"},
		TeeItUpKeys: []string{"dobsonranch", "aguila", "aguila9", "cavecreek", "encanto", "encanto9", "paloverde", "arizonagrand", "cimarron", "granitefallsnorth", "desertsprings", "granitefallssouth", "silverado", "paradisevalley", "legacygolfclub", "lassendas", "starfire", "coldwater", "greenfieldlakes", "coronado", "bellair", "santanhighlands", "royalpalms", "viewpoint", "ranchomanana"},
		Quick18Keys: []string{"papago", "grayhawk", "trilogyvistancia", "coyotelakes", "sunridgecanyon", "orangetree", "goldcanyon", "redmountainranch"},
		GolfWithAccessKeys: []string{"lookoutmountain", "akchinsoutherndunes", "troonnorthpinnacle", "troonnorthmonument", "kierland", "eaglemountain", "lascolinas", "phoenician", "estrella"},
		CourseRevKeys: []string{"wigwamgold", "wigwamblue", "wigwamred", "biltmoreestates", "biltmorelinks"},
		RGuestKeys:    []string{"wekopacholla", "wekopasaguaro", "wildfirefaldo", "wildfirepalmer", "camelbackambiente", "camelbackpadre"},
		CourseCoKeys:  []string{"kenmcdonald"},
		TeeSnapKeys:   []string{"sundance"},
		ForeUpKeys:    []string{"legendtrail", "paintedmountain", "lonetree"},
	},
	"lasvegas": {
		Name:        "Las Vegas",
		Slug:        "lasvegas",
		State:       "NV",
		Tagline:     "Desert Golf Year-Round",
		CourseCount: 38,
		CityCount:   7,
		GolfNowKeys: []string{"lasvegasgolfclub", "lasvegasnational", "aliante", "rhodesranch", "angelparkmountain", "angelparkpalm", "angelparkcloudnine", "legacylv", "arroyoredrock", "losprados", "tpclasvegas", "desertwillow", "wildhorse", "bouldercity", "bouldercreek", "reverelexington", "revereconcord", "reflectionbay", "palmvalleylv", "highlandfalls", "eaglecrestlv", "bearsbest", "chimera", "siena", "painteddesert", "desertpines", "balihai", "stallionmountain", "royallinks", "paiutesun", "paiutesnow", "paiutewolf", "coyotesprings"},
		CPSGolfKeys:        []string{"serket", "cascata"},
		ChronogolfKeys:     []string{"clubatsunrise"},
		TeeItUpKeys:        []string{"durangohills"},
		GolfWithAccessKeys: []string{"mountainfalls"},
	},
}

func GetMetroList() []Metro {
	var list []Metro
	for _, m := range Metros {
		list = append(list, m)
	}
	return list
}
