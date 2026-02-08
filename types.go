package main

type DisplayTeeTime struct {
	Time       string  `json:"time"`
	Course     string  `json:"course"`
	Openings   int     `json:"openings"`
	Holes      string  `json:"holes"`
	Price      float64 `json:"price"`
	BookingURL string  `json:"bookingUrl"`
}

type Alert struct {
	ID        string `json:"id"`
	Phone     string `json:"phone"`
	Course    string `json:"course"`
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}
