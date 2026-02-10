package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

type GolfNowCourseConfig struct {
	FacilityID  int
	SearchURL   string
	BookingURL  string
	DisplayName string
	City        string
	State       string
}

var GolfNowCourses = map[string]GolfNowCourseConfig{
	"murphycreek": {
		FacilityID:  17879,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17879-murphy-creek-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/17879/search",
		DisplayName: "Murphy Creek",
		City: "Aurora", State: "CO",
	},
	"springhill": {
		FacilityID:  17876,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17876-springhill-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/17876/search",
		DisplayName: "Springhill",
		City: "Aurora", State: "CO",
	},
	"meadowhills": {
		FacilityID:  17880,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17880-meadow-hills-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/17880/search",
		DisplayName: "Meadow Hills",
		City: "Aurora", State: "CO",
	},
	"aurorahills": {
		FacilityID:  17878,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17878-aurora-hills-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/17878/search",
		DisplayName: "Aurora Hills",
		City: "Aurora", State: "CO",
	},
	"saddlerock": {
		FacilityID:  17877,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/17877-saddle-rock-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/17877/search",
		DisplayName: "Saddle Rock",
		City: "Aurora", State: "CO",
	},
	"raccooncreek": {
		FacilityID:  515,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/515-raccoon-creek-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/515/search",
		DisplayName: "Raccoon Creek",
		City: "Littleton", State: "CO",
	},
	"arrowhead": {
		FacilityID:  453,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/453-arrowhead-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/453/search",
		DisplayName: "Arrowhead",
		City: "Littleton", State: "CO",
	},
	"tpcscottsdale": {
		FacilityID:  7076,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/7076-tpc-scottsdale-champions-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/7076/search",
		DisplayName: "TPC Scottsdale Champions",
		City: "Scottsdale", State: "AZ",
	},
	"tpcscottsdalestadium": {
		FacilityID:  3482,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3482-the-stadium-course-at-tpc-scottsdale/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/3482/search",
		DisplayName: "TPC Scottsdale Stadium",
		City: "Scottsdale", State: "AZ",
	},
	"ravengolfclub": {
		FacilityID:  1446,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1446-raven-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1446/search",
		DisplayName: "Raven Golf Club",
		City: "Phoenix", State: "AZ",
	},
	"stonecreek": {
		FacilityID:  122,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/122-stonecreek-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/122/search",
		DisplayName: "Stonecreek Golf Club",
		City: "Phoenix", State: "AZ",
	},
	"verrado": {
		FacilityID:  14378,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/14378-verrado-golf-club-victory-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/14378/search",
		DisplayName: "Verrado Victory",
		City: "Buckeye", State: "AZ",
	},
	"verradofounders": {
		FacilityID:  1707,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1707-verrado-golf-club-founders-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1707/search",
		DisplayName: "Verrado Founders",
		City: "Buckeye", State: "AZ",
	},
	"quintero": {
		FacilityID:  6388,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6388-quintero-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/6388/search",
		DisplayName: "Quintero Golf Club",
		City: "Peoria", State: "AZ",
	},
	"longbow": {
		FacilityID:  3021,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3021-longbow-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/3021/search",
		DisplayName: "Longbow Golf Club",
		City: "Mesa", State: "AZ",
	},
	"superstitionsprings": {
		FacilityID:  120,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/120-superstition-springs-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/120/search",
		DisplayName: "Superstition Springs",
		City: "Mesa", State: "AZ",
	},
	"ocotillo": {
		FacilityID:  253,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/253-ocotillo-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/253/search",
		DisplayName: "Ocotillo Golf Club",
		City: "Chandler", State: "AZ",
	},
	"dovevalleyranch": {
		FacilityID:  115,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/115-dove-valley-ranch-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/115/search",
		DisplayName: "Dove Valley Ranch",
		City: "Cave Creek", State: "AZ",
	},
	"mccormickranchpine": {
		FacilityID:  7078,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/7078-mccormick-ranch-golf-club-pine-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/7078/search",
		DisplayName: "McCormick Ranch Pine",
		City: "Scottsdale", State: "AZ",
	},
	"mccormickranchpalm": {
		FacilityID:  1356,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1356-mccormick-ranch-golf-club-palm-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1356/search",
		DisplayName: "McCormick Ranch Palm",
		City: "Scottsdale", State: "AZ",
	},
	"talkingstickoodham": {
		FacilityID:  12968,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/12968-talking-stick-golf-club-oodham-north/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/12968/search",
		DisplayName: "Talking Stick O'odham",
		City: "Scottsdale", State: "AZ",
	},
	"talkingstickpiipaash": {
		FacilityID:  814,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/814-talking-stick-golf-club-piipaash-south/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/814/search",
		DisplayName: "Talking Stick Piipaash",
		City: "Scottsdale", State: "AZ",
	},
	"whirlwinddevilsclaw": {
		FacilityID:  110,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/110-whirlwind-golf-club-devils-claw/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/110/search",
		DisplayName: "Whirlwind Devil's Claw",
		City: "Chandler", State: "AZ",
	},
	"whirlwindcattail": {
		FacilityID:  13192,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/13192-whirlwind-golf-club-cattail/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/13192/search",
		DisplayName: "Whirlwind Cattail",
		City: "Chandler", State: "AZ",
	},
	"westernskies": {
		FacilityID:  123,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/123-western-skies-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/123/search",
		DisplayName: "Western Skies Golf Club",
		City: "Gilbert", State: "AZ",
	},
	"kokopelli": {
		FacilityID:  121,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/121-kokopelli-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/121/search",
		DisplayName: "Kokopelli Golf Club",
		City: "Gilbert", State: "AZ",
	},
	"boulders": {
		FacilityID:  7,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/7-boulders-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/7/search",
		DisplayName: "Boulders Golf Club",
		City: "Carefree", State: "AZ",
	},
	"continental": {
		FacilityID:  428,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/428-continental-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/428/search",
		DisplayName: "Continental Golf Club",
		City: "Scottsdale", State: "AZ",
	},
	"rollinghills": {
		FacilityID:  3633,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3633-rolling-hills-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/3633/search",
		DisplayName: "Rolling Hills Golf Course",
		City: "Tempe", State: "AZ",
	},
	"tokasticks": {
		FacilityID:  1039,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1039-toka-sticks-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1039/search",
		DisplayName: "Toka Sticks Golf Club",
		City: "Mesa", State: "AZ",
	},
	"sanmarcos": {
		FacilityID:  1383,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1383-san-marcos-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1383/search",
		DisplayName: "San Marcos Golf Course",
		City: "Chandler", State: "AZ",
	},
	"palmvalley": {
		FacilityID:  4682,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/4682-palm-valley-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/4682/search",
		DisplayName: "Palm Valley Golf Club",
		City: "Goodyear", State: "AZ",
	},
	"bearcreekbear": {
		FacilityID:  62,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/62-bear-creek-gc-bear-championship/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/62/search",
		DisplayName: "Bear Creek Golf Club",
		City: "Chandler", State: "AZ",
	},
	"bearcreekcub": {
		FacilityID:  3685,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3685-bear-creek-gc-cub-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/3685/search",
		DisplayName: "Bear Creek Cub Course",
		City: "Chandler", State: "AZ",
	},
	"beardance": {
		FacilityID:  516,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/516-the-golf-club-at-bear-dance/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/516/search",
		DisplayName: "Bear Dance",
		City: "Larkspur", State: "CO",
	},
	"ridgecastlepines": {
		FacilityID:  1411,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1411-the-ridge-at-castle-pines-north/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1411/search",
		DisplayName: "Ridge at Castle Pines",
		City: "Castle Pines", State: "CO",
	},
	"heatherridge": {
		FacilityID:  9459,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/9459-the-golf-club-at-heather-ridge/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/9459/search",
		DisplayName: "Heather Ridge",
		City: "Aurora", State: "CO",
	},
	"heritageeaglebend": {
		FacilityID:  1019,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1019-heritage-eagle-bend-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1019/search",
		DisplayName: "Heritage Eagle Bend",
		City: "Aurora", State: "CO",
	},
	// Salt Lake City Metro
	"bonneville": {
		FacilityID:  2442,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2442-bonneville-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2442/search",
		DisplayName: "Bonneville",
		City: "Salt Lake City", State: "UT",
	},
	"forestdale": {
		FacilityID:  2443,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2443-forest-dale-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2443/search",
		DisplayName: "Forest Dale",
		City: "Salt Lake City", State: "UT",
	},
	"glendaleslc": {
		FacilityID:  2339,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2339-glendale-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2339/search",
		DisplayName: "Glendale",
		City: "Salt Lake City", State: "UT",
	},
	"roseparkslc": {
		FacilityID:  2448,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2448-rose-park-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2448/search",
		DisplayName: "Rose Park",
		City: "Salt Lake City", State: "UT",
	},
	"nibleypark": {
		FacilityID:  2447,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2447-nibley-park-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2447/search",
		DisplayName: "Nibley Park",
		City: "Salt Lake City", State: "UT",
	},
	"riveroaks": {
		FacilityID:  2342,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2342-river-oaks-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2342/search",
		DisplayName: "River Oaks",
		City: "Sandy", State: "UT",
	},
	"eaglewood": {
		FacilityID:  2341,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2341-eaglewood/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2341/search",
		DisplayName: "Eaglewood",
		City: "North Salt Lake", State: "UT",
	},
	"ridgeslc": {
		FacilityID:  12453,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/12453-the-ridge-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/12453/search",
		DisplayName: "The Ridge",
		City: "West Valley City", State: "UT",
	},
	"stonebridgeslc": {
		FacilityID:  12452,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/12452-stonebridge-golf-club-at-lake-park/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/12452/search",
		DisplayName: "Stonebridge",
		City: "West Valley City", State: "UT",
	},
	"thanksgivingpoint": {
		FacilityID:  6421,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6421-thanksgiving-point-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/6421/search",
		DisplayName: "Thanksgiving Point",
		City: "Lehi", State: "UT",
	},
	// ===== Las Vegas Metro =====
	"lasvegasgolfclub": {
		FacilityID:  1293,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1293-las-vegas-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1293-las-vegas-golf-club/search",
		DisplayName: "Las Vegas Golf Club",
		City: "Las Vegas", State: "NV",
	},
	"lasvegasnational": {
		FacilityID:  763,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/763-las-vegas-national-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/763-las-vegas-national-golf-course/search",
		DisplayName: "Las Vegas National",
		City: "Las Vegas", State: "NV",
	},
	"aliante": {
		FacilityID:  377,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/377-aliante-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/377-aliante-golf-club/search",
		DisplayName: "Aliante Golf Club",
		City: "North Las Vegas", State: "NV",
	},
	"rhodesranch": {
		FacilityID:  762,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/762-rhodes-ranch-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/762-rhodes-ranch-golf-club/search",
		DisplayName: "Rhodes Ranch",
		City: "Las Vegas", State: "NV",
	},
	"angelparkmountain": {
		FacilityID:  246,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/246-angel-park-golf-club-mountain-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/246-angel-park-golf-club-mountain-course/search",
		DisplayName: "Angel Park Mountain",
		City: "Las Vegas", State: "NV",
	},
	"angelparkpalm": {
		FacilityID:  6390,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6390-angel-park-golf-club-palm-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/6390-angel-park-golf-club-palm-course/search",
		DisplayName: "Angel Park Palm",
		City: "Las Vegas", State: "NV",
	},
	"angelparkcloudnine": {
		FacilityID:  9960,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/9960-angel-park-golf-club-cloud-nine/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/9960-angel-park-golf-club-cloud-nine/search",
		DisplayName: "Angel Park Cloud Nine",
		City: "Las Vegas", State: "NV",
	},
	"legacylv": {
		FacilityID:  247,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/247-the-legacy-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/247/search",
		DisplayName: "The Legacy",
		City: "Henderson", State: "NV",
	},
	"arroyoredrock": {
		FacilityID:  371,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/371-arroyo-golf-club-at-red-rock/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/371-arroyo-golf-club-at-red-rock/search",
		DisplayName: "Arroyo at Red Rock",
		City: "Las Vegas", State: "NV",
	},
	"losprados": {
		FacilityID:  1920,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1920-los-prados-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1920-los-prados-golf-course/search",
		DisplayName: "Los Prados",
		City: "Las Vegas", State: "NV",
	},
	"tpclasvegas": {
		FacilityID:  3508,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/3508-tpc-las-vegas-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/3508/search",
		DisplayName: "TPC Las Vegas",
		City: "Las Vegas", State: "NV",
	},
	"desertwillow": {
		FacilityID:  1336,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1336-desert-willow-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1336-desert-willow-golf-course/search",
		DisplayName: "Desert Willow",
		City: "Henderson", State: "NV",
	},
	"wildhorse": {
		FacilityID:  1111,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1111-wildhorse-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1111-wildhorse-golf-course/search",
		DisplayName: "Wildhorse",
		City: "Henderson", State: "NV",
	},
	"bouldercity": {
		FacilityID:  463,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/463-boulder-city-golf-course/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/463-boulder-city-golf-course/search",
		DisplayName: "Boulder City GC",
		City: "Boulder City", State: "NV",
	},
	"bouldercreek": {
		FacilityID:  157,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/157-boulder-creek-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/157-boulder-creek-golf-club/search",
		DisplayName: "Boulder Creek",
		City: "Boulder City", State: "NV",
	},
	"reverelexington": {
		FacilityID:  188,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/188-the-revere-golf-club-lexington/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/188/search",
		DisplayName: "Revere Lexington",
		City: "Henderson", State: "NV",
	},
	"revereconcord": {
		FacilityID:  4241,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/4241-the-revere-golf-club-concord/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/4241/search",
		DisplayName: "Revere Concord",
		City: "Henderson", State: "NV",
	},
	"reflectionbay": {
		FacilityID:  881,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/881-reflection-bay-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/881-reflection-bay-golf-club/search",
		DisplayName: "Reflection Bay",
		City: "Henderson", State: "NV",
	},
	"palmvalleylv": {
		FacilityID:  2074,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2074-palm-valley-golf-course-at-golf-summerlin/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2074/search",
		DisplayName: "Palm Valley",
		City: "Las Vegas", State: "NV",
	},
	"highlandfalls": {
		FacilityID:  1972,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/1972-highland-falls-golf-club-at-golf-summerlin/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/1972/search",
		DisplayName: "Highland Falls",
		City: "Las Vegas", State: "NV",
	},
	"eaglecrestlv": {
		FacilityID:  2075,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/2075-eagle-crest-golf-course-at-golf-summerlin/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/2075/search",
		DisplayName: "Eagle Crest",
		City: "Las Vegas", State: "NV",
	},
	"bearsbest": {
		FacilityID:  4995,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/4995-bears-best-las-vegas/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/4995-bears-best-las-vegas/search",
		DisplayName: "Bear's Best",
		City: "Las Vegas", State: "NV",
	},
	"chimera": {
		FacilityID:  761,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/761-chimera-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/761-chimera-golf-club/search",
		DisplayName: "Chimera Golf Club",
		City: "Henderson", State: "NV",
	},
	"siena": {
		FacilityID:  370,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/370-siena-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/370-siena-golf-club/search",
		DisplayName: "Siena Golf Club",
		City: "Las Vegas", State: "NV",
	},
	"painteddesert": {
		FacilityID:  764,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/764-painted-desert-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/764-painted-desert-golf-club/search",
		DisplayName: "Painted Desert",
		City: "Las Vegas", State: "NV",
	},
	"desertpines": {
		FacilityID:  5380,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/5380-desert-pines-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/5380-desert-pines-golf-club/search",
		DisplayName: "Desert Pines",
		City: "Las Vegas", State: "NV",
	},
	"balihai": {
		FacilityID:  5359,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/5359-bali-hai-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/5359-bali-hai-golf-club/search",
		DisplayName: "Bali Hai",
		City: "Las Vegas", State: "NV",
	},
	"stallionmountain": {
		FacilityID:  5,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/5-stallion-mountain-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/5-stallion-mountain-golf-club/search",
		DisplayName: "Stallion Mountain",
		City: "Las Vegas", State: "NV",
	},
	"royallinks": {
		FacilityID:  5383,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/5383-royal-links-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/5383-royal-links-golf-club/search",
		DisplayName: "Royal Links",
		City: "Las Vegas", State: "NV",
	},
	"paiutesun": {
		FacilityID:  6331,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6331-paiute-golf-resort-sun/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/6331-paiute-golf-resort-sun/search",
		DisplayName: "Paiute Sun Mountain",
		City: "Las Vegas", State: "NV",
	},
	"paiutesnow": {
		FacilityID:  132,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/132-paiute-golf-resort-snow/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/132-paiute-golf-resort-snow/search",
		DisplayName: "Paiute Snow Mountain",
		City: "Las Vegas", State: "NV",
	},
	"paiutewolf": {
		FacilityID:  6332,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6332-paiute-golf-resort-wolf/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/6332-paiute-golf-resort-wolf/search",
		DisplayName: "Paiute Wolf",
		City: "Las Vegas", State: "NV",
	},
	"coyotesprings": {
		FacilityID:  6204,
		SearchURL:   "https://www.golfnow.com/tee-times/facility/6204-coyote-springs-golf-club/search",
		BookingURL:  "https://www.golfnow.com/tee-times/facility/6204-coyote-springs-golf-club/search",
		DisplayName: "Coyote Springs",
		City: "Moapa", State: "NV",
	},
}

