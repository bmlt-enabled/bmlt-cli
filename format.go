package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Meeting captures the subset of fields we render. JSON tags match the
// Semantic API. All values arrive as strings (BMLT's JSON quirk).
type Meeting struct {
	ID                 string `json:"id_bigint"`
	Name               string `json:"meeting_name"`
	Weekday            string `json:"weekday_tinyint"`
	StartTime          string `json:"start_time"`
	Duration           string `json:"duration_time"`
	LocationText       string `json:"location_text"`
	LocationStreet     string `json:"location_street"`
	LocationMunicipal  string `json:"location_municipality"`
	LocationProvince   string `json:"location_province"`
	LocationPostal     string `json:"location_postal_code_1"`
	LocationInfo       string `json:"location_info"`
	VirtualLink        string `json:"virtual_meeting_link"`
	VirtualPhone       string `json:"phone_meeting_number"`
	VenueType          string `json:"venue_type"`
	Formats            string `json:"formats"`
	FormatSharedIDList string `json:"format_shared_id_list"`
	ServiceBodyName    string `json:"service_body_bigint"`
	Latitude           string `json:"latitude"`
	Longitude          string `json:"longitude"`
}

var weekdayLabels = []string{"", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func weekdayLabel(s string) string {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 7 {
		return "Unknown"
	}
	return weekdayLabels[n]
}

func venueLabel(s string) string {
	switch s {
	case "1":
		return "in-person"
	case "2":
		return "virtual"
	case "3":
		return "hybrid"
	default:
		return ""
	}
}

// trimTime cuts "19:30:00" → "19:30". Returns input unchanged if not HH:MM:SS.
func trimTime(s string) string {
	if len(s) >= 5 && s[2] == ':' {
		return s[:5]
	}
	return s
}

func printMeetings(w io.Writer, body []byte) error {
	var meetings []Meeting
	if err := json.Unmarshal(body, &meetings); err != nil {
		return err
	}
	if len(meetings) == 0 {
		fmt.Fprintln(w, "No meetings.")
		return nil
	}

	// Group by weekday (1..7).
	byDay := map[int][]Meeting{}
	dayKeys := []int{}
	for _, m := range meetings {
		n, _ := strconv.Atoi(m.Weekday)
		if _, ok := byDay[n]; !ok {
			dayKeys = append(dayKeys, n)
		}
		byDay[n] = append(byDay[n], m)
	}
	sort.Ints(dayKeys)

	for i, day := range dayKeys {
		if i > 0 {
			fmt.Fprintln(w)
		}
		label := "Unknown day"
		if day >= 1 && day <= 7 {
			label = weekdayLabels[day]
		}
		fmt.Fprintf(w, "%s (%d)\n", label, len(byDay[day]))
		// Sort within day by time.
		sort.SliceStable(byDay[day], func(a, b int) bool {
			return byDay[day][a].StartTime < byDay[day][b].StartTime
		})
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, m := range byDay[day] {
			loc := compactLocation(m)
			venue := venueLabel(m.VenueType)
			if venue != "" {
				venue = " [" + venue + "]"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s%s\n",
				trimTime(m.StartTime), strings.TrimSpace(m.Name), loc, venue)
		}
		tw.Flush()
	}
	fmt.Fprintf(w, "\n%d meetings\n", len(meetings))
	return nil
}

func compactLocation(m Meeting) string {
	parts := []string{}
	if s := strings.TrimSpace(m.LocationText); s != "" {
		parts = append(parts, s)
	}
	cityState := strings.TrimSpace(m.LocationMunicipal)
	if s := strings.TrimSpace(m.LocationProvince); s != "" {
		if cityState != "" {
			cityState += ", " + s
		} else {
			cityState = s
		}
	}
	if cityState != "" {
		parts = append(parts, cityState)
	}
	if len(parts) == 0 {
		if s := strings.TrimSpace(m.VirtualLink); s != "" {
			return s
		}
	}
	return strings.Join(parts, " — ")
}

// --- Service bodies ---

type ServiceBody struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	ParentID    string `json:"parent_id"`
	URL         string `json:"url"`
	Helpline    string `json:"helpline"`
	WorldID     string `json:"world_id"`
}

func printServiceBodies(w io.Writer, body []byte) error {
	var bodies []ServiceBody
	if err := json.Unmarshal(body, &bodies); err != nil {
		return err
	}
	if len(bodies) == 0 {
		fmt.Fprintln(w, "No service bodies.")
		return nil
	}

	// Build child index for tree rendering.
	children := map[string][]ServiceBody{}
	known := map[string]bool{}
	for _, b := range bodies {
		known[b.ID] = true
	}
	roots := []ServiceBody{}
	for _, b := range bodies {
		if b.ParentID == "" || b.ParentID == "0" || !known[b.ParentID] {
			roots = append(roots, b)
		} else {
			children[b.ParentID] = append(children[b.ParentID], b)
		}
	}

	sortBodies := func(s []ServiceBody) {
		sort.SliceStable(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	}
	sortBodies(roots)
	for k := range children {
		sortBodies(children[k])
	}

	var walk func(b ServiceBody, depth int)
	walk = func(b ServiceBody, depth int) {
		indent := strings.Repeat("  ", depth)
		typ := b.Type
		if typ == "" {
			typ = "?"
		}
		fmt.Fprintf(w, "%s[%s] %s  (id=%s)\n", indent, typ, b.Name, b.ID)
		for _, c := range children[b.ID] {
			walk(c, depth+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
	fmt.Fprintf(w, "\n%d service bodies\n", len(bodies))
	return nil
}

// --- Formats ---

type Format struct {
	ID          string `json:"id"`
	KeyString   string `json:"key_string"`
	Name        string `json:"name_string"`
	Description string `json:"description_string"`
	Language    string `json:"lang"`
	WorldID     string `json:"world_id"`
}

func printFormats(w io.Writer, body []byte) error {
	var formats []Format
	if err := json.Unmarshal(body, &formats); err != nil {
		return err
	}
	if len(formats) == 0 {
		fmt.Fprintln(w, "No formats.")
		return nil
	}
	sort.SliceStable(formats, func(i, j int) bool { return formats[i].KeyString < formats[j].KeyString })
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tKEY\tNAME\tDESCRIPTION")
	for _, f := range formats {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", f.ID, f.KeyString, f.Name, truncate(f.Description, 60))
	}
	tw.Flush()
	fmt.Fprintf(w, "\n%d formats\n", len(formats))
	return nil
}

// --- Field keys / values ---

type FieldKey struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func printFieldKeys(w io.Writer, body []byte) error {
	var keys []FieldKey
	if err := json.Unmarshal(body, &keys); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, k := range keys {
		fmt.Fprintf(tw, "%s\t%s\n", k.Key, k.Description)
	}
	tw.Flush()
	return nil
}

func printFieldValues(w io.Writer, body []byte) error {
	// Response is [{<field>: <value>, "ids": "1,2,3"}, ...]
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for _, r := range rows {
		var val, ids string
		for k, v := range r {
			s := fmt.Sprint(v)
			if k == "ids" {
				ids = s
			} else {
				val = s
			}
		}
		count := 0
		if ids != "" {
			count = strings.Count(ids, ",") + 1
		}
		fmt.Fprintf(w, "%s\t(%d)\n", val, count)
	}
	return nil
}

// --- Generic JSON pretty-print ---

func printPrettyJSON(w io.Writer, body []byte) error {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		// not JSON — passthrough
		_, err := w.Write(body)
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
