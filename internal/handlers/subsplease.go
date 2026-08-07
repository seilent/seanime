package handlers

import (
	"fmt"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/extension"
	hibiketorrent "seanime/internal/extension/hibike/torrent"
	"seanime/internal/torrents/subsplease"
	"strings"

	"github.com/labstack/echo/v4"
)

type SubspleaseShow struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type SubspleaseStatus struct {
	Available    bool                          `json:"available"`
	EpisodeCount int                           `json:"episodeCount"`
	LocalCount   int                           `json:"localCount"`
	ToSync       []*hibiketorrent.AnimeTorrent `json:"toSync"`
	Slug         string                        `json:"slug"`
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

	sid, cachedEpCount, _ := h.App.Database.GetSubspleaseInfo(b.Media.ID)
	slug := h.App.Database.GetSubspleaseSlug(b.Media.ID)

	localFiles, _ := db_bridge.GetLocalFilesByMediaId(h.App.Database, b.Media.ID)
	localSpCount := 0
	syncedEps := make(map[int]bool)
	for _, lf := range localFiles {
		if strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
			localSpCount++
			syncedEps[lf.EpisodeNumber] = true
		}
	}

	if sid != "" && localSpCount >= cachedEpCount && cachedEpCount > 0 {
		return h.RespondWithData(c, SubspleaseStatus{
			Available:    true,
			EpisodeCount: cachedEpCount,
			LocalCount:   localSpCount,
			ToSync:       nil,
			Slug:         slug,
		})
	}

	status := b.Media.GetStatus()
	format := b.Media.GetFormat()
	if status == nil || format == nil {
		return h.RespondWithData(c, SubspleaseStatus{Available: true, EpisodeCount: cachedEpCount, LocalCount: localSpCount, Slug: slug})
	}

	providerExt, ok := extension.GetExtension[extension.AnimeTorrentProviderExtension](
		h.App.ExtensionRepository.GetExtensionBank(), "subsplease",
	)
	if !ok {
		return h.RespondWithData(c, SubspleaseStatus{Available: true, EpisodeCount: cachedEpCount, LocalCount: localSpCount, Slug: slug})
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

	torrents, err := fetchSubspleaseTorrents(providerExt.GetProvider(), queryMedia, slug)
	if err != nil || len(torrents) == 0 {
		return h.RespondWithData(c, SubspleaseStatus{Available: false, Slug: slug})
	}

	_ = h.App.Database.SetSubsPleaseSid(b.Media.ID, "found")
	_ = h.App.Database.SetSubspleaseEpisodeCount(b.Media.ID, len(torrents))

	var toSync []*hibiketorrent.AnimeTorrent
	for _, t := range torrents {
		if t.EpisodeNumber > 0 && !syncedEps[t.EpisodeNumber] {
			toSync = append(toSync, t)
		}
	}

	return h.RespondWithData(c, SubspleaseStatus{
		Available:    true,
		EpisodeCount: len(torrents),
		LocalCount:   localSpCount,
		ToSync:       toSync,
		Slug:         slug,
	})
}

func fetchSubspleaseTorrents(provider hibiketorrent.AnimeProvider, media hibiketorrent.Media, slug string) ([]*hibiketorrent.AnimeTorrent, error) {
	if slug != "" {
		return provider.Search(hibiketorrent.AnimeSearchOptions{Media: media, Query: slug})
	}
	return provider.SmartSearch(hibiketorrent.AnimeSmartSearchOptions{Media: media, EpisodeNumber: 0})
}

func parseSubspleaseSlug(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "/shows/"); i >= 0 {
		s = s[i+len("/shows/"):]
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	s = strings.Trim(s, "/")
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	return s
}

// HandleLinkSubsplease
//
//	@summary links an anime to a SubsPlease show by URL, bypassing title matching.
//	@desc Extracts the show slug from a SubsPlease URL, validates it resolves episodes, and persists it for sync.
//	@route /api/v1/subsplease/link [POST]
//	@returns handlers.SubspleaseStatus
func (h *Handler) HandleLinkSubsplease(c echo.Context) error {

	type body struct {
		MediaID int    `json:"mediaId"`
		Url     string `json:"url"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	slug := parseSubspleaseSlug(b.Url)
	if b.MediaID == 0 || slug == "" {
		return h.RespondWithError(c, fmt.Errorf("invalid SubsPlease URL"))
	}

	providerExt, ok := extension.GetExtension[extension.AnimeTorrentProviderExtension](
		h.App.ExtensionRepository.GetExtensionBank(), "subsplease",
	)
	if !ok {
		return h.RespondWithError(c, fmt.Errorf("SubsPlease provider not available"))
	}

	torrents, err := fetchSubspleaseTorrents(providerExt.GetProvider(), hibiketorrent.Media{ID: b.MediaID}, slug)
	if err != nil || len(torrents) == 0 {
		return h.RespondWithError(c, fmt.Errorf("no episodes found for SubsPlease show '%s'", slug))
	}

	if err := h.App.Database.SetSubspleaseSlug(b.MediaID, slug); err != nil {
		return h.RespondWithError(c, err)
	}
	_ = h.App.Database.SetSubsPleaseSid(b.MediaID, "found")
	_ = h.App.Database.SetSubspleaseEpisodeCount(b.MediaID, len(torrents))

	return h.RespondWithData(c, SubspleaseStatus{
		Available:    true,
		EpisodeCount: len(torrents),
		Slug:         slug,
	})
}

// HandleGetSubspleaseShows
//
//	@summary returns the full list of SubsPlease shows with display title and slug.
//	@desc Fetches the SubsPlease shows index and returns all available shows sorted alphabetically.
//	@route /api/v1/subsplease/shows [GET]
//	@returns []handlers.SubspleaseShow
func (h *Handler) HandleGetSubspleaseShows(c echo.Context) error {
	shows, err := subsplease.FetchShows()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	result := make([]SubspleaseShow, len(shows))
	for i, s := range shows {
		result[i] = SubspleaseShow{Title: s.Title, Slug: s.Slug}
	}

	return h.RespondWithData(c, result)
}
