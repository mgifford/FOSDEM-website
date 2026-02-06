package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
)

type Schedule struct {
	Conference Conference `json:"conference"`
}

type Conference struct {
	Title  string  `json:"title"`
	Rooms  []Room  `json:"rooms"`
	Tracks []Track `json:"tracks"`
	Days   []Day   `json:"days"`
}

type Room struct {
}

type Event struct {
	GUID    string   `json:"guid"`
	Persons []Person `json:"persons"`
}

type Person struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

type Track struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

type Day struct {
	Rooms map[string][]Event `json:"rooms"`
}

var (
	//go:embed data/schedule.json
	rawSchedule []byte
	schedule    Schedule
)

func init() {
	s := struct {
		*Schedule `json:"schedule"`
	}{
		Schedule: &schedule,
	}
	if err := json.NewDecoder(bytes.NewReader(rawSchedule)).Decode(&s); err != nil {
		panic(err)
	}
}

func main() {
	speakers()
	events()
	devRooms()
}

func events() {
	var (
		events  []Event
		visited = make(map[string]struct{})
	)
	for _, day := range schedule.Conference.Days {
		for _, rooms := range day.Rooms {
			for _, event := range rooms {
				if _, ok := visited[event.GUID]; !ok {
					visited[event.GUID] = struct{}{}
					events = append(events, event)
				}
			}
		}
	}
	f, err := os.OpenFile("data/events.json", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(f).Encode(events); err != nil {
		panic(err)
	}
}

func speakers() {
	var (
		speakers []Person
		visited  = make(map[string]struct{})
	)
	for _, day := range schedule.Conference.Days {
		for _, rooms := range day.Rooms {
			for _, event := range rooms {
				for _, person := range event.Persons {
					if _, ok := visited[person.GUID]; !ok {
						visited[person.GUID] = struct{}{}
						speakers = append(speakers, person)
					}
				}
			}
		}
	}
	f, err := os.OpenFile("data/speakers.json", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(f).Encode(speakers); err != nil {
		panic(err)
	}
}

func devRooms() {
	var (
		devrooms []Track
	)
	for _, track := range schedule.Conference.Tracks {
		switch track.Name {
		case "Junior", "Main Track", "Main Track (K-building)",
			"BOF/Unconference", "/dev/random", "Workshops", "Lightning talks":
			// Would be nice if the tracks would be tagged somehow as a devroom.
		default:
			devrooms = append(devrooms, track)
		}
	}
	f, err := os.OpenFile("data/devrooms.json", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(f).Encode(devrooms); err != nil {
		panic(err)
	}
}
