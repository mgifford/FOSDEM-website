package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
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
	if err := writeTrackList(); err != nil {
		return err
	}
	if err := writeTracksByType("data/devrooms.json", "devroom"); err != nil {
		return err
	}
	if err := writeTracksByType("data/maintracks.json", "maintrack"); err != nil {
		return err
	}
	if err := writeKeynotes(); err != nil {
		return err
	}
	return writeRoomInfo()
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

type TrackInfo struct {
	Track
	Rooms    string `json:"rooms"`
	Saturday string `json:"saturday"`
	Sunday   string `json:"sunday"`
}

func computeTrackInfo() []TrackInfo {
	// Room name → description lookup.
	roomDesc := make(map[string]string)
	for _, r := range schedule.Conference.Rooms {
		roomDesc[r.Name] = r.Description
	}

	type dayRange struct {
		min, max time.Time
	}

	// Per-track: collect unique rooms and time ranges per weekday.
	trackRooms := make(map[string][]string)    // track name → room descriptions (ordered)
	trackSeen := make(map[string]map[string]bool) // track name → room names seen
	trackDays := make(map[string]map[string]*dayRange) // track name → weekday → range

	for _, day := range schedule.Conference.Days {
		date, _ := time.Parse("2006-01-02", day.Date)
		weekday := date.Weekday().String()
		for _, events := range day.Rooms {
			for _, event := range events {
				name := event.Track

				// Collect unique rooms in order.
				if trackSeen[name] == nil {
					trackSeen[name] = make(map[string]bool)
				}
				if !trackSeen[name][event.Room] {
					trackSeen[name][event.Room] = true
					if desc, ok := roomDesc[event.Room]; ok {
						trackRooms[name] = append(trackRooms[name], desc)
					}
				}

				// Track time ranges per day.
				start, _ := time.Parse("15:04", event.Start)
				end, _ := time.Parse("15:04", event.End)
				if trackDays[name] == nil {
					trackDays[name] = make(map[string]*dayRange)
				}
				if dr, ok := trackDays[name][weekday]; !ok {
					trackDays[name][weekday] = &dayRange{min: start, max: end}
				} else {
					if start.Before(dr.min) {
						dr.min = start
					}
					if end.After(dr.max) {
						dr.max = end
					}
				}
			}
		}
	}

	formatRange := func(dr *dayRange) string {
		if dr == nil {
			return ""
		}
		return dr.min.Format("15:04") + " - " + dr.max.Format("15:04")
	}

	var infos []TrackInfo
	for _, track := range schedule.Conference.Tracks {
		if len(trackRooms[track.Name]) == 0 {
			continue
		}
		infos = append(infos, TrackInfo{
			Track:    track,
			Rooms:    strings.Join(trackRooms[track.Name], ", "),
			Saturday: formatRange(trackDays[track.Name]["Saturday"]),
			Sunday:   formatRange(trackDays[track.Name]["Sunday"]),
		})
	}
	return infos
}

func writeTrackList() error {
	infos := computeTrackInfo()

	type TrackList struct {
		Other      []TrackInfo `json:"other"`
		MainTracks []TrackInfo `json:"maintracks"`
		DevRooms   []TrackInfo `json:"devrooms"`
	}

	var list TrackList
	for _, info := range infos {
		switch info.Type {
		case "maintrack":
			list.MainTracks = append(list.MainTracks, info)
		case "devroom":
			list.DevRooms = append(list.DevRooms, info)
		default:
			list.Other = append(list.Other, info)
		}
	}
	slices.SortFunc(list.Other, func(a, b TrackInfo) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(list.MainTracks, func(a, b TrackInfo) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(list.DevRooms, func(a, b TrackInfo) int { return strings.Compare(a.Name, b.Name) })
	return writeJSON("data/tracklist.json", list)
}

func writeTracksByType(path, trackType string) error {
	infos := computeTrackInfo()
	var filtered []TrackInfo
	for _, info := range infos {
		if info.Type == trackType {
			filtered = append(filtered, info)
		}
	}
	slices.SortFunc(filtered, func(a, b TrackInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	return writeJSON(path, filtered)
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

func writeRoomInfo() error {
	// Track name → slug lookup.
	trackSlug := make(map[string]string)
	for _, t := range schedule.Conference.Tracks {
		trackSlug[t.Name] = string(t.Slug)
	}

	type dayRange struct{ min, max time.Time }
	type dayInfo struct {
		rng    dayRange
		tracks []string // ordered unique track names
		seen   map[string]bool
	}
	// room name → weekday → info
	roomDays := make(map[string]map[string]*dayInfo)

	for _, day := range schedule.Conference.Days {
		date, _ := time.Parse("2006-01-02", day.Date)
		weekday := date.Weekday().String()
		for roomName, events := range day.Rooms {
			for _, event := range events {
				if roomDays[roomName] == nil {
					roomDays[roomName] = make(map[string]*dayInfo)
				}
				start, _ := time.Parse("15:04", event.Start)
				end, _ := time.Parse("15:04", event.End)
				di, ok := roomDays[roomName][weekday]
				if !ok {
					di = &dayInfo{
						rng:  dayRange{min: start, max: end},
						seen: make(map[string]bool),
					}
					roomDays[roomName][weekday] = di
				} else {
					if start.Before(di.rng.min) {
						di.rng.min = start
					}
					if end.After(di.rng.max) {
						di.rng.max = end
					}
				}
				if !di.seen[event.Track] {
					di.seen[event.Track] = true
					di.tracks = append(di.tracks, event.Track)
				}
			}
		}
	}

	type RoomTrack struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	type RoomDayInfo struct {
		Time   string      `json:"time"`
		Tracks []RoomTrack `json:"tracks"`
	}
	type RoomInfo struct {
		Saturday *RoomDayInfo `json:"saturday,omitempty"`
		Sunday   *RoomDayInfo `json:"sunday,omitempty"`
	}

	formatDayInfo := func(di *dayInfo) *RoomDayInfo {
		if di == nil {
			return nil
		}
		rdi := &RoomDayInfo{
			Time: di.rng.min.Format("15:04") + "-" + di.rng.max.Format("15:04"),
		}
		for _, name := range di.tracks {
			rdi.Tracks = append(rdi.Tracks, RoomTrack{
				Name: name,
				Slug: trackSlug[name],
			})
		}
		return rdi
	}

	info := make(map[string]RoomInfo)
	for name, days := range roomDays {
		info[name] = RoomInfo{
			Saturday: formatDayInfo(days["Saturday"]),
			Sunday:   formatDayInfo(days["Sunday"]),
		}
	}
	return writeJSON("data/roominfo.json", info)
}
