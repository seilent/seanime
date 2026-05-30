package playbackmanager

import (
	"context"
	"errors"
	"fmt"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/continuity"
	"seanime/internal/database/db"
	discordrpc_presence "seanime/internal/discordrpc/presence"
	"seanime/internal/events"
	"seanime/internal/library/anime"
	"seanime/internal/platforms/anilist_platform"
	"seanime/internal/platforms/platform"
	"seanime/internal/util/result"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"github.com/samber/mo"
)

const (
	LocalFilePlayback      PlaybackType = "localfile"
	StreamPlayback         PlaybackType = "stream"
	ManualTrackingPlayback PlaybackType = "manual"
)

var playbackStatePool = sync.Pool{
	New: func() interface{} {
		return &PlaybackState{}
	},
}

type (
	PlaybackType string

	// PlaybackManager manages video playback progress tracking.
	// It receives and dispatch appropriate events for:
	//  - Syncing progress with AniList, etc.
	//  - Sending notifications to the client
	PlaybackManager struct {
		Logger            *zerolog.Logger
		Database          *db.Database
		continuityManager *continuity.Manager

		settings *Settings

		discordPresence            *discordrpc_presence.Presence // DiscordPresence is used to update the user's Discord presence
		wsEventManager             events.WSEventManagerInterface
		platform                   platform.Platform
		metadataProvider           metadata.Provider
		refreshAnimeCollectionFunc func() // This function is called to refresh the AniList collection
		mu                         sync.Mutex
		eventMu                    sync.RWMutex
		cancel                     context.CancelFunc

		// historyMap stores a PlaybackState whose state is "completed"
		historyMap          map[string]map[uint]PlaybackState
		currentPlaybackType PlaybackType
		currentUserID       uint // User ID for the current playback session

		autoPlayMu           sync.Mutex
		nextEpisodeLocalFile mo.Option[*anime.LocalFile] // The next episode's local file (for local file playback)

		// currentMediaListEntry for Local file playback & stream playback
		currentMediaListEntry mo.Option[*anilist.AnimeListEntry]

		// \/ Local file playback
		currentLocalFile             mo.Option[*anime.LocalFile]
		currentLocalFileWrapperEntry mo.Option[*anime.LocalFileWrapperEntry]

		// \/ Stream playback
		currentStreamEpisode      mo.Option[*anime.Episode]
		currentStreamMedia        mo.Option[*anilist.BaseAnime]
		currentStreamAniDbEpisode mo.Option[string]

		// \/ Manual progress tracking (non-integrated external player)
		manualTrackingCtx           context.Context
		manualTrackingCtxCancel     context.CancelFunc
		manualTrackingPlaybackState PlaybackState
		currentManualTrackingState  mo.Option[*ManualTrackingState]
		manualTrackingWg            sync.WaitGroup

		animeCollection mo.Option[*anilist.AnimeCollection]

		playbackStatusSubscribers *result.Map[string, *PlaybackStatusSubscriber]
	}

	// PlaybackStatusSubscriber provides a single event channel for all playback events
	PlaybackStatusSubscriber struct {
		EventCh  chan PlaybackEvent
		canceled atomic.Bool
	}

	// PlaybackEvent is the base interface for all playback events
	PlaybackEvent interface {
		Type() string
	}

	PlaybackStartingEvent struct {
		Filepath      string
		PlaybackType  PlaybackType
		Media         *anilist.BaseAnime
		AniDbEpisode  string
		EpisodeNumber int
		WindowTitle   string
	}

	// Local file playback events

	PlaybackStatusChangedEvent struct {
		State PlaybackState
	}

	VideoStartedEvent struct {
		Filename string
		Filepath string
	}

	VideoStoppedEvent struct {
		Reason string
	}

	VideoCompletedEvent struct {
		Filename string
	}

	// Stream playback events
	StreamStateChangedEvent struct {
		State PlaybackState
	}

	StreamStatusChangedEvent struct {
	}

	StreamStartedEvent struct {
		Filename string
		Filepath string
	}

	StreamStoppedEvent struct {
		Reason string
	}

	StreamCompletedEvent struct {
		Filename string
	}

	PlaybackStateType string

	// PlaybackState is used to keep track of the user's current video playback
	PlaybackState struct {
		EpisodeNumber        int     `json:"episodeNumber"`
		AniDbEpisode         string  `json:"aniDbEpisode"`
		MediaTitle           string  `json:"mediaTitle"`
		MediaCoverImage      string  `json:"mediaCoverImage"`
		MediaTotalEpisodes   int     `json:"mediaTotalEpisodes"`
		Filename             string  `json:"filename"`
		CompletionPercentage float64 `json:"completionPercentage"`
		CanPlayNext          bool    `json:"canPlayNext"`
		ProgressUpdated      bool    `json:"progressUpdated"`
		MediaId              int     `json:"mediaId"`
	}

	NewPlaybackManagerOptions struct {
		WSEventManager             events.WSEventManagerInterface
		SSEEventManager            *events.SSEEventManagerAdapter
		Logger                     *zerolog.Logger
		Platform                   platform.Platform
		MetadataProvider           metadata.Provider
		Database                   *db.Database
		RefreshAnimeCollectionFunc func()
		DiscordPresence            *discordrpc_presence.Presence
		ContinuityManager          *continuity.Manager
	}

	Settings struct {
		AutoPlayNextEpisode bool
	}
)

