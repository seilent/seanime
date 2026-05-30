package handlers

import (
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/extension"
	hibiketorrent "seanime/internal/extension/hibike/torrent"
	"strings"

	"github.com/labstack/echo/v4"
)

// HandleGetSubspleaseEpisodes
//
//	@summary returns available SubsPlease episodes not yet downloaded locally.
//	@desc Checks SubsPlease for available episodes and returns those missing from local files.
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
	queryMedia := hibiketorrent.Media{
		ID:           b.Media.GetID(),
		Status:       string(*b.Media.GetStatus()),
		Format:       string(*b.Media.GetFormat()),
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

	// Get local files for this media to find which episodes need syncing
	localFiles, _ := db_bridge.GetLocalFilesByMediaId(h.App.Database, b.Media.ID)
	syncedEps := make(map[int]bool)
	for _, lf := range localFiles {
		// Only consider it synced if the file is from SubsPlease
		if strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
			syncedEps[lf.EpisodeNumber] = true
		}
	}

	// Return episodes not yet synced (missing OR from a different group)
	var toSync []*hibiketorrent.AnimeTorrent
	for _, t := range torrents {
		if t.EpisodeNumber > 0 && !syncedEps[t.EpisodeNumber] {
			toSync = append(toSync, t)
		}
	}

	return h.RespondWithData(c, toSync)
}
