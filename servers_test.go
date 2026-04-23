package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

var sampleServers = []ServerEntry{
	{ID: "1", Name: "Ohio Region", URL: "https://bmlt.naohio.org/main_server/"},
	{ID: "2", Name: "Buckeye Region", URL: "https://bmlt.buckeye-na.org/main_server/"},
	{ID: "3", Name: "Southeastern Zonal Forum", URL: "https://bmlt.sezf.org/main_server/"},
	{ID: "4", Name: "Aotearoa New Zealand Region", URL: "https://bmlt.nzna.org/main_server/"},
}

func TestFilterServersByName(t *testing.T) {
	got := filterServers(sampleServers, "ohio")
	if len(got) != 1 || got[0].ID != "1" {
		t.Errorf("expected single Ohio match, got %+v", got)
	}
}

func TestFilterServersByURL(t *testing.T) {
	got := filterServers(sampleServers, "sezf")
	if len(got) != 1 || got[0].ID != "3" {
		t.Errorf("expected SEZF match by URL, got %+v", got)
	}
}

func TestFilterServersCaseInsensitive(t *testing.T) {
	got := filterServers(sampleServers, "REGION")
	names := []string{}
	for _, s := range got {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	want := []string{"Aotearoa New Zealand Region", "Buckeye Region", "Ohio Region"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestFilterServersEmptyQuery(t *testing.T) {
	got := filterServers(sampleServers, "")
	if len(got) != len(sampleServers) {
		t.Errorf("empty query should pass everything through, got %d/%d", len(got), len(sampleServers))
	}
}

func TestFilterServersNoMatch(t *testing.T) {
	got := filterServers(sampleServers, "antarctica")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}

// matchServers replicates lookupServer's matching logic without network.
func matchServers(servers []ServerEntry, query string) (*ServerEntry, error) {
	q := strings.ToLower(query)
	var matches []ServerEntry
	for _, s := range servers {
		if strings.Contains(strings.ToLower(s.Name), q) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil, errNoMatch
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}
	for i := range matches {
		if strings.EqualFold(matches[i].Name, query) {
			return &matches[i], nil
		}
	}
	return nil, errAmbiguous
}

var (
	errNoMatch   = &lookupErr{msg: "no match"}
	errAmbiguous = &lookupErr{msg: "ambiguous"}
)

type lookupErr struct{ msg string }

func (e *lookupErr) Error() string { return e.msg }

func TestLookupExactMatchPreferred(t *testing.T) {
	// "Ohio Region" matches "Ohio Region" exactly even though "Buckeye Region"
	// also contains the substring "egion" — but more relevantly, exact match
	// should win in any ambiguous case.
	servers := []ServerEntry{
		{ID: "1", Name: "Ohio Region", URL: "a"},
		{ID: "2", Name: "Ohio Region North", URL: "b"},
	}
	m, err := matchServers(servers, "Ohio Region")
	if err != nil {
		t.Fatalf("expected exact match, got error %v", err)
	}
	if m.ID != "1" {
		t.Errorf("expected ID=1, got ID=%s", m.ID)
	}
}

func TestLookupAmbiguous(t *testing.T) {
	servers := []ServerEntry{
		{ID: "1", Name: "Ohio Region", URL: "a"},
		{ID: "2", Name: "Ohio Region North", URL: "b"},
	}
	if _, err := matchServers(servers, "Ohio"); err != errAmbiguous {
		t.Errorf("expected ambiguous error, got %v", err)
	}
}
