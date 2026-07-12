package cron

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/torrents/subsplease"
	"strconv"
	"strings"

	"github.com/mmcdole/gofeed"
)

const subspleaseRSS = "https://subsplease.org/rss/?t&r=1080"

var spEpisodeRegex = regexp.MustCompile(`- (\d+) \(`)
var spInfoHashRegex = regexp.MustCompile(`btih:([0-9a-zA-Z]+)`)

// SubsPleaseSyncJob polls the SubsPlease RSS feed for new episodes
// and downloads any that match anime in the library.
func SubsPleaseSyncJob(ctx *JobCtx) {
	logger := ctx.App.Logger

	// Get library path
	libraryPath, err := ctx.App.Database.GetLibraryPathFromSettings()
	if err != nil || libraryPath == "" {
		return
	}

	// Get releasing anime in library
	releasingIDs, err := ctx.App.Database.GetReleasingAnimeIDsInLibrary()
	if err != nil || len(releasingIDs) == 0 {
		return
	}

	// Get cached media for these IDs
	cachedMedia, err := ctx.App.Database.GetCachedMediaByIDs(releasingIDs)
	if err != nil {
		return
	}

	// Build a set of releasing IDs for quick lookup
	releasingSet := make(map[int]bool, len(releasingIDs))
	for _, id := range releasingIDs {
		releasingSet[id] = true
	}

	// Fetch RSS feed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(subspleaseRSS)
	if err != nil {
		logger.Error().Err(err).Msg("cron/sp-sync: Failed to fetch RSS")
		return
	}

	// Build title→ID mapping from cached media (romaji + synonyms)
	type mediaRef struct {
		id     int
		cached *anilist.BaseAnime
	}
	titleMap := make(map[string]*mediaRef) // lowercase show name → media
	slugMap := make(map[string]*mediaRef)
	for _, id := range releasingIDs {
		cm, exists := cachedMedia[id]
		if !exists {
			continue
		}
		var media anilist.BaseAnime
		if err := json.Unmarshal(cm.Data, &media); err != nil {
			continue
		}
		ref := &mediaRef{id: id, cached: &media}

		if cm.SubspleaseSlug != "" {
			slugMap[strings.ToLower(cm.SubspleaseSlug)] = ref
		}
		// Add romaji title
		if cm.TitleRomaji != "" {
			titleMap[strings.ToLower(cm.TitleRomaji)] = ref
		}
		// Add synonyms
		for _, syn := range media.GetSynonymsDeref() {
			if isLatinStr(syn) {
				titleMap[strings.ToLower(syn)] = ref
			}
		}
	}

	// Try to start torrent client
	if !ctx.App.TorrentClientRepository.Start() {
		logger.Warn().Msg("cron/sp-sync: Could not start torrent client")
		return
	}

	totalDownloaded := 0

	// Process RSS items
	for _, item := range feed.Items {
		// Parse show name from title: [SubsPlease] ShowName - 08 (1080p) [CRC].mkv
		showName := extractShowName(item.Title)
		if showName == "" {
			continue
		}

		// Match against library
		ref, found := titleMap[strings.ToLower(showName)]
		if !found {
			ref, found = slugMap[strings.ToLower(subsplease.TitleToSlug(showName))]
		}
		if !found {
			continue
		}

		// Parse episode number
		epNum := extractEpNumber(item.Title)
		if epNum <= 0 {
			continue
		}

		// Check if already synced
		localFiles, _ := db_bridge.GetLocalFilesByMediaId(ctx.App.Database, ref.id)
		alreadySynced := false
		for _, lf := range localFiles {
			if lf.EpisodeNumber == epNum && strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
				alreadySynced = true
				break
			}
		}
		if alreadySynced {
			continue
		}

		// Delete old non-SubsPlease file for this episode
		for _, lf := range localFiles {
			if lf.EpisodeNumber == epNum && !strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
				_ = os.Remove(lf.LocalFilePath)
				_ = ctx.App.Database.DeleteGlobalMapping(lf.LocalFilePath)
			}
		}

		// Extract magnet and add to torrent client
		magnet := item.Link
		if magnet == "" || !strings.HasPrefix(magnet, "magnet:") {
			continue
		}

		err = ctx.App.TorrentClientRepository.AddMagnets([]string{magnet}, libraryPath)
		if err != nil {
			logger.Error().Err(err).Str("title", item.Title).Msg("cron/sp-sync: Failed to add magnet")
			continue
		}

		// Record download intent
		hash := extractHexHash(magnet)
		if hash != "" {
			_ = ctx.App.Database.UpsertPendingDownloadIntent(hash, ref.id, libraryPath)
		}

		// Update cached episode count
		_ = ctx.App.Database.SetSubsPleaseSid(ref.id, "found")

		totalDownloaded++
		logger.Info().Str("show", showName).Int("episode", epNum).Msg("cron/sp-sync: Downloaded")
	}

	if totalDownloaded > 0 {
		// Update episode counts for affected anime
		for _, id := range releasingIDs {
			localFiles, _ := db_bridge.GetLocalFilesByMediaId(ctx.App.Database, id)
			spCount := 0
			for _, lf := range localFiles {
				if strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
					spCount++
				}
			}
			if spCount > 0 {
				_ = ctx.App.Database.SetSubspleaseEpisodeCount(id, spCount)
			}
		}
		logger.Info().Int("total", totalDownloaded).Msg("cron/sp-sync: Sync complete")
	}
}

// extractShowName gets the show name from "[SubsPlease] ShowName - 08 (1080p) [CRC].mkv"
func extractShowName(title string) string {
	// Remove [SubsPlease] prefix
	title = strings.TrimPrefix(title, "[SubsPlease] ")
	// Find " - XX (" pattern
	idx := strings.LastIndex(title, " - ")
	if idx <= 0 {
		return ""
	}
	return strings.TrimSpace(title[:idx])
}

func extractEpNumber(title string) int {
	matches := spEpisodeRegex.FindStringSubmatch(title)
	if len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}
	return 0
}

func extractHexHash(magnet string) string {
	matches := spInfoHashRegex.FindStringSubmatch(magnet)
	if len(matches) < 2 {
		return ""
	}
	hash := matches[1]
	if len(hash) == 32 {
		decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(hash))
		if err == nil {
			return strings.ToLower(fmt.Sprintf("%x", decoded))
		}
	}
	return strings.ToLower(hash)
}

func isLatinStr(s string) bool {
	for _, r := range s {
		if r > 0x024F {
			return false
		}
	}
	return true
}
