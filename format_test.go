package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWeekdayLabel(t *testing.T) {
	cases := map[string]string{
		"1": "Sunday", "2": "Monday", "3": "Tuesday", "4": "Wednesday",
		"5": "Thursday", "6": "Friday", "7": "Saturday",
		"0": "Unknown", "8": "Unknown", "": "Unknown", "abc": "Unknown",
	}
	for in, want := range cases {
		if got := weekdayLabel(in); got != want {
			t.Errorf("weekdayLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVenueLabel(t *testing.T) {
	cases := map[string]string{
		"1": "in-person", "2": "virtual", "3": "hybrid", "": "", "9": "",
	}
	for in, want := range cases {
		if got := venueLabel(in); got != want {
			t.Errorf("venueLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTrimTime(t *testing.T) {
	cases := map[string]string{
		"19:30:00": "19:30",
		"19:30":    "19:30",
		"abc":      "abc",
		"":         "",
	}
	for in, want := range cases {
		if got := trimTime(in); got != want {
			t.Errorf("trimTime(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPrintMeetingsGroupsByDay(t *testing.T) {
	body := []byte(`[
		{"id_bigint":"1","meeting_name":"Alpha","weekday_tinyint":"2","start_time":"19:30:00","location_text":"Hall","location_municipality":"Columbus","location_province":"OH","venue_type":"1"},
		{"id_bigint":"2","meeting_name":"Bravo","weekday_tinyint":"2","start_time":"12:00:00","location_text":"Annex","venue_type":"2"},
		{"id_bigint":"3","meeting_name":"Charlie","weekday_tinyint":"4","start_time":"20:00:00","location_text":"Church","venue_type":"3"}
	]`)
	var buf bytes.Buffer
	if err := printMeetings(&buf, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"Monday (2)",
		"Wednesday (1)",
		"Alpha",
		"Bravo",
		"Charlie",
		"19:30",
		"[in-person]",
		"[virtual]",
		"[hybrid]",
		"3 meetings",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}

	// Within a day, earlier time should appear before later time.
	bravoIdx := strings.Index(out, "Bravo")
	alphaIdx := strings.Index(out, "Alpha")
	if bravoIdx < 0 || alphaIdx < 0 || bravoIdx > alphaIdx {
		t.Errorf("expected Bravo (12:00) to print before Alpha (19:30)\n%s", out)
	}
}

func TestPrintMeetingsEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := printMeetings(&buf, []byte(`[]`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No meetings") {
		t.Errorf("expected 'No meetings' message, got %q", buf.String())
	}
}

func TestPrintServiceBodiesTree(t *testing.T) {
	body := []byte(`[
		{"id":"1","name":"Ohio Region","type":"RS","parent_id":"0"},
		{"id":"5","name":"Central Ohio Area","type":"AS","parent_id":"1"},
		{"id":"6","name":"Dayton Area","type":"AS","parent_id":"1"},
		{"id":"99","name":"Orphan Area","type":"AS","parent_id":"unknown"}
	]`)
	var buf bytes.Buffer
	if err := printServiceBodies(&buf, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"[RS] Ohio Region",
		"  [AS] Central Ohio Area",
		"  [AS] Dayton Area",
		"[AS] Orphan Area", // orphan promoted to root
		"4 service bodies",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestPrintFormatsTable(t *testing.T) {
	body := []byte(`[
		{"id":"1","key_string":"B","name_string":"Beginners","description_string":"Newcomer-focused"},
		{"id":"4","key_string":"C","name_string":"Closed","description_string":"Closed to non-addicts"}
	]`)
	var buf bytes.Buffer
	if err := printFormats(&buf, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"ID", "KEY", "Beginners", "Closed", "2 formats"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestPrintFieldKeys(t *testing.T) {
	body := []byte(`[
		{"key":"meeting_name","description":"Meeting Name"},
		{"key":"start_time","description":"Start Time"}
	]`)
	var buf bytes.Buffer
	if err := printFieldKeys(&buf, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"meeting_name", "Meeting Name", "start_time", "Start Time"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestPrintFieldValues(t *testing.T) {
	body := []byte(`[
		{"location_municipality":"Columbus","ids":"1,2,3,4"},
		{"location_municipality":"Dayton","ids":"5,6"}
	]`)
	var buf bytes.Buffer
	if err := printFieldValues(&buf, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Columbus", "(4)", "Dayton", "(2)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestPrintPrettyJSONPassthroughOnInvalid(t *testing.T) {
	var buf bytes.Buffer
	raw := []byte(`not actually json`)
	if err := printPrettyJSON(&buf, raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != string(raw) {
		t.Errorf("expected passthrough, got %q", buf.String())
	}
}
