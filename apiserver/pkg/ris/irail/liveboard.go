package irail

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

const API_URL = "https://api.irail.be"
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

type Departure struct {
	ID          string `json:"id"`
	Station     string `json:"station"`
	Stationinfo struct {
		ID           string `json:"@id"`
		ID0          string `json:"id"`
		Name         string `json:"name"`
		LocationX    string `json:"locationX"`
		LocationY    string `json:"locationY"`
		Standardname string `json:"standardname"`
	} `json:"stationinfo"`
	Time        string `json:"time"`
	Delay       string `json:"delay"`
	Canceled    string `json:"canceled"`
	Left        string `json:"left"`
	IsExtra     string `json:"isExtra"`
	Vehicle     string `json:"vehicle"`
	Vehicleinfo struct {
		Name      string `json:"name"`
		Shortname string `json:"shortname"`
		Number    string `json:"number"`
		Type      string `json:"type"`
		LocationX string `json:"locationX"`
		LocationY string `json:"locationY"`
		ID        string `json:"@id"`
	} `json:"vehicleinfo"`
	Platform     string `json:"platform"`
	Platforminfo struct {
		Name   string `json:"name"`
		Normal string `json:"normal"`
	} `json:"platforminfo"`
	Occupancy struct {
		ID   string `json:"@id"`
		Name string `json:"name"`
	} `json:"occupancy"`
	DepartureConnection string `json:"departureConnection"`
}

type Liveboard struct {
	Version     string `json:"version"`
	Timestamp   string `json:"timestamp"`
	Station     string `json:"station"`
	Stationinfo struct {
		ID           string `json:"@id"`
		ID0          string `json:"id"`
		Name         string `json:"name"`
		LocationX    string `json:"locationX"`
		LocationY    string `json:"locationY"`
		Standardname string `json:"standardname"`
	} `json:"stationinfo"`
	Departures struct {
		Number    string      `json:"number"`
		Departure []Departure `json:"departure"`
	} `json:"departures"`
}

func GetLiveboard(station, arriveOrDeparture, lang string, from time.Time) (Liveboard, error) {
	cacheName := fmt.Sprintf("%s-%s-%s-%d", station, arriveOrDeparture, lang, from.Unix())
	liveboardCacheMutex.RLock()
	if liveboard, ok := liveboardCache[cacheName]; ok {
		liveboardCacheMutex.RUnlock()
		return liveboard, nil
	}
	liveboardCacheMutex.RUnlock()

	tz, _ := time.LoadLocation("Europe/Brussels")
	from = from.In(tz)
	date := from.Format("02012006")
	time := from.Format("1504")

	url := fmt.Sprintf("%s/liveboard/?id=BE.NMBS.%s&arrdep=%s&lang=%s&format=json&alerts=false&date=%s&time=%s", API_URL, station, arriveOrDeparture, lang, date, time)
	log.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Liveboard{}, err
	}
	req.Header.Set("User-Agent", USER_AGENT)

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
	liveboardCache[cacheName] = liveboard
	liveboardCacheMutex.Unlock()

	return liveboard, nil
}

func LiveboardToRISDepartures(station, lang string) ([]ris.Departure, error) {
	out := []ris.Departure{}
	var liveboard Liveboard
	var sncbDepartures []Departure
	fromTime := time.Now()
	nilAttempts := 0

	for len(sncbDepartures) < 30 {
		resp, err := GetLiveboard(station, "departures", lang, fromTime)
		if err != nil {
			return nil, err
		}

		for _, dep := range resp.Departures.Departure {
			if len(sncbDepartures) > 0 && sncbDepartures[len(sncbDepartures)-1].Vehicle == dep.Vehicle {
				continue
			}
			sncbDepartures = append(sncbDepartures, dep)
		}
		liveboard = resp

		if len(sncbDepartures) > 0 {
			if len(resp.Departures.Departure) == 0 {
				fromTime = fromTime.Add(1 * time.Hour)
				nilAttempts++
				if nilAttempts > 10 {
					break
				}
			} else {
				newFromTime := unixTimeToTime(sncbDepartures[len(sncbDepartures)-1].Time)
				if newFromTime == fromTime { // we are at the end of the day
					newFromTime = newFromTime.Add(1 * time.Hour)
				}
				fromTime = newFromTime
			}
		}
	}

	for _, departure := range sncbDepartures {
		departureTime := unixTimeToTime(departure.Time)

		vehicle, err := GetVehicleCached(departure.Vehicleinfo.ID, "nl", departureTime)
		if err != nil {
			return nil, err
		}

		platformNormal := departure.Platforminfo.Name
		if departure.Platforminfo.Normal != "1" {
			platformNormal = "0"
		}

		delay := mustParseInt(departure.Delay)
		timeType := "SCHEDULE"
		if delay > 0 {
			timeType = "PREVIEW"
		}

		transportType := "HIGH_SPEED_TRAIN"
		transportNumber := 0
		sncbShortname := strings.Split(departure.Vehicleinfo.Shortname, " ")
		transportName := sncbShortname[0]
		if len(sncbShortname) > 1 {
			transportNumber = mustParseInt(sncbShortname[1])
		}
		if strings.HasPrefix(transportName, "S") || strings.HasPrefix(transportName, "L") {
			transportType = "REGIONAL_TRAIN"
		}
		if strings.HasPrefix(transportName, "BUS") {
			transportType = "BUS"
		}

		stops := []ris.StopPlace{}
		vias := []ris.Via{}
		foundCurrentStop := false
		viaCount := 0
		for _, stop := range vehicle.Stops.Stop {
			if stop.Station == liveboard.Station {
				foundCurrentStop = true
				continue
			}
			if !foundCurrentStop {
				continue
			}

			stops = append(stops, ris.StopPlace{
				Name:      stop.Station,
				EvaNumber: stop.Stationinfo.ID,
			})

			vias = append(vias, ris.Via{
				Name:            stop.Station,
				EvaNumber:       stop.Stationinfo.ID,
				DisplayPriority: viaCount,
			})
			viaCount++

		}

		out = append(out, ris.Departure{
			Station: ris.Station{
				EvaNumber: liveboard.Stationinfo.ID,
				Name:      liveboard.Stationinfo.Name,
			},
			JourneyID:        departure.ID,
			DepartureID:      departure.ID,
			TimeSchedule:     departureTime,
			Time:             departureTime.Add(time.Duration(delay) * time.Second),
			TimeType:         timeType,
			Platform:         departure.Platforminfo.Name,
			PlatformSchedule: platformNormal,
			Administration: ris.Administration{
				AdministrationID: "80",
				OperatorCode:     "---",
				OperatorName:     "NMBS",
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
				Category:  transportName,
				Number:    transportNumber,
				Label:     "",
				JourneyID: departure.ID,
				Direction: ris.Direction{
					Text:       departure.Station,
					StopPlaces: stops,
				},
				Destination: ris.Destination{
					EvaNumber: departure.Stationinfo.ID,
					Name:      departure.Stationinfo.Name,
					Canceled:  departure.Canceled == "1",
				},
				Via: vias,
			},
		})
	}

	return out, nil
}

func unixTimeToTime(in string) time.Time {
	i, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(i, 0)
}

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
