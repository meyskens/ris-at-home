package delijn

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meyskens/ris-at-home/apiserver/pkg/ris"
)

const API_URL = "https://api.delijn.be"
const USER_AGENT = "RIS-At-Home/1"

var liveboardCache = make(map[string]Liveboard)
var liveboardCacheMutex sync.RWMutex

func init() {
	// clear liveboard cache after 5 minutes
	go func() {
		for {
			liveboardCacheMutex.Lock()
			liveboardCache = make(map[string]Liveboard)
			liveboardCacheMutex.Unlock()
			time.Sleep(5 * time.Minute)
		}
	}()
}

type Line struct {
	ID            string `json:"id"`
	DirectionCode string `json:"directionCode"`
	Line          struct {
		ID            string `json:"id"`
		PublicLineNr  string `json:"publicLineNr"`
		Description   string `json:"description"`
		TransportType string `json:"transportType"`
		ServiceType   string `json:"serviceType"`
		LineColor     struct {
			Foreground       string `json:"foreground"`
			ForegroundBorder string `json:"foregroundBorder"`
			Background       string `json:"background"`
			BackgroundBorder string `json:"backgroundBorder"`
		} `json:"lineColor"`
	} `json:"line"`
}

type Liveboard struct {
	Trips []struct {
		ID                  string `json:"id"`
		Nr                  string `json:"nr"`
		PlanningDestination string `json:"planningDestination"`
		PlaceDestination    string `json:"placeDestination"`
		PatternID           string `json:"patternId"`
		PatternIDOriginal   string `json:"patternIdOriginal"`
		Passages            []struct {
			VisitNr        int `json:"visitNr"`
			PlannedPassage struct {
				ArrivalDateTime   string `json:"arrivalDateTime"`
				DepartureDateTime string `json:"departureDateTime"`
				ArrivalEpoch      int64  `json:"arrivalEpoch"`
				DepartureEpoch    int64  `json:"departureEpoch"`
			} `json:"plannedPassage"`
			RealtimePassage struct {
				ArrivalDateTime   string `json:"arrivalDateTime"`
				DepartureDateTime string `json:"departureDateTime"`
				ArrivalEpoch      int64  `json:"arrivalEpoch"`
				DepartureEpoch    int64  `json:"departureEpoch"`
			} `json:"realtimePassage"`
			ScheduleType string `json:"scheduleType"`
		} `json:"passages"`
		LineDirection struct {
			ID            string `json:"id"`
			DirectionCode string `json:"directionCode"`
			Line          struct {
				ID string `json:"id"`
			} `json:"line"`
		} `json:"lineDirection"`
		ExploitationDate string `json:"exploitationDate"`
		DetourIds        []struct {
			NetworkeventIdentifier string `json:"networkeventIdentifier"`
			DetourIdentifier       string `json:"detourIdentifier"`
		} `json:"detourIds,omitempty"`
		TripStatus string `json:"tripStatus,omitempty"`
	} `json:"trips"`
	ServedLineDirections []Line `json:"servedLineDirections"`
}

func GetLiveboard(stop string) (Liveboard, error) {
	liveboardCacheMutex.RLock()
	if liveboard, ok := liveboardCache[stop]; ok {
		liveboardCacheMutex.RUnlock()
		return liveboard, nil
	}
	liveboardCacheMutex.RUnlock()

	url := fmt.Sprintf("%s/travelinfo-trip/v1/stops/%s/trips", API_URL, stop)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Liveboard{}, err
	}
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Ocp-Apim-Subscription-Key", "2ebe6ee98dc14965b22c294c436c9ac0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Liveboard{}, err
	}
	defer resp.Body.Close()

	var liveboard Liveboard
	if err := json.NewDecoder(resp.Body).Decode(&liveboard); err != nil {
		return Liveboard{}, err
	}

	liveboardCacheMutex.Lock()
	liveboardCache[stop] = liveboard
	liveboardCacheMutex.Unlock()

	return liveboard, nil
}

func LiveboardToRISDepartures(station string) ([]ris.Departure, error) {
	out := []ris.Departure{}

	resp, err := GetLiveboard(station)
	if err != nil {
		return nil, err
	}

	lines := map[string]Line{}

	for _, line := range resp.ServedLineDirections {
		lines[line.ID] = line
	}

	for _, departure := range resp.Trips {
		departureTime, _ := time.Parse("2006-01-02T15:04:05-0700", departure.Passages[0].PlannedPassage.DepartureDateTime)

		realTimeDeparture := departureTime
		if departure.Passages[0].RealtimePassage.DepartureDateTime != "" {
			realTimeDeparture, _ = time.Parse("2006-01-02T15:04:05-0700", departure.Passages[0].RealtimePassage.DepartureDateTime)
		}

		if realTimeDeparture.Before(time.Now()) {
			continue
		}

		log.Println(departure)
		transportType := "BUS"
		log.Println(lines[departure.LineDirection.ID].Line.PublicLineNr)
		transportNumber := mustParseInt(lines[departure.LineDirection.ID].Line.PublicLineNr)

		stops := []ris.StopPlace{}
		vias := []ris.Via{}
		for _, stop := range strings.Split(lines[departure.LineDirection.ID].Line.Description, " - ") {
			stops = append(stops, ris.StopPlace{
				EvaNumber: stop,
				Name:      stop,
			})
			vias = append(vias, ris.Via{
				EvaNumber: stop,
				Name:      stop,
				Canceled:  departure.TripStatus == "CANCELLED",
			})
		}

		directionCode := mustParseInt(departure.LineDirection.DirectionCode) + 1

		out = append(out, ris.Departure{
			Station: ris.Station{
				EvaNumber: departure.PlaceDestination,
				Name:      "",
			},
			JourneyID:        departure.ID,
			DepartureID:      departure.ID,
			TimeSchedule:     departureTime,
			Time:             realTimeDeparture,
			TimeType:         "PREVIEW",
			Platform:         fmt.Sprintf("%d", directionCode),
			PlatformSchedule: fmt.Sprintf("%d", directionCode),
			Administration: ris.Administration{
				AdministrationID: "0",
				OperatorCode:     "---",
				OperatorName:     "De Lijn",
			},
			Disruptions:    []any{},
			Attributes:     []ris.Attribute{},
			Messages:       []ris.Message{},
			JourneyType:    "REGULAR",
			ReliefFor:      []any{},
			ReliefBy:       []any{},
			ReplacementFor: []any{},
			TravelsWith:    []any{},
			Codeshares:     []any{},
			Transport: ris.Transport{
				Type:      transportType,
				Category:  "BUS",
				Number:    transportNumber,
				Label:     "",
				JourneyID: departure.ID,
				Direction: ris.Direction{
					Text:       departure.PlaceDestination,
					StopPlaces: stops,
				},
				Destination: ris.Destination{
					EvaNumber: departure.PlaceDestination,
					Name:      departure.PlaceDestination,
					Canceled:  departure.TripStatus == "CANCELLED",
				},
				Via: vias,
			},
		})
	}

	return out, nil
}

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
