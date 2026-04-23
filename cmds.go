package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// --- servers ---

func runServers(args []string) {
	fs := flag.NewFlagSet("servers", flag.ExitOnError)
	refresh := fs.Bool("refresh", false, "Bypass cache and re-fetch the canonical list")
	asJSON := fs.Bool("json", false, "Output raw JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: bmlt servers [query] [--refresh] [--json]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	var servers []ServerEntry
	var err error
	if *refresh {
		servers, err = fetchServers()
	} else {
		servers, err = loadServers()
	}
	if err != nil {
		die(err)
	}

	query := strings.Join(fs.Args(), " ")
	servers = filterServers(servers, query)
	sort.SliceStable(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })

	if *asJSON {
		_ = printPrettyJSON(os.Stdout, mustMarshal(servers))
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, s := range servers {
		fmt.Fprintf(tw, "%s\t%s\n", s.Name, s.URL)
	}
	tw.Flush()
	if len(servers) == 0 {
		fmt.Fprintln(os.Stderr, "(no matches)")
	} else {
		fmt.Fprintf(os.Stderr, "\n%d servers\n", len(servers))
	}
}

// --- info ---

func runInfo(args []string) {
	co := &commonOpts{}
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	attachCommon(fs, co)
	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetServerInfo", nil))
		return
	}
	body, _, err := c.Fetch("GetServerInfo", nil)
	if err != nil {
		die(err)
	}
	if co.json {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	// Friendly summary.
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	r := rows[0]
	keys := []string{"version", "versionInt", "langs", "nativeLang", "centerLongitude", "centerLatitude", "centerZoom", "defaultDuration", "regionBias", "charSet", "distanceUnits", "available_keys", "google_api_key_missing", "aggregator_mode_enabled"}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, k := range keys {
		if v, ok := r[k]; ok {
			fmt.Fprintf(tw, "%s\t%v\n", k, v)
		}
	}
	tw.Flush()
}

// --- coverage ---

func runCoverage(args []string) {
	co := &commonOpts{}
	fs := flag.NewFlagSet("coverage", flag.ExitOnError)
	attachCommon(fs, co)
	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetCoverageArea", nil))
		return
	}
	body, _, err := c.Fetch("GetCoverageArea", nil)
	if err != nil {
		die(err)
	}
	_ = printPrettyJSON(os.Stdout, body)
}

// --- keys ---

func runFieldKeys(args []string) {
	co := &commonOpts{}
	fs := flag.NewFlagSet("keys", flag.ExitOnError)
	attachCommon(fs, co)
	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetFieldKeys", nil))
		return
	}
	body, _, err := c.Fetch("GetFieldKeys", nil)
	if err != nil {
		die(err)
	}
	if co.json {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	if err := printFieldKeys(os.Stdout, body); err != nil {
		die(err)
	}
}

// --- values ---

func runFieldValues(args []string) {
	co := &commonOpts{}
	var key, formats string
	var allFormats bool
	fs := flag.NewFlagSet("values", flag.ExitOnError)
	attachCommon(fs, co)
	fs.StringVar(&key, "k", "", "Meeting field key (required) — e.g. location_municipality")
	fs.StringVar(&key, "key", "", "Meeting field key (required)")
	fs.StringVar(&formats, "formats", "", "Limit to meetings with these format IDs")
	fs.BoolVar(&allFormats, "all-formats", false, "Combine with --formats: include all formats too")
	_ = fs.Parse(args)
	if key == "" {
		die(fmt.Errorf("values: --key is required (try `bmlt keys`)"))
	}
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	p := url.Values{}
	p.Set("meeting_key", key)
	addCSV(p, "specific_formats", parseList(formats))
	if allFormats {
		p.Set("all_formats", "1")
	}
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetFieldValues", p))
		return
	}
	body, _, err := c.Fetch("GetFieldValues", p)
	if err != nil {
		die(err)
	}
	if co.json {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	if err := printFieldValues(os.Stdout, body); err != nil {
		die(err)
	}
}

// --- formats ---

func runFormats(args []string) {
	co := &commonOpts{}
	var lang, formatIDs, keys string
	var showAll bool
	fs := flag.NewFlagSet("formats", flag.ExitOnError)
	attachCommon(fs, co)
	fs.StringVar(&lang, "lang", "", "Language code (en, de, fr, es, it, pt, pl, sv, dk, fa)")
	fs.StringVar(&formatIDs, "ids", "", "Comma list of format IDs (negative excludes)")
	fs.StringVar(&keys, "key-strings", "", "Filter by format key string (e.g. O,C,BEG)")
	fs.BoolVar(&showAll, "all", false, "Include unused formats")
	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	p := url.Values{}
	addStr(p, "lang_enum", lang)
	addCSV(p, "format_ids", parseList(formatIDs))
	addCSV(p, "key_strings", parseList(keys))
	if showAll {
		p.Set("show_all", "1")
	}
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetFormats", p))
		return
	}
	body, _, err := c.Fetch("GetFormats", p)
	if err != nil {
		die(err)
	}
	if co.json {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	if err := printFormats(os.Stdout, body); err != nil {
		die(err)
	}
}

