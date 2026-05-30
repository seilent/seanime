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

// HandleGetSubspleaseEpisodes
//
//	@summary returns SubsPlease episodes that need syncing (missing or from different group).
//	@desc Checks SubsPlease for available episodes, compares against local files.
//	@desc Returns episodes that are either missing or not from SubsPlease.
//	@route /api/v1/subsplease/episodes [POST]
//	@returns []hibiketorrent.AnimeTorrent
func (h *Handler) HandleGetSubspleaseEpisodes(c echo.Context) error {

	type body struct {
		Media anilist.BaseAnime `json:"media"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get SubsPlease provider
	providerExt, ok := extension.GetExtension[extension.AnimeTorrentProviderExtension](
		h.App.ExtensionRepository.GetExtensionBank(), "subsplease",
	)
	if !ok {
		return h.RespondWithData(c, []*hibiketorrent.AnimeTorrent{})
	}

	// Build media for smart search
	status := b.Media.GetStatus()
	format := b.Media.GetFormat()
	if status == nil || format == nil {
		return h.RespondWithData(c, []*hibiketorrent.AnimeTorrent{})
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

	// Fetch all episodes from SubsPlease
	torrents, err := providerExt.GetProvider().SmartSearch(hibiketorrent.AnimeSmartSearchOptions{
		Media:         queryMedia,
		EpisodeNumber: 0, // all episodes
	})
	if err != nil || len(torrents) == 0 {
		return h.RespondWithData(c, []*hibiketorrent.AnimeTorrent{})
	}

	// Get local files for this media
	localFiles, _ := db_bridge.GetLocalFilesByMediaId(h.App.Database, b.Media.ID)

	// Build maps: which episodes are synced (SubsPlease) vs need replacement
	syncedEps := make(map[int]bool)
	replaceFiles := make(map[int]string) // episodeNumber -> filePath to delete
	for _, lf := range localFiles {
		if strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
			syncedEps[lf.EpisodeNumber] = true
		} else {
			// Non-SubsPlease file that should be replaced
			replaceFiles[lf.EpisodeNumber] = lf.LocalFilePath
		}
	}

	// Filter to episodes that need syncing
	var toSync []*hibiketorrent.AnimeTorrent
	for _, t := range torrents {
		if t.EpisodeNumber > 0 && !syncedEps[t.EpisodeNumber] {
			toSync = append(toSync, t)

			// Delete the old non-SubsPlease file and its mapping
			if oldPath, exists := replaceFiles[t.EpisodeNumber]; exists {
				_ = os.Remove(oldPath)
				_ = h.App.Database.DeleteGlobalMapping(oldPath)
				h.App.Logger.Debug().Str("path", oldPath).Int("episode", t.EpisodeNumber).
					Msg("subsplease sync: deleted old file for replacement")
			}
		}
	}

	return h.RespondWithData(c, toSync)
}
