package handlers

import (
	"errors"
	"seanime/internal/api/metadata"
	"seanime/internal/database/db_bridge"
	"seanime/internal/library/anime"
	"strconv"

	"github.com/labstack/echo/v4"
)

// HandleGetAnimeEpisodeCollection
//
//	@summary gets list of main episodes from local files
//	@desc This returns a list of main episodes for the given AniList anime media id from local library files.
//	@returns anime.EpisodeCollection
//	@param id - int - true - "AniList anime media ID"
//	@route /api/v1/anime/episode-collection/{id} [GET]
func (h *Handler) HandleGetAnimeEpisodeCollection(c echo.Context) error {
	mId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.AddOnRefreshAnilistCollectionFunc("HandleGetAnimeEpisodeCollection", func() {
		anime.ClearEpisodeCollectionCache()
	})

	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	// Get all the local files
	lfs, _, err := db_bridge.GetLocalFiles(h.App.Database)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the user's anilist collection
	animeCollection, err := h.App.GetAnimeCollectionForUser(user, false)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if animeCollection == nil {
		return h.RespondWithError(c, errors.New("anime collection not found"))
	}

	// Get user-specific platform
	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create entry using local files (leverages existing working logic)
	entry, err := anime.NewEntry(c.Request().Context(), &anime.NewEntryOptions{
		MediaId:          mId,
		LocalFiles:       lfs,
		AnimeCollection:  animeCollection,
		Platform:         userPlatform,
		MetadataProvider: h.App.MetadataProvider,
		IsSimulated:      false,
	})
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get metadata for the episode collection
	animeMetadata, _ := h.App.MetadataProvider.GetAnimeMetadata(metadata.AnilistPlatform, mId)

	// Create episode collection from entry episodes
	ec := &anime.EpisodeCollection{
		HasMappingError: false,
		Episodes:        entry.Episodes,
		Metadata:        animeMetadata,
	}

	h.App.FillerManager.HydrateEpisodeFillerData(mId, ec.Episodes)

	return h.RespondWithData(c, ec)
}
