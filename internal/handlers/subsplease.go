package handlers

import (
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/extension"
	hibiketorrent "seanime/internal/extension/hibike/torrent"
	"strings"

	"github.com/labstack/echo/v4"
)

type SubspleaseStatus struct {
	Available    bool                           `json:"available"`
	EpisodeCount int                           `json:"episodeCount"` // cached SP episode count
	LocalCount   int                           `json:"localCount"`   // local SubsPlease files
	ToSync       []*hibiketorrent.AnimeTorrent `json:"toSync"`       // episodes to download
}

// HandleGetSubspleaseEpisodes
//
//	@summary returns SubsPlease sync status and episodes that need syncing.
//	@desc Returns cached availability instantly, then includes episodes to sync if available.
//	@route /api/v1/subsplease/episodes [POST]
//	@returns handlers.SubspleaseStatus
func (h *Handler) HandleGetSubspleaseEpisodes(c echo.Context) error {

	type body struct {
		Media anilist.BaseAnime `json:"media"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Quick check from cache
	sid, cachedEpCount, _ := h.App.Database.GetSubspleaseInfo(b.Media.ID)

	// Count local SubsPlease files
	localFiles, _ := db_bridge.GetLocalFilesByMediaId(h.App.Database, b.Media.ID)
	localSpCount := 0
	syncedEps := make(map[int]bool)
	replaceFiles := make(map[int]string)
	for _, lf := range localFiles {
		if strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
			localSpCount++
			syncedEps[lf.EpisodeNumber] = true
		} else {
			replaceFiles[lf.EpisodeNumber] = lf.LocalFilePath
		}
	}

	// If cached and local count matches, we're in sync — no need to query SubsPlease
	if sid != "" && localSpCount >= cachedEpCount && cachedEpCount > 0 {
		return h.RespondWithData(c, SubspleaseStatus{
			Available:    true,
			EpisodeCount: cachedEpCount,
			LocalCount:   localSpCount,
			ToSync:       nil,
		})
	}

	// Need to fetch actual episodes from SubsPlease
	status := b.Media.GetStatus()
	format := b.Media.GetFormat()
	if status == nil || format == nil {
		return h.RespondWithData(c, SubspleaseStatus{Available: true, EpisodeCount: cachedEpCount, LocalCount: localSpCount})
	}

	providerExt, ok := extension.GetExtension[extension.AnimeTorrentProviderExtension](
		h.App.ExtensionRepository.GetExtensionBank(), "subsplease",
	)
	if !ok {
		return h.RespondWithData(c, SubspleaseStatus{Available: true, EpisodeCount: cachedEpCount, LocalCount: localSpCount})
	}

	queryMedia := hibiketorrent.Media{
		ID:           b.Media.GetID(),
		Status:       string(*status),
		Format:       string(*format),
		EnglishTitle: b.Media.GetTitle().GetEnglish(),
		RomajiTitle:  b.Media.GetRomajiTitleSafe(),
		EpisodeCount: b.Media.GetTotalEpisodeCount(),
		Synonyms:     b.Media.GetSynonymsDeref(),
	}

	torrents, err := providerExt.GetProvider().SmartSearch(hibiketorrent.AnimeSmartSearchOptions{
		Media:         queryMedia,
		EpisodeNumber: 0,
	})
	if err != nil || len(torrents) == 0 {
		return h.RespondWithData(c, SubspleaseStatus{Available: false})
	}

	// Cache that this anime is on SubsPlease
	_ = h.App.Database.SetSubsPleaseSid(b.Media.ID, "found")
	_ = h.App.Database.SetSubspleaseEpisodeCount(b.Media.ID, len(torrents))

	// Find episodes to sync
	var toSync []*hibiketorrent.AnimeTorrent
	for _, t := range torrents {
		if t.EpisodeNumber > 0 && !syncedEps[t.EpisodeNumber] {
			toSync = append(toSync, t)

			// Delete old non-SubsPlease file
			if oldPath, exists := replaceFiles[t.EpisodeNumber]; exists {
				_ = os.Remove(oldPath)
				_ = h.App.Database.DeleteGlobalMapping(oldPath)
			}
		}
	}

	return h.RespondWithData(c, SubspleaseStatus{
		Available:    true,
		EpisodeCount: len(torrents),
		LocalCount:   localSpCount,
		ToSync:       toSync,
	})
}
