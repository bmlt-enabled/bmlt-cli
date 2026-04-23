package main

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestParseList(t *testing.T) {
	cases := map[string][]string{
		"":             nil,
		"a":            {"a"},
		"a,b,c":        {"a", "b", "c"},
		" a , b ,, c ": {"a", "b", "c"},
		"a,,b":         {"a", "b"},
	}
	for in, want := range cases {
		got := parseList(in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("parseList(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseWeekdays(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"mon,wed,fri", []string{"2", "4", "6"}},
		{"2,4,6", []string{"2", "4", "6"}},
		{"-sat,-sun", []string{"-7", "-1"}},
		{"Monday,TUESDAY", []string{"2", "3"}},
		{"thurs,thur,thursday,thu", []string{"5", "5", "5", "5"}},
	}
	for _, tc := range cases {
		got, err := parseWeekdays(tc.in)
		if err != nil {
			t.Errorf("parseWeekdays(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseWeekdays(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseWeekdaysErrors(t *testing.T) {
	for _, in := range []string{"funday", "8", "0", "-9"} {
		if _, err := parseWeekdays(in); err == nil {
			t.Errorf("parseWeekdays(%q) expected error, got nil", in)
		}
	}
}

func TestParseVenueTypes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"in-person", []string{"1"}},
		{"virtual,hybrid", []string{"2", "3"}},
		{"online", []string{"2"}},
		{"-virtual", []string{"-2"}},
		{"1,2,3", []string{"1", "2", "3"}},
	}
	for _, tc := range cases {
		got, err := parseVenueTypes(tc.in)
		if err != nil {
			t.Errorf("parseVenueTypes(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseVenueTypes(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseVenueTypesErrors(t *testing.T) {
	for _, in := range []string{"hologram", "4", "0"} {
		if _, err := parseVenueTypes(in); err == nil {
			t.Errorf("parseVenueTypes(%q) expected error, got nil", in)
		}
	}
}

func TestParseHHMM(t *testing.T) {
	cases := []struct {
		in      string
		h, m    int
		wantErr bool
	}{
		{"", 0, 0, false},
		{"18", 18, 0, false},
		{"18:30", 18, 30, false},
		{"00:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"24:00", 0, 0, true},
		{"12:60", 0, 0, true},
		{"abc", 0, 0, true},
	}
	for _, tc := range cases {
		h, m, err := parseHHMM(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseHHMM(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && (h != tc.h || m != tc.m) {
			t.Errorf("parseHHMM(%q) = %d:%d, want %d:%d", tc.in, h, m, tc.h, tc.m)
		}
	}
}

func TestParseLatLng(t *testing.T) {
	lat, lng, err := parseLatLng("39.96,-83.00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lat != "39.96" || lng != "-83.00" {
		t.Errorf("got (%q,%q), want (39.96,-83.00)", lat, lng)
	}

	for _, in := range []string{"", "39.96", "abc,def", "39.96,-83.00,extra"} {
		if _, _, err := parseLatLng(in); err == nil {
			t.Errorf("parseLatLng(%q) expected error", in)
		}
	}
}

func TestAddHelpers(t *testing.T) {
	p := url.Values{}
	addCSV(p, "weekdays", []string{"2", "4"})
	addCSV(p, "skip", nil)
	addStr(p, "q", "hello")
	addStr(p, "blank", "")
	addInt(p, "page", 3)
	addInt(p, "zero", 0)

	if got := p.Get("weekdays"); got != "2,4" {
		t.Errorf("weekdays = %q", got)
	}
	if p.Has("skip") || p.Has("blank") || p.Has("zero") {
		t.Errorf("zero/empty values leaked into params: %v", p)
	}
	if !strings.Contains(p.Encode(), "page=3") {
		t.Errorf("page=3 missing from %q", p.Encode())
	}
}
