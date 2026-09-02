package main

import (
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
	Building    string `json:"building"`
}

func (r *Room) UnmarshalJSON(data []byte) error {
	type Alias Room
	if err := json.Unmarshal(data, &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}); err != nil {
		return err
	}
	name := strings.ToLower(r.Name)
	switch {
	case strings.HasPrefix(name, "aw"):
		r.Building = "AW"
	case strings.HasPrefix(name, "j"):
		r.Building = "J"
	case strings.HasPrefix(name, "k"):
		r.Building = "K"
	case strings.HasPrefix(name, "h"):
		r.Building = "H"
	case strings.HasPrefix(name, "u"):
		r.Building = "U"
	}
	return nil
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
	Avatar    string `json:"avatar"`
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

const (
	scheduleFile = "schedule.json"
	scheduleURL  = "https://pretalx.fosdem.org/fosdem-2026/schedule/export/schedule_fosdem.json"
)

var schedule Schedule

func main() {
	path := scheduleFile
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	rawSchedule, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("%v\n\nDownload the Pretalx export (login required) and place it in the project root:\n\n\t%s\n", err, scheduleURL)
	}

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
	if err := writeKeynotes(); err != nil {
		return err
	}
	if err := writeRoomInfo(); err != nil {
		return err
	}
	if err := writeRoomTracks(); err != nil {
		return err
	}
	return writeDayGrid()
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
	trackRooms := make(map[string][]string)            // track name → room descriptions (ordered)
	trackSeen := make(map[string]map[string]bool)      // track name → room names seen
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

	list := TrackList{
		Other:      make([]TrackInfo, 0),
		MainTracks: make([]TrackInfo, 0),
		DevRooms:   make([]TrackInfo, 0),
	}
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

type TrackEvent struct {
	Event
	Day string `json:"day"`
}

