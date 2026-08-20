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

	type candidate struct {
		aniListID     int
		title         string
		lastWatchedAt time.Time
	}

	allMappings, err := ctx.App.Database.GetAllGlobalMappings()
	if err != nil {
		return
	}

	aniListIDs := make(map[int]bool)
	for _, m := range allMappings {
		if m.AniListID > 0 {
			aniListIDs[m.AniListID] = true
		}
	}

	idSlice := make([]int, 0, len(aniListIDs))
	for id := range aniListIDs {
		idSlice = append(idSlice, id)
	}

	cachedMedia, err := ctx.App.Database.GetCachedMediaByIDs(idSlice)
	if err != nil {
		return
	}

	var candidates []candidate

	for _, id := range idSlice {
		cm, exists := cachedMedia[id]
		if exists && cm.Status == "RELEASING" {
			continue
		}

		subs, err := ctx.App.Database.GetUsersWithAnime(id)
		if err != nil || len(subs) == 0 {
			continue
		}

		mediaMappings, err := ctx.App.Database.GetGlobalMappingsByAniListID(id)
		if err != nil || len(mediaMappings) == 0 {
			continue
		}

		episodes := make(map[int]bool)
		for _, m := range mediaMappings {
			if m.EpisodeNumber > 0 {
				episodes[m.EpisodeNumber] = true
			}
		}

		if len(episodes) == 0 {
			continue
		}

		allWatched := true
		for _, sub := range subs {
			completed, err := ctx.App.Database.GetCompletedEpisodeNumbers(sub.UserID, id)
			if err != nil {
				allWatched = false
				break
			}
			completedSet := make(map[int]bool, len(completed))
			for _, ep := range completed {
				completedSet[ep] = true
			}
			for ep := range episodes {
				if !completedSet[ep] {
					allWatched = false
					break
				}
			}
			if !allWatched {
				break
			}
		}

		if !allWatched {
			continue
		}

		lastWatched, _ := ctx.App.Database.GetMaxLastWatchedAt(id)
		title := ""
		if cm != nil {
			title = cm.TitleRomaji
		}

		candidates = append(candidates, candidate{
			aniListID:     id,
			title:         title,
			lastWatchedAt: lastWatched,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].lastWatchedAt.Before(candidates[j].lastWatchedAt)
	})

	if !ctx.App.TorrentClientRepository.Start() {
		logger.Warn().Msg("cron/auto-clean: Could not start torrent client")
		return
	}

	allTorrents, _ := ctx.App.TorrentClientRepository.GetList()

	for _, c := range candidates {
		if freeBytes >= diskTargetBytes {
			break
		}

		mediaMappings, err := ctx.App.Database.GetGlobalMappingsByAniListID(c.aniListID)
		if err != nil || len(mediaMappings) == 0 {
			continue
		}

		filePaths := make(map[string]bool)
		for _, m := range mediaMappings {
			filePaths[m.LocalFilePath] = true
		}

		var hashesToRemove []string
		for _, t := range allTorrents {
			if t.ContentPath == "" {
				continue
			}
			cp := strings.TrimSuffix(t.ContentPath, "/")
			for fp := range filePaths {
				if fp == cp || strings.HasPrefix(fp, cp+"/") {
					hashesToRemove = append(hashesToRemove, t.Hash)
					break
				}
			}
		}

		if len(hashesToRemove) > 0 {
			_ = ctx.App.TorrentClientRepository.RemoveTorrents(hashesToRemove)
		}

		for fp := range filePaths {
			_ = os.Remove(fp)
		}

		_ = ctx.App.Database.DeleteGlobalMappingsByAniListID(c.aniListID)

		for fp := range filePaths {
			dir := filepath.Dir(fp)
			if dir != libraryPath && strings.HasPrefix(dir, libraryPath) {
				entries, err := os.ReadDir(dir)
				if err == nil && len(entries) == 0 {
					_ = os.Remove(dir)
				}
			}
		}

		var newStat syscall.Statfs_t
		if err := syscall.Statfs(libraryPath, &newStat); err == nil {
			newFree := newStat.Bavail * uint64(newStat.Bsize)
			freed := int64(0)
			if newFree > freeBytes {
				freed = int64(newFree - freeBytes)
			}
			freeBytes = newFree
			logger.Info().
				Str("title", c.title).
				Int64("freedBytes", freed).
				Uint64("freeBytes", freeBytes).
				Msg("cron/auto-clean: Removed anime")
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
