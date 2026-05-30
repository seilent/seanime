package filehydrator

import (
	"seanime/internal/hook_resolver"
	"seanime/internal/library/anime"
)

// ScanHydrationStartedEvent is triggered when the file hydration process begins.
// Prevent default to skip the rest of the hydration process, in which case the event's local files will be used.
type ScanHydrationStartedEvent struct {
	hook_resolver.Event
	// Local files to be hydrated.
	LocalFiles []*anime.LocalFile `json:"localFiles"`
	// Media to be hydrated.
	AllMedia []*anime.NormalizedMedia `json:"allMedia"`
}

// ScanLocalFileHydrationStartedEvent is triggered when a local file's metadata is about to be hydrated.
// Prevent default to skip the default hydration and override the hydration.
type ScanLocalFileHydrationStartedEvent struct {
	hook_resolver.Event
	LocalFile *anime.LocalFile       `json:"localFile"`
	Media     *anime.NormalizedMedia `json:"media"`
}

// ScanLocalFileHydratedEvent is triggered when a local file's metadata is hydrated
type ScanLocalFileHydratedEvent struct {
	hook_resolver.Event
	LocalFile *anime.LocalFile `json:"localFile"`
	MediaId   int              `json:"mediaId"`
	Episode   int              `json:"episode"`
}