func writeTracks() error {
	tracks := make(map[string][]TrackEvent)
	for _, day := range schedule.Conference.Days {
		date, _ := time.Parse("2006-01-02", day.Date)
		dayName := date.Weekday().String()
		for _, events := range day.Rooms {
			for _, event := range events {
				tracks[event.Track] = append(tracks[event.Track], TrackEvent{
					Event: event,
					Day:   dayName,
				})
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

func writeRoomTracks() error {
	// Track name → slug lookup.
	trackSlug := make(map[string]string)
	for _, t := range schedule.Conference.Tracks {
		trackSlug[t.Name] = string(t.Slug)
	}

	// Per day, per room, per track: compute time range.
	type timeRange struct{ min, max time.Time }
	type dayRoomTracks = map[string]map[string]*timeRange // roomName → trackName → range

	dayData := make(map[string]dayRoomTracks) // weekday → ...

	for _, day := range schedule.Conference.Days {
		date, _ := time.Parse("2006-01-02", day.Date)
		weekday := date.Weekday().String()
		if dayData[weekday] == nil {
			dayData[weekday] = make(dayRoomTracks)
		}
		for roomName, events := range day.Rooms {
			if dayData[weekday][roomName] == nil {
				dayData[weekday][roomName] = make(map[string]*timeRange)
			}
			for _, event := range events {
				start, _ := time.Parse("15:04", event.Start)
				end, _ := time.Parse("15:04", event.End)
				if tr, ok := dayData[weekday][roomName][event.Track]; !ok {
					dayData[weekday][roomName][event.Track] = &timeRange{min: start, max: end}
				} else {
					if start.Before(tr.min) {
						tr.min = start
					}
					if end.After(tr.max) {
						tr.max = end
					}
				}
			}
		}
	}

	const firstHour = 9
	const numHours = 10                      // hours 9–18
	const slotsPerHour = 12                  // 5-minute slots
	const numSlots = numHours * slotsPerHour // 120

	// Convert a time to a slot index (0-based from 09:00).
	toSlot := func(t time.Time) int {
		return (t.Hour()-firstHour)*slotsPerHour + t.Minute()/5
	}

	type RoomCell struct {
		Type      string `json:"type"`
		Span      int    `json:"span"`
		HourStart bool   `json:"hourStart,omitempty"`
		Track     string `json:"track,omitempty"`
		Slug      string `json:"slug,omitempty"`
	}

	// Split an empty range into chunks at hour boundaries.
	emptyChunks := func(from, to int) []RoomCell {
		var cells []RoomCell
		for from < to {
			nextHour := ((from / slotsPerHour) + 1) * slotsPerHour
			if nextHour > to {
				nextHour = to
			}
			cells = append(cells, RoomCell{
				Type:      "empty",
				Span:      nextHour - from,
				HourStart: from%slotsPerHour == 0,
			})
			from = nextHour
		}
		return cells
	}
	type RoomTrackRow struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Slug        string     `json:"slug"`
		Building    string     `json:"building"`
		Cells       []RoomCell `json:"cells"`
	}
	type BuildingGroup struct {
		Name  string         `json:"name"`
		Rooms []RoomTrackRow `json:"rooms"`
	}
	type DayGrid struct {
		Buildings []BuildingGroup `json:"buildings"`
	}
	type RoomTracksData struct {
		Hours    []int   `json:"hours"`
		Saturday DayGrid `json:"saturday"`
		Sunday   DayGrid `json:"sunday"`
	}

	buildDay := func(weekday string) DayGrid {
		drt := dayData[weekday]
		if drt == nil {
			return DayGrid{}
		}

		var buildings []BuildingGroup

		for _, room := range schedule.Conference.Rooms {
			trackRanges, ok := drt[room.Name]
			if !ok {
				continue
			}

			// Start a new building group if needed.
			if len(buildings) == 0 || buildings[len(buildings)-1].Name != room.Building {
				buildings = append(buildings, BuildingGroup{Name: room.Building})
			}

			// Collect and sort track blocks by start time.
			type trackEntry struct {
				name string
				tr   *timeRange
			}
			var entries []trackEntry
			for name, tr := range trackRanges {
				entries = append(entries, trackEntry{name, tr})
			}
			slices.SortFunc(entries, func(a, b trackEntry) int {
				return a.tr.min.Compare(b.tr.min)
			})

			// Convert blocks to cells with gap-filling.
			var cells []RoomCell
			slot := 0
			for _, e := range entries {
				startSlot := toSlot(e.tr.min)
				endSlot := toSlot(e.tr.max)
				if startSlot < 0 {
					startSlot = 0
				}
				if endSlot > numSlots {
					endSlot = numSlots
				}
				span := endSlot - startSlot
				if span <= 0 {
					continue
				}

				// Insert empty cells for gap before this block.
				if startSlot > slot {
					cells = append(cells, emptyChunks(slot, startSlot)...)
				}

				cells = append(cells, RoomCell{
					Type:      "track",
					Span:      span,
					HourStart: startSlot%slotsPerHour == 0,
					Track:     e.name,
					Slug:      trackSlug[e.name],
				})
				slot = startSlot + span
			}
			// Trailing empty cells.
			if slot < numSlots {
				cells = append(cells, emptyChunks(slot, numSlots)...)
			}

			buildings[len(buildings)-1].Rooms = append(
				buildings[len(buildings)-1].Rooms,
				RoomTrackRow{
					Name:        room.Name,
					Description: room.Description,
					Slug:        string(room.Slug),
					Building:    room.Building,
					Cells:       cells,
				},
			)
		}

		return DayGrid{Buildings: buildings}
	}

	hours := make([]int, numHours)
	for i := range hours {
		hours[i] = firstHour + i
	}

	data := RoomTracksData{
		Hours:    hours,
		Saturday: buildDay("Saturday"),
		Sunday:   buildDay("Sunday"),
	}

	return writeJSON("data/roomtracks.json", data)
}

// writeDayGrid produces the classic per-day timetable: rooms as columns, time
// as rows in fixed 5-minute slots. Event cells span multiple rows (rowspan) for
// their duration; cells covered by a rowspan above are precomputed as "covered"
// so the template skips them. Colors cycle 1..10 in emit order (row by row, left
// to right) so adjacent events differ, matching the original site.
func writeDayGrid() error {
	type DayCell struct {
		Type    string `json:"type"`              // "event" | "empty" | "covered"
		Rowspan int    `json:"rowspan,omitempty"` // rows this cell spans (event only)
		Title   string `json:"title,omitempty"`
		Slug    string `json:"slug,omitempty"`
		Color   int    `json:"color,omitempty"`
	}
	type DayRow struct {
		Slot   int       `json:"slot"`   // 5-min slot index from midnight
		Time   string    `json:"time"`   // "09:30"
		Anchor string    `json:"anchor"` // "0930"
		Cells  []DayCell `json:"cells"`
	}
	type DayColumn struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Slug        string `json:"slug"`
		Building    string `json:"building"`
	}
	type DayGridDay struct {
		Weekday string      `json:"weekday"`
		Columns []DayColumn `json:"columns"`
		Rows    []DayRow    `json:"rows"`
	}

	toSlot := func(t time.Time) int { return t.Hour()*12 + t.Minute()/5 }

	buildDay := func(day Day) (DayGridDay, error) {
		date, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			return DayGridDay{}, fmt.Errorf("invalid day date %q: %w", day.Date, err)
		}
		weekday := date.Weekday().String()

		// Columns: conference room order, filtered to rooms active this day.
		var columns []DayColumn
		colIndex := make(map[string]int)
		for _, r := range schedule.Conference.Rooms {
			if evs, ok := day.Rooms[r.Name]; ok && len(evs) > 0 {
				colIndex[r.Name] = len(columns)
				columns = append(columns, DayColumn{
					Name:        r.Name,
					Description: r.Description,
					Slug:        string(r.Slug),
					Building:    r.Building,
				})
			}
		}

		// Placed event: which column, start/end slot, content.
		type placed struct {
			col        int
			start, end int
			cell       DayCell
		}
		var events []placed
		minSlot, maxSlot := 1<<30, 0
		for roomName, evs := range day.Rooms {
			col, ok := colIndex[roomName]
			if !ok {
				continue
			}
			for _, e := range evs {
				start, err := time.Parse("15:04", e.Start)
				if err != nil {
					return DayGridDay{}, fmt.Errorf("event %q bad start: %w", e.Slug, err)
				}
				end, err := time.Parse("15:04", e.End)
				if err != nil {
					return DayGridDay{}, fmt.Errorf("event %q bad end: %w", e.Slug, err)
				}
				s, en := toSlot(start), toSlot(end)
				if en <= s {
					continue
				}
				events = append(events, placed{
					col:   col,
					start: s,
					end:   en,
					cell: DayCell{
						Type:  "event",
						Title: e.Title,
						Slug:  string(e.Slug),
					},
				})
				if s < minSlot {
					minSlot = s
				}
				if en > maxSlot {
					maxSlot = en
				}
			}
		}
		if len(events) == 0 {
			return DayGridDay{Weekday: weekday, Columns: columns}, nil
		}

		// Fixed 5-minute rows across the active range so row height stays
		// proportional to duration, matching the classic timetable.
		numRows := maxSlot - minSlot
		rows := make([]DayRow, numRows)
		for i := range rows {
			t := minSlot + i
			rows[i] = DayRow{
				Slot:   t,
				Time:   fmt.Sprintf("%02d:%02d", t/12, (t%12)*5),
				Anchor: fmt.Sprintf("%02d%02d", t/12, (t%12)*5),
				Cells:  make([]DayCell, len(columns)),
			}
			for c := range rows[i].Cells {
				rows[i].Cells[c] = DayCell{Type: "empty"}
			}
		}

		for _, p := range events {
			startRow := p.start - minSlot
			endRow := p.end - minSlot
			cell := p.cell
			cell.Rowspan = endRow - startRow
			rows[startRow].Cells[p.col] = cell
			for r := startRow + 1; r < endRow; r++ {
				rows[r].Cells[p.col] = DayCell{Type: "covered"}
			}
		}

		// Color events sequentially in emit order (row by row, left to right)
		// so neighbours differ, cycling 1..10.
		color := 0
		for ri := range rows {
			for ci := range rows[ri].Cells {
				if rows[ri].Cells[ci].Type == "event" {
					rows[ri].Cells[ci].Color = color%10 + 1
					color++
				}
			}
		}

		return DayGridDay{Weekday: weekday, Columns: columns, Rows: rows}, nil
	}

	days := make(map[string]DayGridDay)
	for _, day := range schedule.Conference.Days {
		dg, err := buildDay(day)
		if err != nil {
			return err
		}
		days[strings.ToLower(dg.Weekday)] = dg
	}
	return writeJSON("data/daygrid.json", days)
}
