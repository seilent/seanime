package handlers

import (
	"errors"
	
	"github.com/labstack/echo/v4"
)

// HandlePopulateFillerData
//
//	@summary fetches and caches filler data for the given media.
//	@desc This will fetch and cache filler data for the given media.
//	@returns true
//	@route /api/v1/metadata-provider/filler [POST]
func (h *Handler) HandlePopulateFillerData(c echo.Context) error {
	type body struct {
		MediaId int `json:"mediaId"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	animeCollection, err := h.App.GetAnimeCollectionForUser(user, false)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	media, found := animeCollection.FindAnime(b.MediaId)
	if !found {
		// Fetch media
		userPlatform, err := h.GetUserPlatform(c)
		if err != nil {
			return h.RespondWithError(c, err)
		}
		media, err = userPlatform.GetAnime(c.Request().Context(), b.MediaId)
		if err != nil {
			return h.RespondWithError(c, err)
		}
	}

	// Fetch filler data
	err = h.App.FillerManager.FetchAndStoreFillerData(b.MediaId, media.GetAllTitlesDeref())
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleRemoveFillerData
//
//	@summary removes filler data cache.
//	@desc This will remove the filler data cache for the given media.
//	@returns bool
//	@route /api/v1/metadata-provider/filler [DELETE]
func (h *Handler) HandleRemoveFillerData(c echo.Context) error {
	type body struct {
		MediaId int `json:"mediaId"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	err := h.App.FillerManager.RemoveFillerData(b.MediaId)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}
