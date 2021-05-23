package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
)

// https://apisetu.gov.in/public/api/cowin
const (
	baseURL                     = "https://cdn-api.co-vin.in/api"
	calendarByPinURLFormat      = "/v2/appointment/sessions/public/calendarByPin?pincode=%s&date=%s"
	calendarByDistrictURLFormat = "/v2/appointment/sessions/public/calendarByDistrict?district_id=%d&date=%s"
	listStatesURLFormat         = "/v2/admin/location/states"
	listDistrictsURLFormat      = "/v2/admin/location/districts/%d"
)

var (
	stateID, districtID int
	lastAvailability1, lastAvailability2 string
)

type StateList struct {
	States []struct {
		StateID    int    `json:"state_id"`
		StateName  string `json:"state_name"`
		StateNameL string `json:"state_name_l"`
	} `json:"states"`
	TTL int `json:"ttl"`
}

type DistrictList struct {
	Districts []struct {
		StateID       int    `json:"state_id"`
		DistrictID    int    `json:"district_id"`
		DistrictName  string `json:"district_name"`
		DistrictNameL string `json:"district_name_l"`
	} `json:"districts"`
	TTL int `json:"ttl"`
}

type Appointments struct {
	Centers []struct {
		CenterID      int     `json:"center_id"`
		Name          string  `json:"name"`
		NameL         string  `json:"name_l"`
		StateName     string  `json:"state_name"`
		StateNameL    string  `json:"state_name_l"`
		DistrictName  string  `json:"district_name"`
		DistrictNameL string  `json:"district_name_l"`
		BlockName     string  `json:"block_name"`
		BlockNameL    string  `json:"block_name_l"`
		Pincode       int     `json:"pincode"`
		Lat           float64 `json:"lat"`
		Long          float64 `json:"long"`
		From          string  `json:"from"`
		To            string  `json:"to"`
		FeeType       string  `json:"fee_type"`
		VaccineFees   []struct {
			Vaccine string `json:"vaccine"`
			Fee     string `json:"fee"`
		} `json:"vaccine_fees"`
		Sessions []struct {
			SessionID         string   `json:"session_id"`
			Date              string   `json:"date"`
			AvailableCapacity float64  `json:"available_capacity"`
			AvailableCapDose1 float64  `json:"available_capacity_dose1"`
			AvailableCapDose2 float64  `json:"available_capacity_dose2"`
			MinAgeLimit       int      `json:"min_age_limit"`
			Vaccine           string   `json:"vaccine"`
			Slots             []string `json:"slots"`
		} `json:"sessions"`
	} `json:"centers"`
}

func timeNow() string {
	return time.Now().Format("02-01-2006")
}

func timeWeekAfter() string {
	return time.Now().AddDate(0,0,7).Format("02-01-2006")
}

func queryServer(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "hi_IN")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36 Edg/90.0.818.51")

	log.Print("Querying endpoint: ", baseURL+path)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//log.Print("Response: ", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		// Sometimes the API returns "Unauthenticated access!", do not fail in that case
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, nil
		}
		return nil, errors.New(fmt.Sprintf("Request failed with statusCode: %d", resp.StatusCode))
	}
	return bodyBytes, nil
}

func searchByPincode(dose int, pinCode string) error {
	response, err := queryServer(fmt.Sprintf(calendarByPinURLFormat, pinCode, timeNow()))
	response2, err2 := queryServer(fmt.Sprintf(calendarByDistrictURLFormat, districtID, timeWeekAfter()))
	if err != nil && err2 != nil {
		return errors.Wrap(err, "Failed to fetch appointment sessions")
	}

	getAvailableSessions(response2, dose, age, district, &lastAvailability1)

	return getAvailableSessions(response, dose, age, pinCode, &lastAvailability2)
}

func getStateIDByName(state string) (int, error) {
	response, err := queryServer(listStatesURLFormat)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to list states")
	}
	states := StateList{}
	if err := json.Unmarshal(response, &states); err != nil {
		return 0, err
	}
	for _, s := range states.States {
		if strings.ToLower(s.StateName) == strings.ToLower(state) {
			log.Printf("State Details - ID: %d, Name: %s", s.StateID, s.StateName)
			return s.StateID, nil
		}
	}
	return 0, errors.New("Invalid state name passed")
}

func getDistrictIDByName(stateID int, district string) (int, error) {
	response, err := queryServer(fmt.Sprintf(listDistrictsURLFormat, stateID))
	if err != nil {
		return 0, errors.Wrap(err, "Failed to list states")
	}
	dl := DistrictList{}
	if err := json.Unmarshal(response, &dl); err != nil {
		return 0, err
	}
	for _, d := range dl.Districts {
		if strings.ToLower(d.DistrictName) == strings.ToLower(district) {
			log.Printf("District Details - ID: %d, Name: %s", d.DistrictID, d.DistrictName)
			return d.DistrictID, nil
		}
	}
	return 0, errors.New("Invalid district name passed")
}

