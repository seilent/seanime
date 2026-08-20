package cron

import (
	"errors"
	"os"
	"path/filepath"
	"seanime/internal/torrent_clients/torrent_client"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	diskFloorBytes  = 30 * 1024 * 1024 * 1024
	diskTargetBytes = 80 * 1024 * 1024 * 1024
)

func AutoCleanJob(ctx *JobCtx) {
	logger := ctx.App.Logger

	libraryPath, err := ctx.App.Database.GetLibraryPathFromSettings()
	if err != nil || libraryPath == "" {
		return
	}

	if _, err := os.Stat(libraryPath); err != nil {
		logger.Warn().Str("path", libraryPath).Msg("cron/auto-clean: Library path does not exist on disk, skipping")
		return
	}

	mappings, err := ctx.App.Database.GetAllGlobalMappings()
	if err != nil {
		return
	}

	removed := 0
	for _, m := range mappings {
		if _, err := os.Stat(m.LocalFilePath); errors.Is(err, os.ErrNotExist) {
			_ = ctx.App.Database.DeleteGlobalMapping(m.LocalFilePath)
			removed++
		}
	}
	if removed > 0 {
		logger.Info().Int("count", removed).Msg("cron/auto-clean: Removed stale mappings")
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(libraryPath, &stat); err != nil {
		logger.Warn().Err(err).Msg("cron/auto-clean: Statfs failed")
		return
	}
	freeBytes := stat.Bavail * uint64(stat.Bsize)

	if freeBytes >= diskFloorBytes {
		resumeErroredTorrents(ctx)
		return
	}

	type deletableItem struct {
		filePath      string
		aniListID     int
		episodeNumber int
		title         string
		watchedAt     time.Time
	}

	allMappings, err := ctx.App.Database.GetAllGlobalMappings()
	if err != nil {
		return
	}

	watchTimeCache := make(map[int]map[int]time.Time)

	var items []deletableItem
	for _, m := range allMappings {
		if m.EpisodeNumber <= 0 || m.AniListID <= 0 {
			continue
		}

		completed, err := ctx.App.Database.IsEpisodeCompletedByAllUsers(m.AniListID, m.EpisodeNumber)
		if err != nil || !completed {
			continue
		}

		wt, exists := watchTimeCache[m.AniListID]
		if !exists {
			wt, _ = ctx.App.Database.GetEpisodeWatchTimes(m.AniListID)
			if wt == nil {
				wt = make(map[int]time.Time)
			}
			watchTimeCache[m.AniListID] = wt
		}

		title := m.Title
		if title == "" {
			title = m.RomajiTitle
		}

		items = append(items, deletableItem{
			filePath:      m.LocalFilePath,
			aniListID:     m.AniListID,
			episodeNumber: m.EpisodeNumber,
			title:         title,
			watchedAt:     wt[m.EpisodeNumber],
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].watchedAt.Before(items[j].watchedAt)
	})

	if !ctx.App.TorrentClientRepository.Start() {
		logger.Warn().Msg("cron/auto-clean: Could not start torrent client")
		return
	}

	allTorrents, _ := ctx.App.TorrentClientRepository.GetList()

	for _, item := range items {
		if freeBytes >= diskTargetBytes {
			break
		}

		var matchedTorrent *struct {
			hash        string
			contentPath string
		}
		insideMultiFile := false

		for _, t := range allTorrents {
			if t.ContentPath == "" {
				continue
			}
			cp := strings.TrimSuffix(t.ContentPath, "/")
			if item.filePath == cp {
				matchedTorrent = &struct {
					hash        string
					contentPath string
				}{hash: t.Hash, contentPath: cp}
				break
			}
			if strings.HasPrefix(item.filePath, cp+"/") {
				insideMultiFile = true
				break
			}
		}

		if insideMultiFile {
			continue
		}

		if matchedTorrent != nil {
			_ = ctx.App.TorrentClientRepository.RemoveTorrents([]string{matchedTorrent.hash})
		} else {
			_ = os.Remove(item.filePath)
		}

		_ = ctx.App.Database.DeleteGlobalMapping(item.filePath)

		dir := filepath.Dir(item.filePath)
		if dir != libraryPath && strings.HasPrefix(dir, libraryPath) {
			entries, err := os.ReadDir(dir)
			if err == nil && len(entries) == 0 {
				_ = os.Remove(dir)
			}
		}

		var newStat syscall.Statfs_t
		if err := syscall.Statfs(libraryPath, &newStat); err == nil {
			freeBytes = newStat.Bavail * uint64(newStat.Bsize)
			logger.Info().
				Str("title", item.title).
				Int("episode", item.episodeNumber).
				Uint64("freeBytes", freeBytes).
				Msg("cron/auto-clean: Removed episode")
		}
	}
}

func resumeErroredTorrents(ctx *JobCtx) {
	if !ctx.App.TorrentClientRepository.Start() {
		return
	}

	torrents, err := ctx.App.TorrentClientRepository.GetList()
	if err != nil {
		return
	}

	var errorHashes []string
	for _, t := range torrents {
		if t.Status == torrent_client.TorrentStatusOther && t.Progress < 1.0 {
			errorHashes = append(errorHashes, t.Hash)
		}
	}

	if len(errorHashes) > 0 {
		_ = ctx.App.TorrentClientRepository.ResumeTorrents(errorHashes)
		ctx.App.Logger.Info().Int("count", len(errorHashes)).Msg("cron/auto-clean: Resumed errored torrents")
	}
}
