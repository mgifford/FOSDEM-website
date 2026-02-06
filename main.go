package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"time"
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

type Room struct{}

type Event struct {
	GUID     string   `json:"guid"`
	Title    string   `json:"title"`
	Abstract string   `json:"abstract"`
	Track    string   `json:"track"`
	Start    string   `json:"start"`
	Slug     string   `json:"slug"`
	Duration string   `json:"duration"`
	Room     string   `json:"room"`
	Persons  []Person `json:"persons"`
	Links    []Link   `json:"links"`

	// NOTE: all fields below get calculated upon unmarshaling.
	End string `json:"end"`
}

func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	if err := json.Unmarshal(data, &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}); err != nil {
		return err
	}

	startTime, err := time.Parse("15:04", e.Start)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}

	var hours, minutes int
	_, err = fmt.Sscanf(e.Duration, "%02d:%02d", &hours, &minutes)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
	endTime := startTime.Add(duration)

	e.End = endTime.Format("15:04")

	return nil
}

type Person struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
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
	tracks()
}

func events() {
	var (
		events = make(map[string]Event)
	)
	for _, day := range schedule.Conference.Days {
		for _, rooms := range day.Rooms {
			for _, event := range rooms {
				if _, ok := events[event.Slug]; !ok {
					events[event.Slug] = event
				}
			}
		}
	}
	f, err := os.OpenFile("data/events.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
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
	f, err := os.OpenFile("data/speakers.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
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
	f, err := os.OpenFile("data/devrooms.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(f).Encode(devrooms); err != nil {
		panic(err)
	}
}

func tracks() {
	var (
		tracks = make(map[string][]Event)
	)
	for _, day := range schedule.Conference.Days {
		for _, events := range day.Rooms {
			for _, event := range events {
				track := event.Track
				tracks[track] = append(tracks[track], event)
			}
		}
	}
	f, err := os.OpenFile("data/tracks.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(f).Encode(tracks); err != nil {
		panic(err)
	}
}
