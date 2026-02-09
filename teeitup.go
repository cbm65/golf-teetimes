package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TeeItUpCourseConfig struct {
	Alias       string
	FacilityID  string
	BookingURL  string
	DisplayName string
	City        string
	State       string
}

var TeeItUpCourses = map[string]TeeItUpCourseConfig{
	"hylandhills": {
		Alias:       "hyland-hills-park-recreation-district",
		FacilityID:  "9201",
		BookingURL:  "https://hyland-hills-park-recreation-district.book.teeitup.com/teetimes",
		DisplayName: "Hyland Hills",
		City: "Westminster", State: "CO",
	},
	"stoneycreek": {
		Alias:       "stoney-creek-golf-course",
		FacilityID:  "13099",
		BookingURL:  "https://stoney-creek-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Stoney Creek",
		City: "Arvada", State: "CO",
	},
	"commonground": {
		Alias:       "commonground-golf-course",
		FacilityID:  "5275",
		BookingURL:  "https://commonground-golf-course.book.teeitup.com/teetimes",
		DisplayName: "CommonGround",
		City: "Aurora", State: "CO",
	},
	"buffalorun": {
		Alias:       "buffalo-run-golf-course",
		FacilityID:  "513",
		BookingURL:  "https://buffalo-run-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Buffalo Run",
		City: "Commerce City", State: "CO",
	},
	"dobsonranch": {
		Alias:       "dobson-ranch",
		FacilityID:  "6501",
		BookingURL:  "https://dobson-ranch.book.teeitup.golf/teetimes",
		DisplayName: "Dobson Ranch",
		City: "Mesa", State: "AZ",
	},
	"aguila": {
		Alias:       "city-of-phoenix-golf-courses",
		FacilityID:  "287",
		BookingURL:  "https://city-of-phoenix-golf-courses.book.teeitup.golf/teetimes",
		DisplayName: "Aguila Golf Course",
		City: "Phoenix", State: "AZ",
	},
	"aguila9": {
		Alias:       "city-of-phoenix-golf-courses",
		FacilityID:  "4322",
		BookingURL:  "https://city-of-phoenix-golf-courses.book.teeitup.golf/teetimes",
		DisplayName: "Aguila Golf Course 9",
		City: "Phoenix", State: "AZ",
	},
	"cavecreek": {
		Alias:       "city-of-phoenix-golf-courses",
		FacilityID:  "288",
		BookingURL:  "https://city-of-phoenix-golf-courses.book.teeitup.golf/teetimes",
		DisplayName: "Cave Creek Golf Course",
		City: "Phoenix", State: "AZ",
	},
	"encanto": {
		Alias:       "city-of-phoenix-golf-courses",
		FacilityID:  "289",
		BookingURL:  "https://city-of-phoenix-golf-courses.book.teeitup.golf/teetimes",
		DisplayName: "Encanto Golf Course",
		City: "Phoenix", State: "AZ",
	},
	"encanto9": {
		Alias:       "city-of-phoenix-golf-courses",
		FacilityID:  "4323",
		BookingURL:  "https://city-of-phoenix-golf-courses.book.teeitup.golf/teetimes",
		DisplayName: "Encanto Golf Course 9",
		City: "Phoenix", State: "AZ",
	},
	"paloverde": {
		Alias:       "city-of-phoenix-golf-courses",
		FacilityID:  "3209",
		BookingURL:  "https://city-of-phoenix-golf-courses.book.teeitup.golf/teetimes",
		DisplayName: "Palo Verde",
		City: "Phoenix", State: "AZ",
	},
	"arizonagrand": {
		Alias:       "arizona-grand-golf-course",
		FacilityID:  "12",
		BookingURL:  "https://arizona-grand-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Arizona Grand",
		City: "Phoenix", State: "AZ",
	},
	"cimarron": {
		Alias:       "cimarron-golf-course",
		FacilityID:  "5216",
		BookingURL:  "https://cimarron-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Cimarron Golf Course",
		City: "Surprise", State: "AZ",
	},
	"granitefallsnorth": {
		Alias:       "granite-falls-golf-course-north",
		FacilityID:  "167",
		BookingURL:  "https://granite-falls-golf-course-north.book.teeitup.golf/teetimes",
		DisplayName: "Granite Falls North",
		City: "Surprise", State: "AZ",
	},
	"desertsprings": {
		Alias:       "desert-springs-golf-course",
		FacilityID:  "164",
		BookingURL:  "https://desert-springs-golf-course.book.teeitup.golf/teetimes",
		DisplayName: "Desert Springs",
		City: "Surprise", State: "AZ",
	},
	"granitefallssouth": {
		Alias:       "granite-falls-golf-course-south",
		FacilityID:  "11485",
		BookingURL:  "https://granite-falls-golf-course-south.book.teeitup.golf/teetimes",
		DisplayName: "Granite Falls South",
		City: "Surprise", State: "AZ",
	},
}

type TeeItUpResponse struct {
	Teetimes []TeeItUpTeeTime `json:"teetimes"`
}

type TeeItUpTeeTime struct {
	Teetime       string          `json:"teetime"`
	MaxPlayers    int             `json:"maxPlayers"`
	BookedPlayers int             `json:"bookedPlayers"`
	Rates         []TeeItUpRate   `json:"rates"`
}

type TeeItUpRate struct {
	Name           string  `json:"name"`
	Holes          int     `json:"holes"`
	GreenFeeCart   float64 `json:"greenFeeCart"`
	GreenFeeWalking float64 `json:"greenFeeWalking"`
	AllowedPlayers []int   `json:"allowedPlayers"`
}

var denverLoc *time.Location

func init() {
	var err error
	denverLoc, err = time.LoadLocation("America/Denver")
	if err != nil {
		denverLoc = time.FixedZone("MST", -7*60*60)
	}
}

func fetchTeeItUp(config TeeItUpCourseConfig, date string) ([]DisplayTeeTime, error) {
	var url string = fmt.Sprintf(
		"https://phx-api-be-east-1b.kenna.io/v2/tee-times?date=%s&facilityIds=%s&dateMax=%s",
		date, config.FacilityID, date,
	)

	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-be-alias", config.Alias)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://"+config.Alias+".book.teeitup.com")

	var client http.Client
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data []TeeItUpResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	var results []DisplayTeeTime
	for _, tt := range data[0].Teetimes {
		// Parse UTC time and convert to Denver time
		var t time.Time
		t, err = time.Parse("2006-01-02T15:04:05.000Z", tt.Teetime)
		if err != nil {
			continue
		}
		t = t.In(denverLoc)
		var timeStr string = t.Format("3:04 PM")

		var openings int = tt.MaxPlayers

		// Get price and holes from cheapest rate
		var price float64 = 0
		var holes int = 18
		if len(tt.Rates) > 0 {
			holes = tt.Rates[0].Holes
			var bestPrice float64 = 0
			for _, rate := range tt.Rates {
				var ratePrice float64 = 0
				if rate.GreenFeeCart > 0 {
					ratePrice = rate.GreenFeeCart / 100
				} else if rate.GreenFeeWalking > 0 {
					ratePrice = rate.GreenFeeWalking / 100
				}
				if ratePrice > 0 && (bestPrice == 0 || ratePrice < bestPrice) {
					bestPrice = ratePrice
				}
				if rate.Holes > 0 {
					holes = rate.Holes
				}
			}
			price = bestPrice
		}

		var holesStr string = fmt.Sprintf("%d", holes)

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holesStr,
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
