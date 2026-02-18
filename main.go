package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
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

// Slug strips the numeric prefix from Pretalx slugs.
// e.g. "261-janson" → "janson"
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

	// Calculated from Start + Duration during unmarshaling.
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
	endTime := startTime.Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute)

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
	Type  string `json:"type"`
}

type Day struct {
	Date  string             `json:"date"`
	Rooms map[string][]Event `json:"rooms"`
}

var keynoteSlugs = map[string]bool{
	"welcome-to-fosdem-2026":                                true,
	"foss-in-times-of-war-scarcity-and-adversarial-ai":      true,
	"free-as-in-burned-out-who-really-pays-for-open-source": true,
	"open-source-security-in-spite-of-ai":                   true,
	"closing-fosdem-2026":                                   true,
}

var (
	//go:embed schedule.json
	rawSchedule []byte
	schedule    Schedule
)

func main() {
	if err := json.Unmarshal(rawSchedule, &struct {
		*Schedule `json:"schedule"`
	}{
		Schedule: &schedule,
	}); err != nil {
		log.Fatal(err)
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := writeJSON("data/schedule.json", schedule); err != nil {
		return err
	}
	if err := writeEvents(); err != nil {
		return err
	}
	if err := writeSpeakers(); err != nil {
		return err
	}
	if err := writeTracks(); err != nil {
		return err
	}
	if err := writeTracksByType("data/devrooms.json", "devroom"); err != nil {
		return err
	}
	if err := writeTracksByType("data/maintracks.json", "maintrack"); err != nil {
		return err
	}
	return writeKeynotes()
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(v)
}

func writeEvents() error {
	events := make(map[string][]Event)
	for _, day := range schedule.Conference.Days {
		for _, rooms := range day.Rooms {
			for _, event := range rooms {
				id := string(event.Slug)
				events[id] = append(events[id], event)
			}
		}
	}
	return writeJSON("data/events.json", events)
}

func writeSpeakers() error {
	type Speaker struct {
		Person
		Events []Slug `json:"events"`
	}

	speakers := make(map[string]Speaker)
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
	return writeJSON("data/speakers.json", speakers)
}

func writeTracksByType(path, trackType string) error {
	var tracks []Track
	for _, track := range schedule.Conference.Tracks {
		if track.Type == trackType {
			tracks = append(tracks, track)
		}
	}
	return writeJSON(path, tracks)
}

func writeKeynotes() error {
	type KeynoteEvent struct {
		Event
		Day string `json:"day"`
	}

	var keynotes []KeynoteEvent
	for _, day := range schedule.Conference.Days {
		date, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			return fmt.Errorf("invalid day date %q: %w", day.Date, err)
		}
		dayName := date.Weekday().String()
		for _, events := range day.Rooms {
			for _, event := range events {
				if keynoteSlugs[string(event.Slug)] {
					keynotes = append(keynotes, KeynoteEvent{
						Event: event,
						Day:   dayName,
					})
				}
			}
		}
	}
	return writeJSON("data/keynotes.json", keynotes)
}

func writeTracks() error {
	tracks := make(map[string][]Event)
	for _, day := range schedule.Conference.Days {
		for _, events := range day.Rooms {
			for _, event := range events {
				tracks[event.Track] = append(tracks[event.Track], event)
			}
		}
	}
	return writeJSON("data/tracks.json", tracks)
}
