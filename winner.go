package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
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
}

type lastRequest struct {
	Events []event `json:"events"`
}

type profile struct {
	LastName  string `json:"first_name"`
	FirstName string `json:"last_name"`
	Email     string `json:"email"`
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

func winner(w http.ResponseWriter, r *http.Request) {

	token := os.Getenv("TOKEN")
	orgaID := os.Getenv("ORGA_ID")

	resp, err := http.Get("https://www.eventbriteapi.com/v3/events/search/?token=" + token + "&organizer.id=" + orgaID)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	res := lastRequest{}
	json.Unmarshal(body, &res)
	if len(res.Events) == 0 {
		io.WriteString(w, "no event available")
		return
	}
	var i = 1
	var result = []attendee{}
	for i != 0 {
		resp, err := http.Get("https://www.eventbriteapi.com/v3/events/" + res.Events[0].ID + "/attendees/?page=" + strconv.Itoa(i) + "&token=" + token)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		res := eventAttend{}
		json.Unmarshal(body, &res)
		result = append(result, res.Attendees...)
		i = res.Pagination.PageCount - res.Pagination.PageNumber
	}
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
	if nbWinner < int(0) || nbWinner > len(result) {
		http.Error(w, "request < 0 or > "+strconv.Itoa(len(result)), http.StatusBadRequest)
		return
	}
	var winners = []profile{}
	for nbWinner != 0 {
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		index := r1.Intn(len(result))
		winners = append(winners, result[index].Profile)
		nbWinner--
	}

	winner, _ := json.Marshal(winners)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(winner))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/winners", winner)
	http.ListenAndServe(":8000", cors(mux))
}
