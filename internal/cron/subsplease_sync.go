package cron

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/extension"
	hibiketorrent "seanime/internal/extension/hibike/torrent"
	"strings"
)

// SubsPleaseSyncJob syncs all releasing anime from SubsPlease.
// For each airing anime in the cache that has local files, it downloads
// missing episodes from SubsPlease and replaces non-SubsPlease files.
func SubsPleaseSyncJob(ctx *JobCtx) {
	logger := ctx.App.Logger

	// Get SubsPlease provider
	providerExt, ok := extension.GetExtension[extension.AnimeTorrentProviderExtension](
		ctx.App.ExtensionRepository.GetExtensionBank(), "subsplease",
	)
	if !ok {
		logger.Warn().Msg("cron/subsplease-sync: SubsPlease provider not found")
		return
	}

	// Get all releasing anime IDs that have local files (are in the library)
	releasingIDs, err := ctx.App.Database.GetReleasingAnimeIDsInLibrary()
	if err != nil {
		logger.Error().Err(err).Msg("cron/subsplease-sync: Failed to get releasing anime")
		return
	}

	if len(releasingIDs) == 0 {
		return
	}

	// Get cached media for metadata (titles, synonyms, episode count)
	cachedMedia, err := ctx.App.Database.GetCachedMediaByIDs(releasingIDs)
	if err != nil {
		logger.Error().Err(err).Msg("cron/subsplease-sync: Failed to get cached media")
		return
	}

	// Get library path from settings
	libraryPath, err := ctx.App.Database.GetLibraryPathFromSettings()
	if err != nil || libraryPath == "" {
		logger.Warn().Msg("cron/subsplease-sync: No library path configured")
		return
	}

	// Try to start torrent client
	if !ctx.App.TorrentClientRepository.Start() {
		logger.Warn().Msg("cron/subsplease-sync: Could not start torrent client")
		return
	}

	totalDownloaded := 0

	for _, id := range releasingIDs {
		cached, exists := cachedMedia[id]
		if !exists {
			continue
		}

		// Decode the cached media data to get full BaseAnime
		var media anilist.BaseAnime
		if err := json.Unmarshal(cached.Data, &media); err != nil {
			continue
		}

		status := media.GetStatus()
		format := media.GetFormat()
		if status == nil || format == nil {
			continue
		}

		// Build query media
		queryMedia := hibiketorrent.Media{
			ID:           media.GetID(),
			Status:       string(*status),
			Format:       string(*format),
			EnglishTitle: media.GetTitle().GetEnglish(),
			RomajiTitle:  media.GetRomajiTitleSafe(),
			EpisodeCount: media.GetTotalEpisodeCount(),
			Synonyms:     media.GetSynonymsDeref(),
		}

		// Fetch all episodes from SubsPlease
		torrents, err := providerExt.GetProvider().SmartSearch(hibiketorrent.AnimeSmartSearchOptions{
			Media:         queryMedia,
			EpisodeNumber: 0,
		})
		if err != nil || len(torrents) == 0 {
			continue
		}

		// Cache the SubsPlease sid if not already cached
		if cached.SubsPleaseSid == "" {
			_ = ctx.App.Database.SetSubsPleaseSid(id, "found")
		}
		// Always update episode count
		_ = ctx.App.Database.SetSubspleaseEpisodeCount(id, len(torrents))

		// Get local files for this media
		localFiles, _ := db_bridge.GetLocalFilesByMediaId(ctx.App.Database, id)
		syncedEps := make(map[int]bool)
		for _, lf := range localFiles {
			if strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
				syncedEps[lf.EpisodeNumber] = true
			}
		}

		// Find episodes to sync
		var toSync []*hibiketorrent.AnimeTorrent
		for _, t := range torrents {
			if t.EpisodeNumber > 0 && !syncedEps[t.EpisodeNumber] {
				toSync = append(toSync, t)

				// Delete old non-SubsPlease file
				for _, lf := range localFiles {
					if lf.EpisodeNumber == t.EpisodeNumber && !strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
						_ = os.Remove(lf.LocalFilePath)
						_ = ctx.App.Database.DeleteGlobalMapping(lf.LocalFilePath)
					}
				}
			}
		}

		if len(toSync) == 0 {
			continue
		}

		// Add magnets to torrent client
		magnets := make([]string, 0, len(toSync))
		for _, t := range toSync {
			magnets = append(magnets, t.MagnetLink)
		}

		err = ctx.App.TorrentClientRepository.AddMagnets(magnets, libraryPath)
		if err != nil {
			logger.Error().Err(err).Int("mediaId", id).Msg("cron/subsplease-sync: Failed to add magnets")
			continue
		}

		// Record download intents
		for _, t := range toSync {
			hash := t.InfoHash
			if hash == "" {
				hash = parseHashFromMagnet(t.MagnetLink)
			}
			_ = ctx.App.Database.UpsertPendingDownloadIntent(hash, id, libraryPath)
		}

		totalDownloaded += len(toSync)
		logger.Info().Int("mediaId", id).Str("title", cached.TitleRomaji).Int("episodes", len(toSync)).
			Msg("cron/subsplease-sync: Queued episodes")
	}

	if totalDownloaded > 0 {
		logger.Info().Int("total", totalDownloaded).Msg("cron/subsplease-sync: Sync complete")
	}
}

func parseHashFromMagnet(magnet string) string {
	idx := strings.Index(magnet, "btih:")
	if idx == -1 {
		return ""
	}
	hash := magnet[idx+5:]
	if ampIdx := strings.Index(hash, "&"); ampIdx != -1 {
		hash = hash[:ampIdx]
	}
	if len(hash) == 32 {
		decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(hash))
		if err == nil {
			return strings.ToLower(fmt.Sprintf("%x", decoded))
		}
	}
	return strings.ToLower(hash)
}
