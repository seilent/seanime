package cron

import (
	"errors"
	"os"
	"path/filepath"
	"seanime/internal/database/models"
	"seanime/internal/mediastream/videofile"
	"seanime/internal/torrent_clients/torrent_client"
	"seanime/internal/util"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	diskFloorBytes  = 30 * 1024 * 1024 * 1024
	diskTargetBytes = 80 * 1024 * 1024 * 1024

	neverWatchedGraceDays   = 14
	abandonedGraceDays      = 60
	recentActivityGuardDays = 14
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

	cleanVideofilesCache(ctx)

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

	evictCacheForSpace(ctx)

	if err := syscall.Statfs(libraryPath, &stat); err != nil {
		logger.Warn().Err(err).Msg("cron/auto-clean: Statfs failed after cache eviction")
		return
	}
	freeBytes = stat.Bavail * uint64(stat.Bsize)
	if freeBytes >= diskFloorBytes {
		return
	}

	allMappings, err := ctx.App.Database.GetAllGlobalMappings()
	if err != nil {
		return
	}

	activityTimes, err := ctx.App.Database.GetMediaActivityTimes()
	if err != nil {
		logger.Warn().Err(err).Msg("cron/auto-clean: Failed to load activity times, skipping cleanup")
		return
	}

	animeGroups := make(map[int][]*models.GlobalAnimeFileMapping)
	for _, m := range allMappings {
		if m.AniListID <= 0 {
			continue
		}
		animeGroups[m.AniListID] = append(animeGroups[m.AniListID], m)
	}

	type tierACandidate struct {
		aniListID int
		refTime   time.Time
		mappings  []*models.GlobalAnimeFileMapping
	}

	var tierACandidates []tierACandidate
	now := time.Now()

	for aniListID, group := range animeGroups {
		activity, hasActivity := activityTimes[aniListID]

		if hasActivity {
			if time.Since(activity) <= time.Duration(abandonedGraceDays)*24*time.Hour {
				continue
			}
			tierACandidates = append(tierACandidates, tierACandidate{
				aniListID: aniListID,
				refTime:   activity,
				mappings:  group,
			})
		} else {
			var oldestCreated time.Time
			for _, m := range group {
				if oldestCreated.IsZero() || m.CreatedAt.Before(oldestCreated) {
					oldestCreated = m.CreatedAt
				}
			}
			if now.Sub(oldestCreated) <= time.Duration(neverWatchedGraceDays)*24*time.Hour {
				continue
			}
			tierACandidates = append(tierACandidates, tierACandidate{
				aniListID: aniListID,
				refTime:   oldestCreated,
				mappings:  group,
			})
		}
	}

	sort.Slice(tierACandidates, func(i, j int) bool {
		return tierACandidates[i].refTime.Before(tierACandidates[j].refTime)
	})

	if !ctx.App.TorrentClientRepository.Start() {
		logger.Warn().Msg("cron/auto-clean: Could not start torrent client")
		return
	}

	allTorrents, _ := ctx.App.TorrentClientRepository.GetList()

	deletedAnime := make(map[int]bool)

	for _, candidate := range tierACandidates {
		if freeBytes >= diskTargetBytes {
			break
		}

		filePaths := make(map[string]bool)
		for _, m := range candidate.mappings {
			filePaths[m.LocalFilePath] = true
		}

		var hashesToRemove []string
		for _, t := range allTorrents {
			if t.ContentPath == "" {
				continue
			}
			cp := strings.TrimSuffix(t.ContentPath, "/")

			if filePaths[cp] {
				hashesToRemove = append(hashesToRemove, t.Hash)
				continue
			}

			for fp := range filePaths {
				if strings.HasPrefix(fp, cp+"/") {
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

		_ = ctx.App.Database.DeleteGlobalMappingsByAniListID(candidate.aniListID)
		deletedAnime[candidate.aniListID] = true

		removedDirs := make(map[string]bool)
		for fp := range filePaths {
			dir := filepath.Dir(fp)
			if removedDirs[dir] {
				continue
			}
			if dir != libraryPath && strings.HasPrefix(dir, libraryPath) {
				entries, err := os.ReadDir(dir)
				if err == nil && len(entries) == 0 {
					_ = os.Remove(dir)
					removedDirs[dir] = true
				}
			}
		}

		var newStat syscall.Statfs_t
		if err := syscall.Statfs(libraryPath, &newStat); err == nil {
			freeBytes = newStat.Bavail * uint64(newStat.Bsize)
		}

		title := candidate.mappings[0].Title
		if title == "" {
			title = candidate.mappings[0].RomajiTitle
		}

		logger.Info().
			Str("title", title).
			Uint64("freeBytes", freeBytes).
			Msg("cron/auto-clean: Removed abandoned anime")
	}

	if freeBytes >= diskTargetBytes {
		return
	}

	type deletableItem struct {
		filePath      string
		aniListID     int
		episodeNumber int
		title         string
		watchedAt     time.Time
	}

	watchTimeCache := make(map[int]map[int]time.Time)

	var items []deletableItem
	for _, m := range allMappings {
		if m.EpisodeNumber <= 0 || m.AniListID <= 0 || deletedAnime[m.AniListID] {
			continue
		}

		if activity, ok := activityTimes[m.AniListID]; ok {
			if time.Since(activity) <= time.Duration(recentActivityGuardDays)*24*time.Hour {
				continue
			}
		}

		completed, err := ctx.App.Database.IsEpisodeCompletedByActiveWatchers(m.AniListID, m.EpisodeNumber)
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

func cleanVideofilesCache(ctx *JobCtx) {
	logger := ctx.App.Logger
	cacheDir := ctx.App.Config.Cache.Dir
	if cacheDir == "" {
		return
	}

	videofilesDir := filepath.Join(cacheDir, "videofiles")
	entries, err := os.ReadDir(videofilesDir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		return
	}

	allMappings, err := ctx.App.Database.GetAllGlobalMappings()
	if err != nil {
		return
	}

	liveHashes := make(map[string]struct{})
	for _, m := range allMappings {
		hash, err := videofile.GetHashFromPath(m.LocalFilePath)
		if err != nil {
			continue
		}
		liveHashes[hash] = struct{}{}
	}

	var activeHashes map[string]struct{}
	if ctx.App.MediastreamRepository != nil {
		activeHashes = ctx.App.MediastreamRepository.ActiveVideoFileHashes()
	}
	if activeHashes == nil {
		activeHashes = make(map[string]struct{})
	}

	var freedBytes int64
	var freedCount int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, live := liveHashes[name]; live {
			continue
		}
		if _, active := activeHashes[name]; active {
			continue
		}
		if info, err := entry.Info(); err == nil && time.Since(info.ModTime()) < time.Hour {
			continue
		}
		dirPath := filepath.Join(videofilesDir, name)
		size := dirSize(dirPath)
		if err := os.RemoveAll(dirPath); err == nil {
			freedBytes += size
			freedCount++
		}
	}

	if freedCount > 0 {
		logger.Info().Int("count", freedCount).Int64("bytes", freedBytes).Msg("cron/auto-clean: Removed orphaned cache dirs")
	}
}

func evictCacheForSpace(ctx *JobCtx) {
	logger := ctx.App.Logger
	cacheDir := ctx.App.Config.Cache.Dir
	if cacheDir == "" {
		return
	}

	videofilesDir := filepath.Join(cacheDir, "videofiles")
	entries, err := os.ReadDir(videofilesDir)
	if err != nil || len(entries) == 0 {
		return
	}

	cacheFree, err := util.GetFreeSpace(videofilesDir)
	if err != nil {
		return
	}
	if cacheFree >= diskTargetBytes {
		return
	}

	var activeHashes map[string]struct{}
	if ctx.App.MediastreamRepository != nil {
		activeHashes = ctx.App.MediastreamRepository.ActiveVideoFileHashes()
	}
	if activeHashes == nil {
		activeHashes = make(map[string]struct{})
	}

	allMappings, _ := ctx.App.Database.GetAllGlobalMappings()
	hashToMapping := make(map[string]*models.GlobalAnimeFileMapping)
	for _, m := range allMappings {
		hash, err := videofile.GetHashFromPath(m.LocalFilePath)
		if err != nil {
			continue
		}
		hashToMapping[hash] = m
	}

	type cacheEntry struct {
		hash       string
		mp4Path    string
		mp4Size    int64
		lastWatch  time.Time
		dirModTime time.Time
	}

	var candidates []cacheEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, active := activeHashes[name]; active {
			continue
		}

		dirPath := filepath.Join(videofilesDir, name)
		mp4Path := filepath.Join(dirPath, "direct.mp4")
		mp4Info, mp4Err := os.Stat(mp4Path)
		if mp4Err != nil {
			continue
		}

		var dirModTime time.Time
		if info, err := entry.Info(); err == nil {
			dirModTime = info.ModTime()
			if time.Since(dirModTime) < time.Hour {
				continue
			}
		}

		var lastWatch time.Time
		if m, ok := hashToMapping[name]; ok {
			lw, err := ctx.App.Database.GetEpisodeLastWatched(m.AniListID, m.EpisodeNumber)
			if err == nil && !lw.IsZero() {
				lastWatch = lw
			}
		}

		if lastWatch.IsZero() {
			lastWatch = dirModTime
		}

		candidates = append(candidates, cacheEntry{
			hash:       name,
			mp4Path:    mp4Path,
			mp4Size:    mp4Info.Size(),
			lastWatch:  lastWatch,
			dirModTime: dirModTime,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].lastWatch.Before(candidates[j].lastWatch)
	})

	freeBytes := cacheFree
	for _, c := range candidates {
		if freeBytes >= diskTargetBytes {
			break
		}
		if err := os.Remove(c.mp4Path); err != nil {
			continue
		}
		freeBytes += uint64(c.mp4Size)
		logger.Info().Str("hash", c.hash).Int64("bytes", c.mp4Size).Msg("cron/auto-clean: Evicted direct.mp4 for space")

		free, err := util.GetFreeSpace(videofilesDir)
		if err == nil {
			freeBytes = free
		}
	}
}

func dirSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil {
			size += info.Size()
		}
		return nil
	})
	return size
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
