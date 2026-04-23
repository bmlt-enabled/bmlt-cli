package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// commonOpts holds flags shared across all server-bound subcommands.
type commonOpts struct {
	server  string
	json    bool
	tsml    bool
	urlOnly bool
	timeout int
}

func attachCommon(fs *flag.FlagSet, o *commonOpts) {
	fs.StringVar(&o.server, "s", "", "BMLT root server URL or fuzzy name (env: BMLT_SERVER)")
	fs.StringVar(&o.server, "server", "", "BMLT root server URL or fuzzy name (env: BMLT_SERVER)")
	fs.BoolVar(&o.json, "json", false, "Output raw JSON")
	fs.BoolVar(&o.tsml, "tsml", false, "Output TSML JSON (search only)")
	fs.BoolVar(&o.urlOnly, "url", false, "Print the request URL and exit")
	fs.IntVar(&o.timeout, "timeout", 30, "HTTP timeout in seconds")
}

// resolveClient turns the server flag (URL or fuzzy name) into a Client.
func resolveClient(o *commonOpts) (*Client, error) {
	server := o.server
	if server == "" {
		server = os.Getenv("BMLT_SERVER")
	}
	if server == "" {
		return nil, fmt.Errorf("no server specified — pass -s <url|name> or set BMLT_SERVER")
	}
	if !strings.Contains(server, "://") {
		entry, err := lookupServer(server)
		if err != nil {
			return nil, err
		}
		server = entry.URL
		fmt.Fprintf(os.Stderr, "→ %s (%s)\n", entry.Name, server)
	}
	c := NewClient(server, time.Duration(o.timeout)*time.Second)
	switch {
	case o.tsml:
		c.Format = "tsml"
	case o.json:
		c.Format = "json"
	}
	return c, nil
}

// parseList splits a comma-separated CLI value and trims whitespace.
func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

var weekdayNames = map[string]int{
	"sun": 1, "sunday": 1,
	"mon": 2, "monday": 2,
	"tue": 3, "tues": 3, "tuesday": 3,
	"wed": 4, "weds": 4, "wednesday": 4,
	"thu": 5, "thur": 5, "thurs": 5, "thursday": 5,
	"fri": 6, "friday": 6,
	"sat": 7, "saturday": 7,
}

// parseWeekdays accepts "mon,wed,fri" or "2,4,6" (or mixed, with negatives).
func parseWeekdays(s string) ([]string, error) {
	parts := parseList(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		neg := strings.HasPrefix(p, "-")
		bare := strings.TrimPrefix(p, "-")
		if n, err := strconv.Atoi(bare); err == nil {
			if n < 1 || n > 7 {
				return nil, fmt.Errorf("weekday %d out of range (1-7)", n)
			}
			if neg {
				n = -n
			}
			out = append(out, strconv.Itoa(n))
			continue
		}
		n, ok := weekdayNames[strings.ToLower(bare)]
		if !ok {
			return nil, fmt.Errorf("unknown weekday %q", p)
		}
		if neg {
			n = -n
		}
		out = append(out, strconv.Itoa(n))
	}
	return out, nil
}

var venueNames = map[string]int{
	"in-person": 1, "inperson": 1, "in_person": 1, "physical": 1,
	"virtual": 2, "online": 2, "remote": 2,
	"hybrid": 3,
}

func parseVenueTypes(s string) ([]string, error) {
	parts := parseList(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		neg := strings.HasPrefix(p, "-")
		bare := strings.TrimPrefix(p, "-")
		if n, err := strconv.Atoi(bare); err == nil {
			if n < 1 || n > 3 {
				return nil, fmt.Errorf("venue type %d out of range (1-3)", n)
			}
			if neg {
				n = -n
			}
			out = append(out, strconv.Itoa(n))
			continue
		}
		n, ok := venueNames[strings.ToLower(bare)]
		if !ok {
			return nil, fmt.Errorf("unknown venue type %q (try in-person, virtual, hybrid)", p)
		}
		if neg {
			n = -n
		}
		out = append(out, strconv.Itoa(n))
	}
	return out, nil
}

// parseHHMM splits "18:30" into (18, 30).
func parseHHMM(s string) (int, int, error) {
	if s == "" {
		return 0, 0, nil
	}
	parts := strings.SplitN(s, ":", 2)
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("invalid hour in %q", s)
	}
	m := 0
	if len(parts) == 2 {
		m, err = strconv.Atoi(parts[1])
		if err != nil || m < 0 || m > 59 {
			return 0, 0, fmt.Errorf("invalid minute in %q", s)
		}
	}
	return h, m, nil
}

// parseLatLng splits "lat,lng" into two floats (returned as strings to preserve precision).
func parseLatLng(s string) (string, string, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected lat,lng — got %q", s)
	}
	lat := strings.TrimSpace(parts[0])
	lng := strings.TrimSpace(parts[1])
	if _, err := strconv.ParseFloat(lat, 64); err != nil {
		return "", "", fmt.Errorf("invalid latitude %q", lat)
	}
	if _, err := strconv.ParseFloat(lng, 64); err != nil {
		return "", "", fmt.Errorf("invalid longitude %q", lng)
	}
	return lat, lng, nil
}

// addCSV adds a comma-list parameter if non-empty.
func addCSV(p url.Values, key string, vals []string) {
	if len(vals) > 0 {
		p.Set(key, strings.Join(vals, ","))
	}
}

// addStr adds a string param if non-empty.
func addStr(p url.Values, key, val string) {
	if val != "" {
		p.Set(key, val)
	}
}

// addInt adds an int param if non-zero.
func addInt(p url.Values, key string, val int) {
	if val != 0 {
		p.Set(key, strconv.Itoa(val))
	}
}
