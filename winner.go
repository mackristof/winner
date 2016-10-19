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

type event struct {
	Id string `json:"id"`
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
	orgaId := os.Getenv("ORGA_ID")
	//fmt.Println("https://www.eventbriteapi.com/v3/events/search/?token=" + token + "&organizer.id=" + orgaId)
	resp, err := http.Get("https://www.eventbriteapi.com/v3/events/search/?token=" + token + "&organizer.id=" + orgaId)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	res := lastRequest{}
	json.Unmarshal(body, &res)
	//fmt.Printf("getting eventId from eventbrite %+v\n", res.Events[0].Id)
	var i = 1
	var result = []attendee{}
	for i != 0 {
		resp, err := http.Get("https://www.eventbriteapi.com/v3/events/" + res.Events[0].Id + "/attendees/?page=" + strconv.Itoa(i) + "&token=" + token)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		res := eventAttend{}
		json.Unmarshal(body, &res)
		//fmt.Printf("attendees count : %+v\n", len(res.Attendees))
		result = append(result, res.Attendees...)
		i = res.Pagination.PageCount - res.Pagination.PageNumber
	}
	nbWinnerS := r.URL.Query().Get("n")
	nbWinner, err := strconv.Atoi(nbWinnerS)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	if nbWinner < int(0) || nbWinner > len(result) {
		io.WriteString(w, "request < 0 or > "+strconv.Itoa(len(result)))
		return
	}
	//fmt.Printf("attendees final count : %+v\n", len(result))
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
	http.HandleFunc("/", winner)
	http.ListenAndServe(":8000", nil)
}
