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
}

func GetMetroList() []Metro {
	var list []Metro
	for _, m := range Metros {
		list = append(list, m)
	}
	return list
}
