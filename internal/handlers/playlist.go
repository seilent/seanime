package handlers

import (
	"context"
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"seanime/internal/database/db_bridge"
	"seanime/internal/library/anime"
	"seanime/internal/util"
)

// HandleCreatePlaylist
//
//	@summary creates a new playlist.
//	@desc This will create a new playlist with the given name and local file paths.
//	@desc The response is ignored, the client should re-fetch the playlists after this.
//	@route /api/v1/playlist [POST]
//	@returns anime.Playlist
func (h *Handler) HandleCreatePlaylist(c echo.Context) error {

	type body struct {
		Name  string   `json:"name"`
		Paths []string `json:"paths"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Get the user's local files
	dbLfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Filter the local files
	lfs := make([]*anime.LocalFile, 0)
	for _, path := range b.Paths {
		for _, lf := range dbLfs {
			if lf.GetNormalizedPath() == util.NormalizePath(path) {
				lfs = append(lfs, lf)
				break
			}
		}
	}

	// Create the playlist
	playlist := anime.NewPlaylist(b.Name)
	playlist.SetLocalFiles(lfs)

	// Save the playlist for this user
	if err := db_bridge.SavePlaylist(h.App.Database, user.ID, playlist); err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, playlist)
}

// HandleGetPlaylists
//
//	@summary returns all playlists for the current user.
//	@route /api/v1/playlists [GET]
//	@returns []anime.Playlist
func (h *Handler) HandleGetPlaylists(c echo.Context) error {

	// Get current user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	playlists, err := db_bridge.GetPlaylists(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, playlists)
}

// HandleUpdatePlaylist
//
//	@summary updates a playlist.
//	@returns the updated playlist
//	@desc The response is ignored, the client should re-fetch the playlists after this.
//	@route /api/v1/playlist [PATCH]
//	@param id - int - true - "The ID of the playlist to update."
//	@returns anime.Playlist
func (h *Handler) HandleUpdatePlaylist(c echo.Context) error {

	type body struct {
		DbId  uint     `json:"dbId"`
		Name  string   `json:"name"`
		Paths []string `json:"paths"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Get the user's local files
	dbLfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Filter the local files
	lfs := make([]*anime.LocalFile, 0)
	for _, path := range b.Paths {
		for _, lf := range dbLfs {
			if lf.GetNormalizedPath() == util.NormalizePath(path) {
				lfs = append(lfs, lf)
				break
			}
		}
	}

	// Recreate playlist
	playlist := anime.NewPlaylist(b.Name)
	playlist.DbId = b.DbId
	playlist.Name = b.Name
	playlist.SetLocalFiles(lfs)

	// Save the playlist (ensure it belongs to this user)
	if err := db_bridge.UpdatePlaylist(h.App.Database, user.ID, playlist); err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, playlist)
}

// HandleDeletePlaylist
//
//	@summary deletes a playlist.
//	@route /api/v1/playlist [DELETE]
//	@returns bool
func (h *Handler) HandleDeletePlaylist(c echo.Context) error {

	type body struct {
		DbId uint `json:"dbId"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)

	}

	// Get current user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Delete playlist (ensure it belongs to this user)
	if err := db_bridge.DeletePlaylist(h.App.Database, user.ID, b.DbId); err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleGetPlaylistEpisodes
//
//	@summary returns all the local files of a playlist media entry that have not been watched.
//	@route /api/v1/playlist/episodes/{id}/{progress} [GET]
//	@param id - int - true - "The ID of the media entry."
//	@param progress - int - true - "The progress of the media entry."
//	@returns []anime.LocalFile
func (h *Handler) HandleGetPlaylistEpisodes(c echo.Context) error {

	// Get current user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	mediaId, _ := strconv.Atoi(c.Param("id"))
	progress, _ := strconv.Atoi(c.Param("progress"))

	// Get the media collection for the user
	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	animeCollection, err := userPlatform.GetAnimeCollectionWithRelations(context.Background())
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Check if media exists (not strictly necessary for this endpoint)
	_, found := animeCollection.GetListEntryFromMediaId(mediaId)
	if !found {
		return h.RespondWithError(c, errors.New("media not found"))
	}

	// Group local files by media id
	groupedLfs := anime.GroupLocalFilesByMediaID(lfs)

	// Get the local files for the media
	var ret []*anime.LocalFile
	if _lfs, ok := groupedLfs[mediaId]; ok {
		ret = lo.Filter(_lfs, func(lf *anime.LocalFile, _ int) bool {
			ep := lf.GetEpisodeNumber()
			return ep > progress
		})
	}

	// If no local files are found, return empty slice
	if len(ret) == 0 {
		return h.RespondWithData(c, make([]*anime.LocalFile, 0))
	}

	return h.RespondWithData(c, ret)
}