type GolfNowSearchRequest struct {
	Radius                     int     `json:"Radius"`
	Latitude                   float64 `json:"Latitude"`
	Longitude                  float64 `json:"Longitude"`
	PageSize                   int     `json:"PageSize"`
	PageNumber                 int     `json:"PageNumber"`
	SearchType                 int     `json:"SearchType"`
	SortBy                     string  `json:"SortBy"`
	SortDirection              int     `json:"SortDirection"`
	Date                       string  `json:"Date"`
	BestDealsOnly              bool    `json:"BestDealsOnly"`
	PriceMin                   string  `json:"PriceMin"`
	PriceMax                   string  `json:"PriceMax"`
	Players                    string  `json:"Players"`
	TimePeriod                 string  `json:"TimePeriod"`
	Holes                      string  `json:"Holes"`
	FacilityType               int     `json:"FacilityType"`
	RateType                   string  `json:"RateType"`
	TimeMin                    string  `json:"TimeMin"`
	TimeMax                    string  `json:"TimeMax"`
	FacilityId                 int     `json:"FacilityId"`
	SortByRollup               string  `json:"SortByRollup"`
	View                       string  `json:"View"`
	ExcludeFeaturedFacilities  bool    `json:"ExcludeFeaturedFacilities"`
	TeeTimeCount               int     `json:"TeeTimeCount"`
	PromotedCampaignsOnly      bool    `json:"PromotedCampaignsOnly"`
}

