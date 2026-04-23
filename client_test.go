package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeRoot(t *testing.T) {
	cases := map[string]string{
		"https://bmlt.example.org/main_server":                                "https://bmlt.example.org/main_server/",
		"https://bmlt.example.org/main_server/":                               "https://bmlt.example.org/main_server/",
		"  https://bmlt.example.org/main_server  ":                            "https://bmlt.example.org/main_server/",
		"https://bmlt.example.org/main_server/client_interface/json/?foo=bar": "https://bmlt.example.org/main_server/",
	}
	for in, want := range cases {
		if got := normalizeRoot(in); got != want {
			t.Errorf("normalizeRoot(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildURL(t *testing.T) {
	c := NewClient("https://bmlt.example.org/main_server/", time.Second)
	p := url.Values{}
	p.Set("weekdays", "2,4")
	p.Set("venue_types", "2")
	got := c.BuildURL("GetSearchResults", p)

	for _, want := range []string{
		"https://bmlt.example.org/main_server/client_interface/json/?",
		"switcher=GetSearchResults",
		"weekdays=2%2C4",
		"venue_types=2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("URL missing %q\n  got: %s", want, got)
		}
	}
}

func TestBuildURLNilParams(t *testing.T) {
	c := NewClient("https://bmlt.example.org/main_server/", time.Second)
	got := c.BuildURL("GetServerInfo", nil)
	want := "https://bmlt.example.org/main_server/client_interface/json/?switcher=GetServerInfo"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildURLFormatSwitch(t *testing.T) {
	c := NewClient("https://bmlt.example.org/main_server/", time.Second)
	c.Format = "csv"
	got := c.BuildURL("GetNAWSDump", nil)
	if !strings.Contains(got, "/client_interface/csv/?") {
		t.Errorf("expected csv format in URL, got %q", got)
	}
}

func TestFetchSendsExpectedRequest(t *testing.T) {
	var gotPath, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/main_server/", time.Second)
	body, _, err := c.Fetch("GetServerInfo", nil)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if string(body) != `[{"id":"1"}]` {
		t.Errorf("unexpected body: %s", body)
	}
	if !strings.Contains(gotPath, "switcher=GetServerInfo") {
		t.Errorf("server didn't see switcher param, got path %q", gotPath)
	}
	if !strings.HasPrefix(gotUA, "bmlt-cli/") {
		t.Errorf("expected bmlt-cli User-Agent, got %q", gotUA)
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"error":"unprocessable"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/main_server/", time.Second)
	_, _, err := c.Fetch("GetSearchResults", nil)
	if err == nil {
		t.Fatal("expected error on 422")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("expected status code in error, got %v", err)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("short string should pass through, got %q", got)
	}
	if got := truncate("hello world", 5); got != "hello…" {
		t.Errorf("long string should be truncated with ellipsis, got %q", got)
	}
}
