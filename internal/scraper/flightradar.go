package scraper

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

type flightInfo struct {
	Date       string `json:"Date"`
	From       string `json:"From"`
	To         string `json:"To"`
	Flight     string `json:"Flight"`
	FlightTime string `json:"FlightTime"`
	STD        string `json:"STD"`
	ATD        string `json:"ATD"`
	STA        string `json:"STA"`
	Status     string `json:"Status"`
}

type flightRadarInfo struct {
	Aircraft     string        `json:"Aircraft"`
	Airline      string        `json:"Airline"`
	Operator     string        `json:"Operator"`
	TypeCode     string        `json:"TypeCode"`
	AirlineCode  string        `json:"AirlineCode"`
	OperatorCode string        `json:"OperatorCode"`
	ModeS        string        `json:"ModeS"`
	Flights      []*flightInfo `json:"Flights"`
}

type flightRadarRes struct {
	Res *flightRadarInfo
	Err error
}

const frAircraftURL = "https://www.flightradar24.com/data/aircraft/"

func getFlightRadarStruct(q *Queries, done chan flightRadarRes) {
	URL := fmt.Sprintf("%s%s", frAircraftURL, q.Reg)
	b, err := fetchHTML(URL)
	if err != nil {
		result := flightRadarRes{Res: nil, Err: err}
		done <- result
		return
	}
	s := newScraper(b)

	var aircraft string
	var airline string
	var operator string
	var typeCode string
	var airlineCode string
	var operatorCode string
	var modeS string
	var flights []*flightInfo

	aircraftArr, err := s.fetchText("span", "details", 1)
	if err != nil {
		result := flightRadarRes{Res: nil, Err: err}
		done <- result
		return
	}
	aircraft = strings.TrimSpace(aircraftArr[0])

	err = s.advance("span", "details", 1)
	if err != nil {
		result := flightRadarRes{Res: nil, Err: err}
		done <- result
		return
	}
	airlineArr, err := s.fetchText("a", "", 1)
	if err != nil {
		result := flightRadarRes{Res: nil, Err: err}
		done <- result
		return
	}
	airline = strings.TrimSpace(airlineArr[0])

	res, err := s.fetchText("span", "details", 5)
	if err != nil {
		result := flightRadarRes{Res: nil, Err: err}
		done <- result
		return
	}
	operator = strings.TrimSpace(res[0])
	typeCode = strings.TrimSpace(res[1])
	airlineCode = strings.TrimSpace(res[2])
	operatorCode = strings.TrimSpace(res[3])
	modeS = strings.TrimSpace(res[4])

	fr := &flightRadarInfo{
		Aircraft:     aircraft,
		Airline:      airline,
		Operator:     operator,
		TypeCode:     typeCode,
		AirlineCode:  airlineCode,
		OperatorCode: operatorCode,
		ModeS:        modeS,
		Flights:      flights,
	}

	err = s.advance("td", "w40 hidden-xs hidden-sm", 3)
	if err != nil {
		log.Printf("flights: %v\n", err)
		result := flightRadarRes{Res: fr, Err: nil}
		done <- result
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < q.Flights; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			flight, err := getFlight(s)
			if err != nil {
				if err.Error() == "query not found" {
					return
				}
			}
			mu.Lock()
			flights = append(flights, flight)
			mu.Unlock()
		}()
	}
	wg.Wait()
	fr.Flights = flights

	result := flightRadarRes{Res: fr, Err: nil}
	done <- result
}

func getFlight(s *scraper) (*flightInfo, error) {
	var date string
	var from string
	var to string
	var flight string
	var flightTime string
	var std string
	var atd string
	var sta string
	var status string

	dateArr, err := s.fetchText("td", "hidden-xs hidden-sm", 1)
	if err != nil {
		return nil, err
	}
	date = strings.TrimSpace(dateArr[0])

	fromToArr, err := s.fetchText("td", "text-center-sm hidden-xs hidden-sm", 2)
	if err != nil {
		return nil, err
	}
	from = strings.TrimSpace(fromToArr[0])
	to = strings.TrimSpace(fromToArr[1])

	err = s.advance("td", "hidden-xs hidden-sm", 1)
	if err != nil {
		return nil, err
	}

	flightArr, err := s.fetchText("a", "fbold", 1)
	if err != nil {
		return nil, err
	}
	flight = strings.TrimSpace(flightArr[0])

	res, err := s.fetchText("td", "hidden-xs hidden-sm", 4)
	if err != nil {
		return nil, err
	}
	flightTime = strings.TrimSpace(res[0])
	std = strings.TrimSpace(res[1])
	atd = strings.TrimSpace(res[2])
	sta = strings.TrimSpace(res[3])

	statusArr, err := s.fetchText("td", "hidden-xs hidden-sm", 2)
	if err != nil {
		return nil, err
	}
	status = strings.TrimSpace(statusArr[1])

	f := &flightInfo{
		Date:       date,
		From:       from,
		To:         to,
		Flight:     flight,
		FlightTime: flightTime,
		STD:        std,
		ATD:        atd,
		STA:        sta,
		Status:     status,
	}
	return f, nil
}
