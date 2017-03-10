package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
	"log"
)

const (
	options          string = "OPTIONS"
	allowOrigin      string = "Access-Control-Allow-Origin"
	allowMethods     string = "Access-Control-Allow-Methods"
	allowHeaders     string = "Access-Control-Allow-Headers"
	allowCredentials string = "Access-Control-Allow-Credentials"
	exposeHeaders    string = "Access-Control-Expose-Headers"
	credentials      string = "true"
	origin           string = "Origin"
	methods          string = "POST, GET, OPTIONS, PUT, DELETE, HEAD, PATCH"

	// If you want to expose some other headers add it here
	headers string = "Accept, Accept-Encoding, Authorization, Content-Length, Content-Type, X-CSRF-Token"
)

// Handler will allow cross-origin HTTP requests
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set allow origin to match origin of our request or fall back to *
		if o := r.Header.Get(origin); o != "" {
			w.Header().Set(allowOrigin, o)
		} else {
			w.Header().Set(allowOrigin, "*")
		}

		// Set other headers
		w.Header().Set(allowHeaders, headers)
		w.Header().Set(allowMethods, methods)
		w.Header().Set(allowCredentials, credentials)
		w.Header().Set(exposeHeaders, headers)

		// If this was preflight options request let's write empty ok response and return
		if r.Method == options {
			w.WriteHeader(http.StatusOK)
			w.Write(nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type event struct {
	ID string `json:"id"`
	Date start `json:"start"`
}

type start struct {
	Utc string `json:"utc"`
}

type lastRequest struct {
	Events []event `json:"events"`
}

type profile struct {
	LastName  string `json:"first_name"`
	FirstName string `json:"last_name"`
}

type attendee struct {
	Profile profile `json:"profile"`
}

type pagination struct {
	PageNumber int `json:"page_number"`
	PageCount  int `json:"page_count"`
}

type eventAttend struct {
	Attendees  []attendee `json:"attendees"`
	Pagination pagination `json:"pagination"`
}

func isPresent(index int, randoms []int) bool {
	for i := 0; i < len(randoms); i++ {
		if randoms[i] == index {
			return true
		}
	}
	return false
}

func getRandoms(nbWinner int) []int {
	count := nbWinner
	var randoms = []int{}
	for count != 0 {
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		index := r1.Intn(nbWinner)
		exist := isPresent(index, randoms)
		if !exist {
			count--
			randoms = append(randoms, index)
		}
	}
	return randoms
}

func getWinners(attendees []attendee, nbWinner int) []profile {
	var winnersProfile = []profile{}
	randoms := getRandoms(len(attendees))
	for nbWinner != 0 {
		winnersProfile = append(winnersProfile, attendees[randoms[nbWinner-1]].Profile)
		nbWinner--
	}
	return winnersProfile
}

func preFetchWinner(nbWinner int, result []attendee) (string, error) {
	winnersProfile := getWinners(result, nbWinner)
	podium, _ := json.Marshal(winnersProfile)
	return string(podium), nil
}

func winner(chanAttendees chan []attendee) func(http.ResponseWriter, *http.Request) {
	var attendees []attendee
	var winnerPreFetch = make([]string, 10)

	go func() {
		for {
			attendees = <- chanAttendees
			for i := 1; i < 10; i++ {
				winnerPreFetch[i], _ = preFetchWinner(i, attendees)
			}
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		nbWinnerS := r.URL.Query().Get("nb")
		if len(nbWinnerS) == 0 {
			http.Error(w, "bad Request", http.StatusBadRequest)
			return
		}
		nbWinner, err := strconv.Atoi(nbWinnerS)

		if err != nil {
			http.Error(w, "bad Request", http.StatusBadRequest)
			return
		}
		if nbWinner < int(0) {
			http.Error(w, "request < 0 ", http.StatusBadRequest)
			return
		}
		if nbWinner >= len(attendees) || nbWinner > len(winnerPreFetch) - 1 {
			n := nbWinner
			if nbWinner >= len(attendees) {
				n = len(attendees)
			}
			winnersProfile := getWinners(attendees, n)
			totalattendees, _ := json.Marshal(winnersProfile)
			io.WriteString(w, string(totalattendees))
		} else {
			io.WriteString(w, winnerPreFetch[nbWinner])
		}

		chanAttendees <- attendees
	}
}

func getLastEvent() (*string, *time.Time, error) {
	token := os.Getenv("TOKEN")
	orgaID := "1464915124"

	resp, err := http.Get("https://www.eventbriteapi.com/v3/events/search/?token=" + token + "&organizer.id=" + orgaID)
	if err != nil {
		return nil,nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil,nil, err
	}

	var request lastRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil,nil, err
	}

	if len(request.Events) == 0 {
		return nil,nil, errors.New("No event found")
	}

	date, err := time.Parse(time.RFC3339,request.Events[0].Date.Utc)
	log.Println(date)
	return &request.Events[0].ID, &date, nil
}

func getAttendees(lastEventID string) ([]attendee, error){
	token := os.Getenv("TOKEN")
	var i = 1
	var result = []attendee{}
	for {
		log.Println("Fetch attendees")
		log.Println("i:"+strconv.Itoa(i))
		resp, err := http.Get("https://www.eventbriteapi.com/v3/events/" + lastEventID + "/attendees/?page=" + strconv.Itoa(i) + "&token=" + token)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		res := eventAttend{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, err
		}
		result = append(result, res.Attendees...)
		log.Println("Page Count:" + strconv.Itoa(res.Pagination.PageCount))
		log.Println("PageNumber:" + strconv.Itoa(res.Pagination.PageNumber))

		if i == res.Pagination.PageCount {
			break
		}
		i++
	}
	return result, nil
}

func main() {
	mux := http.NewServeMux()

	lastEventId,lastEventDate, err := getLastEvent()
	if err != nil {
		log.Println(err)
	}

	log.Println("Event id: " + *lastEventId + " / Event date : "+ lastEventDate.String())

	newEventID := make(chan string)
	chanNewAttendees := make(chan []attendee)

	go func () {
		var eventID string
		for {
			var listAttendee []attendee
			var err error
			select {
			case n := <-newEventID:
				eventID = n
				listAttendee, err = getAttendees(eventID)
				if err != nil {
					log.Println(err)
				}
				log.Println("List attendees : ")
				log.Println(listAttendee)
			case <-time.After(1 * time.Hour):
				listAttendee, err = getAttendees(eventID)
				if err != nil {
					log.Println(err)
				}
				log.Println("List attendees : ")
				log.Println(listAttendee)
			}
			chanNewAttendees <- listAttendee
		}

	}()
	newEventID <- *lastEventId

	go func() {
		for range time.Tick(24 * time.Hour){
			newLastEventId,lastEventDate, err := getLastEvent()
			if err != nil {
				log.Println(err)
			}
			log.Println(*lastEventId + " "+ lastEventDate.String())
			if *newLastEventId != *lastEventId {
				newEventID <- *newLastEventId
			}
		}
	}()

	mux.HandleFunc("/winners", winner(chanNewAttendees))
	if err := http.ListenAndServe(":8000", cors(mux)); err != nil {
		log.Println(err)
	}
}