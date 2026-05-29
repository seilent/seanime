package playbackmanager

import (
	"context"
	"errors"
	"seanime/internal/events"
	"seanime/internal/util"
)

var (
	ErrProgressUpdateAnilist = errors.New("playback manager: Failed to update progress on AniList")
	ErrProgressUpdateMAL     = errors.New("playback manager: Failed to update progress on MyAnimeList")
)

// autoSyncCurrentProgress syncs the current video playback progress with providers.
// This is called once when a "video complete" event is heard.
func (pm *PlaybackManager) autoSyncCurrentProgress(_ps *PlaybackState) {

	shouldUpdate, err := pm.Database.AutoUpdateProgressIsEnabled()
	if err != nil {
		pm.Logger.Error().Err(err).Msg("playback manager: Failed to check if auto update progress is enabled")
		return
	}

	if !shouldUpdate {
		return
	}

	switch pm.currentPlaybackType {
	case LocalFilePlayback:
		if pm.currentMediaListEntry.IsAbsent() || pm.currentLocalFileWrapperEntry.IsAbsent() || pm.currentLocalFile.IsAbsent() {
			return
		}
		epProgressNum := pm.currentLocalFileWrapperEntry.MustGet().GetProgressNumber(pm.currentLocalFile.MustGet())
		if *pm.currentMediaListEntry.MustGet().Progress >= epProgressNum {
			return
		}

	case StreamPlayback:
		if pm.currentStreamEpisode.IsAbsent() || pm.currentStreamMedia.IsAbsent() {
			return
		}
		epProgressNum := pm.currentStreamEpisode.MustGet().GetProgressNumber()
		if pm.currentMediaListEntry.IsPresent() && *pm.currentMediaListEntry.MustGet().Progress >= epProgressNum {
			return
		}
	}

	pm.Logger.Debug().Msg("playback manager: Updating progress on AniList")
	err = pm.updateProgress()

	if err != nil {
		_ps.ProgressUpdated = false
		pm.wsEventManager.SendEventToUser(pm.currentUserID, events.ErrorToast, "Failed to update progress on AniList")
	} else {
		_ps.ProgressUpdated = true
		pm.wsEventManager.SendEventToUser(pm.currentUserID, events.PlaybackManagerProgressUpdated, _ps)
	}
}

// SyncCurrentProgress syncs the current video playback progress with providers.
// This method is called when the user manually requests to sync the progress.
func (pm *PlaybackManager) SyncCurrentProgress() error {
	pm.eventMu.RLock()

	err := pm.updateProgress()
	if err != nil {
		pm.eventMu.RUnlock()
		return err
	}

	pm.refreshAnimeCollectionFunc()

	pm.eventMu.RUnlock()
	return nil
}

// updateProgress updates the progress of the current video playback on AniList and MyAnimeList.
func (pm *PlaybackManager) updateProgress() (err error) {

	var mediaId int
	var epNum int
	var totalEpisodes int

	switch pm.currentPlaybackType {
	case LocalFilePlayback:
		if pm.currentLocalFileWrapperEntry.IsAbsent() || pm.currentLocalFile.IsAbsent() || pm.currentMediaListEntry.IsAbsent() {
			return errors.New("no video is being watched")
		}

		defer util.HandlePanicInModuleWithError("playbackmanager/updateProgress", &err)

		mediaId = pm.currentMediaListEntry.MustGet().GetMedia().GetID()
		epNum = pm.currentLocalFileWrapperEntry.MustGet().GetProgressNumber(pm.currentLocalFile.MustGet())
		totalEpisodes = pm.currentMediaListEntry.MustGet().GetMedia().GetTotalEpisodeCount()

	case StreamPlayback:
		if pm.currentStreamEpisode.IsAbsent() || pm.currentStreamMedia.IsAbsent() {
			return errors.New("no video is being watched")
		}

		mediaId = pm.currentStreamMedia.MustGet().ID
		epNum = pm.currentStreamEpisode.MustGet().GetProgressNumber()
		totalEpisodes = pm.currentStreamMedia.MustGet().GetTotalEpisodeCount()

	case ManualTrackingPlayback:
		if pm.currentManualTrackingState.IsAbsent() {
			return errors.New("no media file is being manually tracked")
		}

		defer func() {
			if pm.manualTrackingCtxCancel != nil {
				pm.manualTrackingCtxCancel()
			}
		}()

		mediaId = pm.currentManualTrackingState.MustGet().MediaId
		epNum = pm.currentManualTrackingState.MustGet().EpisodeNumber
		totalEpisodes = pm.currentManualTrackingState.MustGet().TotalEpisodes

	default:
		return errors.New("unknown playback type")
	}

	if mediaId == 0 {
		return errors.New("media ID not found")
	}

	userPlatform, err := pm.getUserSpecificPlatform()
	if err != nil {
		pm.Logger.Error().Err(err).Uint("userID", pm.currentUserID).Msg("playback manager: Failed to get user-specific platform for progress update")
		return err
	}

	err = userPlatform.UpdateEntryProgress(
		context.Background(),
		mediaId,
		epNum,
		&totalEpisodes,
	)
	if err != nil {
		pm.Logger.Error().Err(err).Msg("playback manager: Error occurred while updating progress on AniList")
		return ErrProgressUpdateAnilist
	}

	pm.refreshAnimeCollectionFunc()

	pm.Logger.Info().Msg("playback manager: Updated progress on AniList")

	return nil
}