type GolfNowResponse struct {
	TTResults GolfNowResults `json:"ttResults"`
	Total     int            `json:"total"`
}

type GolfNowResults struct {
	TeeTimes []GolfNowTeeTime `json:"teeTimes"`
}

type GolfNowTeeTime struct {
	Time                  string             `json:"time"`
	FormattedTime         string             `json:"formattedTime"`
	FormattedTimeMeridian string             `json:"formattedTimeMeridian"`
	DisplayRate           float64            `json:"displayRate"`
	MultipleHolesRate     json.Number        `json:"multipleHolesRate"`
	PlayerRule            int                `json:"playerRule"`
	Facility              GolfNowFacility    `json:"facility"`
	TeeTimeRates          []GolfNowRate      `json:"teeTimeRates"`
}

type GolfNowFacility struct {
	FacilityId int    `json:"facilityId"`
	Name       string `json:"name"`
}

type GolfNowRate struct {
	HoleCount int    `json:"holeCount"`
	RateName  string `json:"rateName"`
}

func formatGolfNowDate(date string) string {
	var t time.Time
	var err error
	t, err = time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Jan 02 2006")
}

func getVerificationToken(facilityURL string) (string, error) {
	var client http.Client
	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", facilityURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var html string = string(body)

	// Look for the token in a meta tag or hidden input
	var re *regexp.Regexp = regexp.MustCompile(`__RequestVerificationToken[^>]*value="([^"]+)"`)
	var matches []string = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Also try meta tag format
	re = regexp.MustCompile(`name="__RequestVerificationToken"[^>]*content="([^"]+)"`)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Try data attribute format
	re = regexp.MustCompile(`data-request-verification-token="([^"]+)"`)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("verification token not found")
}

