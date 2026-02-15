package platforms

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
)

type ResortSuiteCourseConfig struct {
	Key         string `json:"key"`
	Metro       string `json:"metro"`
	BaseURL     string `json:"baseUrl"`
	CourseID    string `json:"courseId"`
	DisplayName string `json:"displayName"`
	City        string `json:"city"`
	State       string `json:"state"`
	BookingURL  string `json:"bookingUrl"`
}

var ResortSuiteCourses = map[string]ResortSuiteCourseConfig{}

var (
	rsTeeTimeRe = regexp.MustCompile(`<TeeTime>(.*?)</TeeTime>`)
	rsTimeRe    = regexp.MustCompile(`<Time>([^<]+)</Time>`)
	rsSlotsRe   = regexp.MustCompile(`<SlotsAvailable>(\d+)</SlotsAvailable>`)
	rsPriceRe   = regexp.MustCompile(`<ItemType>Green Fee[s]?</ItemType>.*?<Price>([^<]+)</Price>`)
	rsPublicRe  = regexp.MustCompile(`<ItemName>[^<]*Public[^<]*</ItemName>.*?<Price>([^<]+)</Price>`)
)

const soapEnvelopeOpen = `<?xml version="1.0" encoding="UTF-8" ?>
<soapenv:Envelope xmlns:g="http://www.resortsuite.com/RSWS/v1/Golf/Types" xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
<soapenv:Body>`
const soapEnvelopeClose = `</soapenv:Body></soapenv:Envelope>`

func FetchResortSuite(config ResortSuiteCourseConfig, date string) ([]DisplayTeeTime, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Step 1: Establish session
	req, _ := http.NewRequest("GET", config.BaseURL+"/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ResortSuite %s session: %w", config.Key, err)
	}
	resp.Body.Close()

	// Step 2: Fetch tee sheet
	body := fmt.Sprintf(`%s<g:FetchGolfTeeSheetRequest><CourseId>%s</CourseId><Date>%s</Date><GroupCode>undefined</GroupCode><Version>2</Version><WebFolioId>0</WebFolioId></g:FetchGolfTeeSheetRequest>%s`,
		soapEnvelopeOpen, config.CourseID, date, soapEnvelopeClose)

	soapURL := config.BaseURL + "/wso2wsas/services/RSWS?action=FetchGolfTeeSheet"
	req, _ = http.NewRequest("POST", soapURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Set("SOAPAction", "FetchGolfTeeSheet")
	req.Header.Set("Accept", "application/xml, text/xml, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", config.BaseURL)
	req.Header.Set("Referer", config.BaseURL+"/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ResortSuite %s teesheet: %w", config.Key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil
	}

	raw, _ := io.ReadAll(resp.Body)
	xml := string(raw)

	matches := rsTeeTimeRe.FindAllStringSubmatch(xml, -1)
	var results []DisplayTeeTime
	for _, m := range matches {
		block := m[1]

		slotsMatch := rsSlotsRe.FindStringSubmatch(block)
		if slotsMatch == nil {
			continue
		}
		slots, _ := strconv.Atoi(slotsMatch[1])
		if slots == 0 {
			continue
		}

		timeMatch := rsTimeRe.FindStringSubmatch(block)
		if timeMatch == nil {
			continue
		}

		// Get public rate price if available, otherwise first green fee price
		var price float64
		publicMatch := rsPublicRe.FindStringSubmatch(block)
		if publicMatch != nil {
			price, _ = strconv.ParseFloat(publicMatch[1], 64)
		} else {
			priceMatch := rsPriceRe.FindStringSubmatch(block)
			if priceMatch != nil {
				price, _ = strconv.ParseFloat(priceMatch[1], 64)
			}
		}

		results = append(results, DisplayTeeTime{
			Time:       strings.TrimSpace(timeMatch[1]),
			Course:     config.DisplayName,
			City:       config.City,
			State:      config.State,
			Openings:   slots,
			Holes:      "18",
			Price:      price,
			BookingURL: config.BookingURL,
		})
	}

	return results, nil
}
