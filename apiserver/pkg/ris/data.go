package ris

import "time"

type DeparturesResponse struct {
	Departures  []Departure `json:"departures"`
	Disruptions []any       `json:"disruptions"`
}

type Station struct {
	EvaNumber string `json:"evaNumber"`
	Name      string `json:"name"`
}

type Transport struct {
	Type                 string      `json:"type"`
	Category             string      `json:"category"`
	Number               int         `json:"number"`
	Line                 any         `json:"line"`
	Label                string      `json:"label"`
	ReplacementTransport any         `json:"replacementTransport"`
	Direction            Direction   `json:"direction"`
	JourneyID            string      `json:"journeyID"`
	Destination          Destination `json:"destination"`
	DifferingDestination any         `json:"differingDestination"`
	Via                  []Via       `json:"via"`
}

type StopPlace struct {
	EvaNumber string `json:"evaNumber"`
	Name      string `json:"name"`
}

type Direction struct {
	Text       any         `json:"text"`
	StopPlaces []StopPlace `json:"stopPlaces"`
}

type Destination struct {
	EvaNumber string `json:"evaNumber"`
	Name      string `json:"name"`
	Canceled  bool   `json:"canceled"`
}

type Via struct {
	EvaNumber       string `json:"evaNumber"`
	Name            string `json:"name"`
	Canceled        bool   `json:"canceled"`
	Additional      bool   `json:"additional"`
	DisplayPriority int    `json:"displayPriority"`
}

type Administration struct {
	AdministrationID string `json:"administrationID"`
	OperatorCode     string `json:"operatorCode"`
	OperatorName     string `json:"operatorName"`
}

type Message struct {
	Code            string `json:"code"`
	Type            string `json:"type"`
	DisplayPriority any    `json:"displayPriority"`
	Category        any    `json:"category"`
	Text            string `json:"text"`
	TextShort       any    `json:"textShort"`
}

type Attribute struct {
	DisplayPriority       any    `json:"displayPriority"`
	DisplayPriorityDetail any    `json:"displayPriorityDetail"`
	Code                  string `json:"code"`
	Text                  string `json:"text"`
}

type Departure struct {
	Station           Station        `json:"station"`
	JourneyID         string         `json:"journeyID"`
	TimeSchedule      time.Time      `json:"timeSchedule"`
	TimeType          string         `json:"timeType"`
	Time              time.Time      `json:"time"`
	OnDemand          bool           `json:"onDemand"`
	PlatformSchedule  string         `json:"platformSchedule"`
	Platform          string         `json:"platform"`
	Administration    Administration `json:"administration"`
	Messages          []Message      `json:"messages"`
	Disruptions       []any          `json:"disruptions"`
	Attributes        []Attribute    `json:"attributes"`
	DepartureID       string         `json:"departureID"`
	Transport         Transport      `json:"transport"`
	JourneyType       string         `json:"journeyType"`
	Additional        bool           `json:"additional"`
	Canceled          bool           `json:"canceled"`
	ReliefFor         []any          `json:"reliefFor"`
	ReliefBy          []any          `json:"reliefBy"`
	ReplacementFor    []any          `json:"replacementFor"`
	ReplacedBy        []any          `json:"replacedBy"`
	ContinuationBy    any            `json:"continuationBy"`
	TravelsWith       []any          `json:"travelsWith"`
	Codeshares        []any          `json:"codeshares"`
	FutureDisruptions bool           `json:"futureDisruptions"`
}
