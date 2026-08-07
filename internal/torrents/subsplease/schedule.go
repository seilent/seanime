package subsplease

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

const scheduleURL = apiURL + "?f=schedule&tz=UTC"

type ScheduleShow struct {
	Title    string `json:"title"`
	Slug     string `json:"slug"`
	ImageURL string `json:"imageUrl"`
	Day      string `json:"day"`
	Time     string `json:"time"`
}

var (
	scheduleCacheMu   sync.RWMutex
	scheduleCacheData []ScheduleShow
	scheduleCacheTime time.Time
	scheduleCacheTTL  = 1 * time.Hour
)

func FetchSchedule() ([]ScheduleShow, error) {
	scheduleCacheMu.RLock()
	if scheduleCacheData != nil && time.Since(scheduleCacheTime) < scheduleCacheTTL {
		data := scheduleCacheData
		scheduleCacheMu.RUnlock()
		return data, nil
	}
	scheduleCacheMu.RUnlock()

	shows, err := fetchScheduleFromAPI()
	if err != nil {
		scheduleCacheMu.RLock()
		if scheduleCacheData != nil {
			data := scheduleCacheData
			scheduleCacheMu.RUnlock()
			return data, nil
		}
		scheduleCacheMu.RUnlock()
		return nil, err
	}

	scheduleCacheMu.Lock()
	scheduleCacheData = shows
	scheduleCacheTime = time.Now()
	scheduleCacheMu.Unlock()

	return shows, nil
}

type scheduleAPIResponse struct {
	Tz       string                       `json:"tz"`
	Schedule map[string][]scheduleAPIShow `json:"schedule"`
}

type scheduleAPIShow struct {
	Title    string `json:"title"`
	Page     string `json:"page"`
	ImageURL string `json:"image_url"`
	Time     string `json:"time"`
}

func fetchScheduleFromAPI() ([]ScheduleShow, error) {
	resp, err := showsHTTPClient.Get(scheduleURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("schedule API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp scheduleAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var shows []ScheduleShow
	for day, entries := range apiResp.Schedule {
		for _, entry := range entries {
			slug := strings.Trim(entry.Page, "/")
			if slug == "" {
				continue
			}
			if seen[slug] {
				continue
			}
			seen[slug] = true

			imageURL := entry.ImageURL
			if strings.HasPrefix(imageURL, "/") {
				imageURL = "https://subsplease.org" + imageURL
			}

			shows = append(shows, ScheduleShow{
				Title:    entry.Title,
				Slug:     slug,
				ImageURL: imageURL,
				Day:      day,
				Time:     entry.Time,
			})
		}
	}

	sort.Slice(shows, func(i, j int) bool {
		return strings.ToLower(shows[i].Title) < strings.ToLower(shows[j].Title)
	})

	if len(shows) == 0 {
		return nil, fmt.Errorf("no shows parsed from SubsPlease schedule")
	}

	return shows, nil
}