func fetchGolfNow(config GolfNowCourseConfig, date string) ([]DisplayTeeTime, error) {
	// Step 1: Get verification token
	var token string
	var err error
	token, err = getVerificationToken(config.SearchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	// Step 2: Search for tee times
	var searchDate string = formatGolfNowDate(date)
	var reqBody GolfNowSearchRequest = GolfNowSearchRequest{
		Radius:                    35,
		Latitude:                  39.6855,
		Longitude:                 -104.7076,
		PageSize:                  50,
		PageNumber:                0,
		SearchType:                1,
		SortBy:                    "Date",
		SortDirection:             0,
		Date:                      searchDate,
		BestDealsOnly:             false,
		PriceMin:                  "0",
		PriceMax:                  "10000",
		Players:                   "0",
		TimePeriod:                "3",
		Holes:                     "3",
		FacilityType:              0,
		RateType:                  "all",
		TimeMin:                   "0",
		TimeMax:                   "48",
		FacilityId:                config.FacilityID,
		SortByRollup:              "Date.MinDate",
		View:                      "Grouping",
		ExcludeFeaturedFacilities: true,
		TeeTimeCount:              50,
		PromotedCampaignsOnly:     false,
	}

	var jsonData []byte
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", "https://www.golfnow.com/api/tee-times/tee-time-results", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://www.golfnow.com")
	req.Header.Set("Referer", config.SearchURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("__requestverificationtoken", token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

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

	// Check for HTML response (Cloudflare block)
	if len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("blocked by bot protection")
	}

	var data GolfNowResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var results []DisplayTeeTime
	for _, tt := range data.TTResults.TeeTimes {
		var timeStr string = tt.FormattedTime + " " + tt.FormattedTimeMeridian

		var holes string = tt.MultipleHolesRate.String()

		// playerRule is a bitmask: bit 0 = 1 player, bit 1 = 2 players, etc.
		// Highest set bit + 1 = max openings
		var openings int = 0
		var rule int = tt.PlayerRule
		for rule > 0 {
			openings++
			rule = rule >> 1
		}

		results = append(results, DisplayTeeTime{
			Time:       timeStr,
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   openings,
			Holes:      holes,
			Price:      tt.DisplayRate,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