func searchByStateDistrict(dose int, age int, state, district string) error {
	var err1 error
	if stateID == 0 {
		stateID, err1 = getStateIDByName(state)
		if err1 != nil {
			return err1
		}
	}
	if districtID == 0 {
		districtID, err1 = getDistrictIDByName(stateID, district)
		if err1 != nil {
			return err1
		}
	}
	response, err := queryServer(fmt.Sprintf(calendarByDistrictURLFormat, districtID, timeNow()))
	response2, err2 := queryServer(fmt.Sprintf(calendarByDistrictURLFormat, districtID, timeWeekAfter()))

	if err != nil && err2 != nil {
		return errors.Wrap(err, "Failed to fetch appointment sessions")
	}

	getAvailableSessions(response2, dose, age, district, &lastAvailability1)

	return getAvailableSessions(response, dose, age, district, &lastAvailability2)
}

// isPreferredAvailable checks for availability of preferences
func isPreferredAvailable(current, preference string) bool {
	if preference == "" {
		return true
	} else {
		return strings.ToLower(current) == preference
	}
}

func getAvailableSessions(response []byte, dose int, age int, district string, lastAvailability *string) error {
	if response == nil {
		log.Printf("Received unexpected response, rechecking after %v seconds", interval)
		return nil
	}
	appnts := Appointments{}
	err := json.Unmarshal(response, &appnts)
	if err != nil {
		return err
	}
	var buf bytes.Buffer

	w := tabwriter.NewWriter(&buf, 1, 8, 1, '\t', 0)
	for _, center := range appnts.Centers {
		if !isPreferredAvailable(center.FeeType, fee) {
			continue
		}
		for _, s := range center.Sessions {

			if dose == 1 {
				if s.AvailableCapDose1 == 0 {
					continue
				}
			} else {
				if s.AvailableCapDose2 == 0 {
					continue
				}
			}

			if s.MinAgeLimit <= age && isPreferredAvailable(s.Vaccine, vaccine) {
				fmt.Fprintln(w, fmt.Sprintf("Center\t%s", center.Name))
				fmt.Fprintln(w, fmt.Sprintf("State\t%s", center.StateName))
				fmt.Fprintln(w, fmt.Sprintf("District\t%s", center.DistrictName))
				fmt.Fprintln(w, fmt.Sprintf("PinCode\t%d", center.Pincode))
				fmt.Fprintln(w, fmt.Sprintf("Fee\t%s", center.FeeType))
				if len(center.VaccineFees) != 0 {
					fmt.Fprintln(w, fmt.Sprintf("Vaccine\t"))
				}
				for _, v := range center.VaccineFees {
					fmt.Fprintln(w, fmt.Sprintf("\tName\t%s", v.Vaccine))
					fmt.Fprintln(w, fmt.Sprintf("\tFees\t%s", v.Fee))
				}
				fmt.Fprintln(w, fmt.Sprintf("Sessions\t"))
				fmt.Fprintln(w, fmt.Sprintf("\tDate\t%s", s.Date))

				if dose == 1 {
					fmt.Fprintln(w, fmt.Sprintf("\tAvailableCapacityDose1\t%f", s.AvailableCapDose1))
				} else {
					fmt.Fprintln(w, fmt.Sprintf("\tAvailableCapacityDose2\t%f", s.AvailableCapDose2))
				}

				fmt.Fprintln(w, fmt.Sprintf("\tMinAgeLimit\t%d", s.MinAgeLimit))
				fmt.Fprintln(w, fmt.Sprintf("\tVaccine\t%s", s.Vaccine))
				fmt.Fprintln(w, fmt.Sprintf("\tSlots"))
				for _, slot := range s.Slots {
					fmt.Fprintln(w, fmt.Sprintf("\t\t%s", slot))
				}
				fmt.Fprintln(w, "-----------------------------")
			}
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if buf.Len() == 0 {
		log.Printf("No slots available, rechecking after %v seconds", interval)
		return nil
	}

	var mailSendError error

	if(*lastAvailability != buf.String()){
	 log.Print("Found available slots, sending email")
	 //log.Print(buf.String())
	 *lastAvailability=buf.String()
	 mailSendError=sendMail(strconv.Itoa(int(age)), strconv.Itoa(int(dose)), district, email, password, buf.String())
	}else{
	 log.Print("Found available slots same as before, skip sending email")
	 *lastAvailability=buf.String()
	}

	return mailSendError;
	//return sendMail(strconv.Itoa(int(age)), strconv.Itoa(int(dose)), district, email, password, buf.String())
}
