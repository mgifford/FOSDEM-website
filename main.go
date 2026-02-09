package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Schedule struct {
	Conference Conference `json:"conference"`
}

type Conference struct {
	Title   string  `json:"title"`
	Acronym string  `json:"acronym"`
	Rooms   []Room  `json:"rooms"`
	Tracks  []Track `json:"tracks"`
	Days    []Day   `json:"days"`
}

// Slug, but without number prefix.
// I.e. `261-janson -> janson`
type Slug string

func (s *Slug) UnmarshalJSON(data []byte) error {
	var slug string
	if err := json.Unmarshal(data, &slug); err != nil {
		return err
	}
	conferencePrefix := fmt.Sprintf("%s-", schedule.Conference.Acronym)
	if after, ok := strings.CutPrefix(slug, conferencePrefix); ok {
		slug = after
	}
	if _, err := fmt.Sscanf(slug, "%d-%s", new(uint), s); err != nil {
		return err
	}
	return nil
}

type Room struct {
	Name        string `json:"name"`
	Slug        Slug   `json:"slug"`
	GUID        string `json:"guid"`
	Description string `json:"description"`
	Capacity    uint   `json:"capacity"`
}

type Event struct {
	GUID     string   `json:"guid"`
	Title    string   `json:"title"`
	Abstract string   `json:"abstract"`
	Track    string   `json:"track"`
	Start    string   `json:"start"`
	Slug     Slug     `json:"slug"`
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
	if _, err := fmt.Sscanf(e.Duration, "%02d:%02d", &hours, &minutes); err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
	endTime := startTime.Add(duration)

	e.End = endTime.Format("15:04")

	return nil
}

type Person struct {
	GUID      string `json:"guid"`
	Name      string `json:"name"`
	Biography string `json:"biography"`
}

type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Track struct {
	Name  string `json:"name"`
	Slug  Slug   `json:"slug"`
	Color string `json:"color"`
}

type Day struct {
	Rooms map[string][]Event `json:"rooms"`
}

var (
	//go:embed schedule.json
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

	f, err := os.OpenFile("data/schedule.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(f).Encode(schedule); err != nil {
		panic(err)
	}
}

func events() {
	var (
		events = make(map[string][]Event)
	)
	for _, day := range schedule.Conference.Days {
		for _, rooms := range day.Rooms {
			for _, event := range rooms {
				id := string(event.Slug)
				events[id] = append(events[id], event)
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
	type Speaker struct {
		Person
		Events []Slug `json:"events"`
	}

	var speakers = make(map[string]Speaker)
	for _, day := range schedule.Conference.Days {
		for _, rooms := range day.Rooms {
			for _, event := range rooms {
				for _, person := range event.Persons {
					id := person.GUID
					if speaker, ok := speakers[id]; !ok {
						speakers[id] = Speaker{
							Person: person,
							Events: []Slug{event.Slug},
						}
					} else {
						speaker.Events = append(speaker.Events, event.Slug)
						speakers[id] = speaker
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
