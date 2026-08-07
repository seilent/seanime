package subsplease

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Show struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

var (
	showsCacheMu    sync.RWMutex
	showsCacheData  []Show
	showsCacheTime  time.Time
	showsCacheTTL   = 6 * time.Hour
	showsHTTPClient = &http.Client{Timeout: 15 * time.Second}
)

func FetchShows() ([]Show, error) {
	showsCacheMu.RLock()
	if showsCacheData != nil && time.Since(showsCacheTime) < showsCacheTTL {
		data := showsCacheData
		showsCacheMu.RUnlock()
		return data, nil
	}
	showsCacheMu.RUnlock()

	shows, err := fetchShowsFromSite()
	if err != nil {
		showsCacheMu.RLock()
		if showsCacheData != nil {
			data := showsCacheData
			showsCacheMu.RUnlock()
			return data, nil
		}
		showsCacheMu.RUnlock()
		return nil, err
	}

	showsCacheMu.Lock()
	showsCacheData = shows
	showsCacheTime = time.Now()
	showsCacheMu.Unlock()

	return shows, nil
}

func fetchShowsFromSite() ([]Show, error) {
	resp, err := showsHTTPClient.Get(showsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("shows page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var shows []Show
	for _, m := range showLinkRegex.FindAllStringSubmatch(string(body), -1) {
		slug := strings.TrimSuffix(m[1], "/")
		if seen[slug] {
			continue
		}
		seen[slug] = true
		title := html.UnescapeString(m[2])
		shows = append(shows, Show{Title: title, Slug: slug})
	}

	sort.Slice(shows, func(i, j int) bool {
		return strings.ToLower(shows[i].Title) < strings.ToLower(shows[j].Title)
	})

	if len(shows) == 0 {
		return nil, fmt.Errorf("no shows parsed from SubsPlease page")
	}

	return shows, nil
}