// --- bodies ---

func runBodies(args []string) {
	co := &commonOpts{}
	var services string
	var recursive, parents bool
	fs := flag.NewFlagSet("bodies", flag.ExitOnError)
	attachCommon(fs, co)
	fs.StringVar(&services, "ids", "", "Filter to specific service body IDs (negative excludes)")
	fs.BoolVar(&recursive, "recursive", false, "Include children of selected bodies")
	fs.BoolVar(&parents, "parents", false, "Include ancestors of selected bodies")
	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	p := url.Values{}
	addCSV(p, "services", parseList(services))
	if recursive {
		p.Set("recursive", "1")
	}
	if parents {
		p.Set("parents", "1")
	}
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetServiceBodies", p))
		return
	}
	body, _, err := c.Fetch("GetServiceBodies", p)
	if err != nil {
		die(err)
	}
	if co.json {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	if err := printServiceBodies(os.Stdout, body); err != nil {
		die(err)
	}
}

// --- changes ---

func runChanges(args []string) {
	co := &commonOpts{}
	var start, end string
	var meetingID, sbID int
	fs := flag.NewFlagSet("changes", flag.ExitOnError)
	attachCommon(fs, co)
	fs.StringVar(&start, "from", "", "Start date YYYY-MM-DD")
	fs.StringVar(&end, "to", "", "End date YYYY-MM-DD")
	fs.IntVar(&meetingID, "meeting", 0, "Single meeting ID")
	fs.IntVar(&sbID, "service-body", 0, "Limit to a service body ID")
	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	p := url.Values{}
	addStr(p, "start_date", start)
	addStr(p, "end_date", end)
	addInt(p, "meeting_id", meetingID)
	addInt(p, "service_body_id", sbID)
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetChanges", p))
		return
	}
	body, _, err := c.Fetch("GetChanges", p)
	if err != nil {
		die(err)
	}
	_ = printPrettyJSON(os.Stdout, body)
}

// --- naws ---

func runNAWS(args []string) {
	co := &commonOpts{}
	var sbID int
	fs := flag.NewFlagSet("naws", flag.ExitOnError)
	attachCommon(fs, co)
	fs.IntVar(&sbID, "b", 0, "Service body ID (required)")
	fs.IntVar(&sbID, "service-body", 0, "Service body ID (required)")
	_ = fs.Parse(args)
	if sbID == 0 {
		die(fmt.Errorf("naws: --service-body (-b) is required"))
	}
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}
	c.Format = "csv"
	p := url.Values{}
	p.Set("sb_id", strconv.Itoa(sbID))
	if co.urlOnly {
		fmt.Println(c.BuildURL("GetNAWSDump", p))
		return
	}
	body, _, err := c.Fetch("GetNAWSDump", p)
	if err != nil {
		die(err)
	}
	os.Stdout.Write(body)
}

// --- search ---