// Event type implementations
func (e PlaybackStatusChangedEvent) Type() string { return "playback_status_changed" }
func (e VideoStartedEvent) Type() string          { return "video_started" }
func (e VideoStoppedEvent) Type() string          { return "video_stopped" }
func (e VideoCompletedEvent) Type() string        { return "video_completed" }
func (e StreamStateChangedEvent) Type() string    { return "stream_state_changed" }
func (e StreamStatusChangedEvent) Type() string   { return "stream_status_changed" }
func (e StreamStartedEvent) Type() string         { return "stream_started" }
func (e StreamStoppedEvent) Type() string         { return "stream_stopped" }
func (e StreamCompletedEvent) Type() string       { return "stream_completed" }
func (e PlaybackStartingEvent) Type() string      { return "playback_starting" }

func New(opts *NewPlaybackManagerOptions) *PlaybackManager {
	// Prefer SSE over WebSocket if available
	var eventManager events.WSEventManagerInterface
	if opts.SSEEventManager != nil {
		eventManager = opts.SSEEventManager
	} else {
		eventManager = opts.WSEventManager
	}

	pm := &PlaybackManager{
		Logger:                       opts.Logger,
		Database:                     opts.Database,
		settings:                     &Settings{},
		discordPresence:              opts.DiscordPresence,
		wsEventManager:               eventManager,
		platform:                     opts.Platform,
		metadataProvider:             opts.MetadataProvider,
		refreshAnimeCollectionFunc:   opts.RefreshAnimeCollectionFunc,
		mu:                           sync.Mutex{},
		autoPlayMu:                   sync.Mutex{},
		eventMu:                      sync.RWMutex{},
		historyMap:                   make(map[string]map[uint]PlaybackState),
		nextEpisodeLocalFile:         mo.None[*anime.LocalFile](),
		currentStreamEpisode:         mo.None[*anime.Episode](),
		currentStreamMedia:           mo.None[*anilist.BaseAnime](),
		currentStreamAniDbEpisode:    mo.None[string](),
		animeCollection:              mo.None[*anilist.AnimeCollection](),
		currentManualTrackingState:   mo.None[*ManualTrackingState](),
		currentLocalFile:             mo.None[*anime.LocalFile](),
		currentLocalFileWrapperEntry: mo.None[*anime.LocalFileWrapperEntry](),
		currentMediaListEntry:        mo.None[*anilist.AnimeListEntry](),
		continuityManager:            opts.ContinuityManager,
		playbackStatusSubscribers:    result.NewResultMap[string, *PlaybackStatusSubscriber](),
	}

	return pm
}

func (pm *PlaybackManager) SetAnimeCollection(ac *anilist.AnimeCollection) {
	pm.animeCollection = mo.Some(ac)
}

func (pm *PlaybackManager) SetSettings(s *Settings) {
	pm.settings = s
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// GetNextEpisode gets the next [anime.LocalFile] of the local media that is being watched.
// It will return nil if there is no next episode.
func (pm *PlaybackManager) GetNextEpisode() (ret *anime.LocalFile) {
	defer func() {
		if r := recover(); r != nil {
			ret = nil
		}
	}()

	switch pm.currentPlaybackType {
	case LocalFilePlayback:
		if lf, found := pm.nextEpisodeLocalFile.Get(); found {
			ret = lf
		}
		return
	}

	return nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// getUserSpecificPlatform creates a user-specific platform instance for the current user
func (pm *PlaybackManager) getUserSpecificPlatform() (platform.Platform, error) {
	if pm.currentUserID == 0 {
		return nil, errors.New("no current user ID set")
	}

	// Get user's AniList account data
	account, err := pm.Database.GetAccountForUser(pm.currentUserID)
	if err != nil || account == nil || account.Token == "" {
		return nil, fmt.Errorf("user %d has no AniList connection: %v", pm.currentUserID, err)
	}

	// Create user-specific AniList platform
	client := anilist.NewAnilistClient(account.Token)
	userPlatform := anilist_platform.NewAnilistPlatform(client, pm.Logger)
	userPlatform.SetUsername(account.Username)

	return userPlatform, nil
}

func (pm *PlaybackManager) checkOrLoadAnimeCollection() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in checkOrLoadAnimeCollection: %v", r)
		}
	}()

	if pm.animeCollection.IsAbsent() {
		userPlatform, err := pm.getUserSpecificPlatform()
		if err != nil {
			pm.Logger.Error().Err(err).Uint("userID", pm.currentUserID).Msg("playback manager: Failed to get user-specific platform")
			return err
		}

		collection, err := userPlatform.GetAnimeCollection(context.Background(), false)
		if err != nil {
			pm.Logger.Error().Err(err).Uint("userID", pm.currentUserID).Msg("playback manager: Failed to get anime collection for user")
			return err
		}
		pm.animeCollection = mo.Some(collection)
		pm.Logger.Debug().Uint("userID", pm.currentUserID).Msg("playback manager: Loaded user-specific anime collection")
	}
	return nil
}

func (pm *PlaybackManager) SubscribeToPlaybackStatus(id string) *PlaybackStatusSubscriber {
	subscriber := &PlaybackStatusSubscriber{
		EventCh: make(chan PlaybackEvent, 100),
	}
	pm.playbackStatusSubscribers.Set(id, subscriber)
	return subscriber
}

func (pm *PlaybackManager) UnsubscribeFromPlaybackStatus(id string) {
	defer func() {
		if r := recover(); r != nil {
			pm.Logger.Warn().Msg("playback manager: Failed to unsubscribe from playback status")
		}
	}()
	subscriber, ok := pm.playbackStatusSubscribers.Get(id)
	if !ok {
		return
	}
	subscriber.canceled.Store(true)
	pm.playbackStatusSubscribers.Delete(id)
	close(subscriber.EventCh)
}
