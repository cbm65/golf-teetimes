package platforms

import "time"

const PlatformTimeout = 15 * time.Second

type DisplayTeeTime struct {
	Time       string  `json:"time"`
	Course     string  `json:"course"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	Openings   int     `json:"openings"`
	Holes      string  `json:"holes"`
	Price      float64 `json:"price"`
	BookingURL string  `json:"bookingUrl"`
}

type Alert struct {
	ID         string `json:"id"`
	Phone      string `json:"phone"`
	Course     string `json:"course"`
	Date       string `json:"date"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	MinPlayers int    `json:"minPlayers,omitempty"`
	Holes      string `json:"holes,omitempty"`
	Active     bool   `json:"active"`
	CreatedAt  string `json:"createdAt"`
	ConsentAt  string `json:"consentAt"`
}