func runSearch(args []string) {
	co := &commonOpts{}
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	attachCommon(fs, co)

	var (
		weekdays, venueTypes, formats, services, formatsOp string
		recursive                                          bool
		searchString                                       string
		near                                               string
		radius, radiusKM                                   string
		startsAfter, startsBefore, endsBefore              string
		minDur, maxDur                                     string
		fields, sortKeys, sortKey                          string
		pageSize, pageNum                                  int
		meetingIDs                                         string
		published                                          string
		lang                                               string
		rootIDs                                            string
		key, keyValue                                      string
	)

	fs.StringVar(&weekdays, "weekdays", "", "Days: mon,wed,fri  or  2,4,6 (1=Sun, neg=exclude)")
	fs.StringVar(&venueTypes, "venue-types", "", "in-person, virtual, hybrid (or 1,2,3)")
	fs.StringVar(&formats, "formats", "", "Format IDs (negative excludes)")
	fs.StringVar(&formatsOp, "formats-op", "", "AND (default) or OR")
	fs.StringVar(&services, "services", "", "Service body IDs (negative excludes)")
	fs.BoolVar(&recursive, "recursive", false, "Include children when filtering by service body")
	fs.StringVar(&searchString, "q", "", "Full-text search across name/location/notes")
	fs.StringVar(&near, "near", "", "Geographic center: lat,lng")
	fs.StringVar(&radius, "radius", "", "Radius in miles (negative N = nearest N meetings)")
	fs.StringVar(&radiusKM, "radius-km", "", "Radius in km (negative N = nearest N meetings)")
	fs.StringVar(&startsAfter, "starts-after", "", "Earliest start time HH:MM")
	fs.StringVar(&startsBefore, "starts-before", "", "Latest start time HH:MM")
	fs.StringVar(&endsBefore, "ends-before", "", "Ends before HH:MM")
	fs.StringVar(&minDur, "min-duration", "", "Minimum duration HH:MM")
	fs.StringVar(&maxDur, "max-duration", "", "Maximum duration HH:MM")
	fs.StringVar(&fields, "fields", "", "Restrict response to these field keys (comma list)")
	fs.StringVar(&sortKeys, "sort", "", "Multi-key sort (e.g. weekday_tinyint,start_time)")
	fs.StringVar(&sortKey, "sort-by", "", "Predefined sort: weekday | time | town | state | weekday_state")
	fs.IntVar(&pageSize, "page-size", 0, "Results per page")
	fs.IntVar(&pageNum, "page", 0, "Page number (1-indexed)")
	fs.StringVar(&meetingIDs, "ids", "", "Specific meeting IDs (negative excludes)")
	fs.StringVar(&published, "published", "", "all | unpublished (default: published only)")
	fs.StringVar(&lang, "lang", "", "Language for format names")
	fs.StringVar(&rootIDs, "root-ids", "", "[aggregator only] root server IDs to include/exclude")
	fs.StringVar(&key, "field-key", "", "Search on an arbitrary field key (with --field-value)")
	fs.StringVar(&keyValue, "field-value", "", "Value for --field-key")

	_ = fs.Parse(args)
	c, err := resolveClient(co)
	if err != nil {
		die(err)
	}

	p := url.Values{}

	wd, err := parseWeekdays(weekdays)
	if err != nil {
		die(err)
	}
	addCSV(p, "weekdays", wd)

	vt, err := parseVenueTypes(venueTypes)
	if err != nil {
		die(err)
	}
	addCSV(p, "venue_types", vt)

	addCSV(p, "formats", parseList(formats))
	if formatsOp != "" {
		p.Set("formats_comparison_operator", strings.ToUpper(formatsOp))
	}
	addCSV(p, "services", parseList(services))
	if recursive {
		p.Set("recursive", "1")
	}
	addCSV(p, "meeting_ids", parseList(meetingIDs))
	addStr(p, "SearchString", searchString)

	if near != "" {
		lat, lng, err := parseLatLng(near)
		if err != nil {
			die(err)
		}
		p.Set("lat_val", lat)
		p.Set("long_val", lng)
		p.Set("sort_results_by_distance", "1")
		switch {
		case radiusKM != "":
			p.Set("geo_width_km", radiusKM)
		case radius != "":
			p.Set("geo_width", radius)
		default:
			p.Set("geo_width_km", "10") // sensible default
		}
	} else if radius != "" || radiusKM != "" {
		die(fmt.Errorf("search: --radius / --radius-km requires --near lat,lng"))
	}

	if startsAfter != "" {
		h, m, err := parseHHMM(startsAfter)
		if err != nil {
			die(err)
		}
		p.Set("StartsAfterH", strconv.Itoa(h))
		p.Set("StartsAfterM", strconv.Itoa(m))
	}
	if startsBefore != "" {
		h, m, err := parseHHMM(startsBefore)
		if err != nil {
			die(err)
		}
		p.Set("StartsBeforeH", strconv.Itoa(h))
		p.Set("StartsBeforeM", strconv.Itoa(m))
	}
	if endsBefore != "" {
		h, m, err := parseHHMM(endsBefore)
		if err != nil {
			die(err)
		}
		p.Set("EndsBeforeH", strconv.Itoa(h))
		p.Set("EndsBeforeM", strconv.Itoa(m))
	}
	if minDur != "" {
		h, m, err := parseHHMM(minDur)
		if err != nil {
			die(err)
		}
		p.Set("MinDurationH", strconv.Itoa(h))
		p.Set("MinDurationM", strconv.Itoa(m))
	}
	if maxDur != "" {
		h, m, err := parseHHMM(maxDur)
		if err != nil {
			die(err)
		}
		p.Set("MaxDurationH", strconv.Itoa(h))
		p.Set("MaxDurationM", strconv.Itoa(m))
	}

	addStr(p, "data_field_key", strings.Join(parseList(fields), ","))
	addStr(p, "sort_keys", strings.Join(parseList(sortKeys), ","))
	addStr(p, "sort_key", sortKey)
	addInt(p, "page_size", pageSize)
	addInt(p, "page_num", pageNum)
	addStr(p, "lang_enum", lang)
	addCSV(p, "root_server_ids", parseList(rootIDs))

	if key != "" {
		p.Set("meeting_key", key)
		p.Set("meeting_key_value", keyValue)
	}

	switch published {
	case "":
		// default — published only
	case "all":
		p.Set("advanced_published", "0")
	case "unpublished":
		p.Set("advanced_published", "-1")
	default:
		die(fmt.Errorf("--published must be 'all' or 'unpublished'"))
	}

	if co.urlOnly {
		fmt.Println(c.BuildURL("GetSearchResults", p))
		return
	}

	body, _, err := c.Fetch("GetSearchResults", p)
	if err != nil {
		die(err)
	}

	if co.tsml || co.json {
		_ = printPrettyJSON(os.Stdout, body)
		return
	}
	if err := printMeetings(os.Stdout, body); err != nil {
		die(err)
	}
}

// --- helpers ---

func die(err error) {
	fmt.Fprintln(os.Stderr, "bmlt:", err)
	os.Exit(1)
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		die(err)
	}
	return b
}
