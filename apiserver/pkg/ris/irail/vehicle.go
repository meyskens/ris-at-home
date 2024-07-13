package irail

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

var vehicleCacheMutex sync.RWMutex
var vehicleCache = make(map[string]Vehicle)

func init() {
	// clear Vehicle cache after 4 hours
	go func() {
		for {
			vehicleCacheMutex.Lock()
			vehicleCache = make(map[string]Vehicle)
			vehicleCacheMutex.Unlock()
			time.Sleep(4 * time.Hour)
		}
	}()
}

type Vehicle struct {
	Version     string `json:"version"`
	Timestamp   string `json:"timestamp"`
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
	Stops struct {
		Number string `json:"number"`
		Stop   []struct {
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
			Time         string `json:"time"`
			Platform     string `json:"platform"`
			Platforminfo struct {
				Name   string `json:"name"`
				Normal string `json:"normal"`
			} `json:"platforminfo"`
			ScheduledDepartureTime string `json:"scheduledDepartureTime"`
			ScheduledArrivalTime   string `json:"scheduledArrivalTime"`
			Delay                  string `json:"delay"`
			Canceled               string `json:"canceled"`
			DepartureDelay         string `json:"departureDelay"`
			DepartureCanceled      string `json:"departureCanceled"`
			ArrivalDelay           string `json:"arrivalDelay"`
			ArrivalCanceled        string `json:"arrivalCanceled"`
			Left                   string `json:"left"`
			Arrived                string `json:"arrived"`
			IsExtraStop            string `json:"isExtraStop"`
			Occupancy              struct {
				ID   string `json:"@id"`
				Name string `json:"name"`
			} `json:"occupancy,omitempty"`
			DepartureConnection string `json:"departureConnection,omitempty"`
		} `json:"stop"`
	} `json:"stops"`
}

func GetVehicleCached(id, lang string, date time.Time) (Vehicle, error) {
	dateString := date.Format("02012006")
	cacheName := id + dateString
	vehicleCacheMutex.RLock()
	if vehicle, ok := vehicleCache[cacheName]; ok {
		vehicleCacheMutex.RUnlock()
		return vehicle, nil
	}
	vehicleCacheMutex.RUnlock()

	url := API_URL + "/vehicle/?id=" + id + "&lang=" + lang + "&format=json&alerts=false&date=" + dateString
	log.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Vehicle{}, err
	}
	req.Header.Set("User-Agent", USER_AGENT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Vehicle{}, err
	}
	defer resp.Body.Close()

	var vehicle Vehicle
	if err := json.NewDecoder(resp.Body).Decode(&vehicle); err != nil {
		return Vehicle{}, err
	}

	vehicleCacheMutex.Lock()
	vehicleCache[cacheName] = vehicle
	vehicleCacheMutex.Unlock()

	return vehicle, nil
}
