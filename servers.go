package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const serverListURL = "https://raw.githubusercontent.com/bmlt-enabled/aggregator/refs/heads/main/serverList.json"

// cacheTTL — the canonical list rarely changes. 24h is fine.
const cacheTTL = 24 * time.Hour

type ServerEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func cachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "bmlt-cli", "serverList.json")
}

func loadServers() ([]ServerEntry, error) {
	path := cachePath()
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < cacheTTL {
		if data, err := os.ReadFile(path); err == nil {
			var list []ServerEntry
			if json.Unmarshal(data, &list) == nil && len(list) > 0 {
				return list, nil
			}
		}
	}
	return fetchServers()
}

func fetchServers() ([]ServerEntry, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(serverListURL)
	if err != nil {
		return nil, fmt.Errorf("fetch server list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server list HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var list []ServerEntry
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse server list: %w", err)
	}
	// Best-effort write to cache; ignore errors.
	if path := cachePath(); path != "" {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, data, 0o644)
	}
	return list, nil
}

// lookupServer finds a single server entry by fuzzy substring match on name.
// Errors if zero or multiple ambiguous matches.
func lookupServer(query string) (*ServerEntry, error) {
	servers, err := loadServers()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var matches []ServerEntry
	for _, s := range servers {
		if strings.Contains(strings.ToLower(s.Name), q) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no server matches %q (try `bmlt servers` to browse)", query)
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}
	// Prefer exact (case-insensitive) match if present.
	for i := range matches {
		if strings.EqualFold(matches[i].Name, query) {
			return &matches[i], nil
		}
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("ambiguous server query %q — matches:\n  %s", query, strings.Join(names, "\n  "))
}

func filterServers(servers []ServerEntry, query string) []ServerEntry {
	if query == "" {
		return servers
	}
	q := strings.ToLower(query)
	out := make([]ServerEntry, 0)
	for _, s := range servers {
		if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.URL), q) {
			out = append(out, s)
		}
	}
	return out
}
