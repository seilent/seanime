package continuity

import (
	"time"
)

const (
	IgnoreRatioThreshold = 0.9 // Keep this as it might be used elsewhere
)

type (
	// WatchHistory is a map of WatchHistoryItem.
	// The key is the WatchHistoryItem.MediaId.
	WatchHistory map[int]*WatchHistoryItem

	// WatchHistoryItem are stored in the file cache.
	// The history is used to resume playback from the last known position.
	// Item.MediaId and Item.EpisodeNumber are used to identify the media and episode.
	// Only one Item per MediaId should exist in the history.
	WatchHistoryItem struct {
		Kind Kind `json:"kind"`
		// Used for MediastreamKind and ExternalPlayerKind.
		Filepath      string `json:"filepath"`
		MediaId       int    `json:"mediaId"`
		EpisodeNumber int    `json:"episodeNumber"`
		// The current playback time in seconds.
		// Used to determine when to remove the item from the history.
		CurrentTime float64 `json:"currentTime"`
		// The duration of the media in seconds.
		Duration float64 `json:"duration"`
		// Timestamp of when the item was added to the history.
		TimeAdded time.Time `json:"timeAdded"`
		// TimeAdded is used in conjunction with TimeUpdated
		// Timestamp of when the item was last updated.
		// Used to determine when to remove the item from the history (First in, first out).
		TimeUpdated time.Time `json:"timeUpdated"`
	}

	WatchHistoryItemResponse struct {
		Item  *WatchHistoryItem `json:"item"`
		Found bool              `json:"found"`
	}

	UpdateWatchHistoryItemOptions struct {
		CurrentTime   float64 `json:"currentTime"`
		Duration      float64 `json:"duration"`
		MediaId       int     `json:"mediaId"`
		EpisodeNumber int     `json:"episodeNumber"`
		Filepath      string  `json:"filepath,omitempty"`
		Kind          Kind    `json:"kind"`
	}
)

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// NOTE: Legacy file cache methods have been removed and replaced with database-backed
// user-aware methods. The old method signatures are maintained as bridge methods in manager.go
// that delegate to the new user-specific database methods.
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////


//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// NOTE: GetExternalPlayerEpisodeWatchHistoryItem and UpdateExternalPlayerEpisodeWatchHistoryItem
// have been removed as they were part of the legacy file cache system. External player
// functionality now goes through the user-aware database methods via the bridge methods.
