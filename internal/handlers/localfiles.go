package handlers

import (
	"errors"
	"os"
	"path/filepath"
	"seanime/internal/database/db_bridge"
	"seanime/internal/database/models"
	"seanime/internal/library/anime"
	"seanime/internal/library/filesystem"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sourcegraph/conc/pool"
)

// HandleGetLocalFiles
//
//	@summary returns all local files.
//	@desc Reminder that local files are scanned from the library path.
//	@route /api/v1/library/local-files [GET]
//	@returns []anime.LocalFile
func (h *Handler) HandleGetLocalFiles(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, lfs)
}

// HandleImportLocalFiles
//
//	@summary imports local files from the given path.
//	@desc This will import local files from the given path.
//	@desc The response is ignored, the client should refetch the entire library collection and media entry.
//	@route /api/v1/library/local-files/import [POST]
func (h *Handler) HandleImportLocalFiles(c echo.Context) error {
	return h.RespondWithError(c, errors.New("import is no longer supported"))
}

// HandleLocalFileBulkAction
//
//	@summary performs an action on all local files.
//	@desc This will perform the given action on all local files.
//	@desc The response is ignored, the client should refetch the entire library collection and media entry.
//	@route /api/v1/library/local-files [POST]
//	@returns []anime.LocalFile
func (h *Handler) HandleLocalFileBulkAction(c echo.Context) error {

	type body struct {
		Action string `json:"action"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, lfs)
}

// HandleUpdateLocalFileData
//
//	@summary updates the local file with the given path.
//	@desc This will update the local file with the given path.
//	@desc The response is ignored, the client should refetch the entire library collection and media entry.
//	@route /api/v1/library/local-file [PATCH]
//	@returns []anime.LocalFile
func (h *Handler) HandleUpdateLocalFileData(c echo.Context) error {

	type body struct {
		Path     string                   `json:"path"`
		Metadata *anime.LocalFileMetadata `json:"metadata"`
		Ignored  bool                     `json:"ignored"`
		MediaId  int                      `json:"mediaId"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Get existing mapping
	mapping, err := h.App.Database.GetGlobalMapping(b.Path)
	if err != nil {
		return h.RespondWithError(c, errors.New("local file not found"))
	}

	// Update mapping fields
	mapping.AniListID = b.MediaId
	mapping.Ignored = b.Ignored
	if b.Metadata != nil {
		mapping.EpisodeNumber = b.Metadata.Episode
		mapping.FileType = string(b.Metadata.Type)
	}

	if err := h.App.Database.UpdateGlobalMapping(mapping); err != nil {
		return h.RespondWithError(c, err)
	}

	// Return updated local files
	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, lfs)
}

// HandleUpdateLocalFiles
//
//	@summary updates local files with the given paths.
//	@desc The client should refetch the entire library collection and media entry.
//	@route /api/v1/library/local-files [PATCH]
//	@returns bool
func (h *Handler) HandleUpdateLocalFiles(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	type body struct {
		Paths   []string `json:"paths"`
		Action  string   `json:"action"`
		MediaId int      `json:"mediaId,omitempty"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	switch b.Action {
	case "lock", "unlock":
		// No-op
	case "ignore":
		for _, path := range b.Paths {
			_ = h.App.Database.SetGlobalMappingIgnored(path, true)
		}
	case "unignore":
		for _, path := range b.Paths {
			h.App.Database.Gorm().Model(&models.GlobalAnimeFileMapping{}).
				Where("local_file_path = ?", path).
				Updates(map[string]interface{}{"ignored": false})
		}
	case "unmatch":
		for _, path := range b.Paths {
			h.App.Database.Gorm().Model(&models.GlobalAnimeFileMapping{}).
				Where("local_file_path = ?", path).
				Updates(map[string]interface{}{"anilist_id": 0, "ignored": false})
		}
	case "match":
		for _, path := range b.Paths {
			_ = h.App.Database.Gorm().Model(&models.GlobalAnimeFileMapping{}).
				Where("local_file_path = ?", path).
				Updates(map[string]interface{}{"anilist_id": b.MediaId, "ignored": false})
		}
	}

	return h.RespondWithData(c, true)
}

// HandleDeleteLocalFiles
//
//	@summary deletes local files with the given paths.
//	@desc This will delete the local files with the given paths.
//	@desc The client should refetch the entire library collection and media entry.
//	@route /api/v1/library/local-files [DELETE]
//	@returns bool
func (h *Handler) HandleDeleteLocalFiles(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	type body struct {
		Paths []string `json:"paths"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	libraryPaths, err := h.App.Database.GetAllLibraryPathsFromSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}
	for _, p := range b.Paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return h.RespondWithError(c, errors.New("invalid path"))
		}
		inside := false
		for _, lp := range libraryPaths {
			if strings.HasPrefix(abs, lp+string(filepath.Separator)) || abs == lp {
				inside = true
				break
			}
		}
		if !inside {
			return h.RespondWithError(c, errors.New("path is outside library directories"))
		}
	}

	// Delete the files from disk
	p := pool.New().WithErrors()
	for _, path := range b.Paths {
		path := path
		p.Go(func() error {
			return os.Remove(path)
		})
	}
	if err := p.Wait(); err != nil {
		return h.RespondWithError(c, err)
	}

	// Remove from global mappings
	_ = h.App.Database.DeleteGlobalMappingsByPaths(b.Paths)

	return h.RespondWithData(c, true)
}

// HandleRemoveEmptyDirectories
//
//	@summary removes empty directories.
//	@desc This will remove empty directories in the library path.
//	@route /api/v1/library/empty-directories [DELETE]
//	@returns bool
func (h *Handler) HandleRemoveEmptyDirectories(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	libraryPaths, err := h.App.Database.GetAllLibraryPathsFromSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	for _, path := range libraryPaths {
		filesystem.RemoveEmptyDirectories(path, h.App.Logger)
	}

	return h.RespondWithData(c, true)
}

// HandleGetMediaAvailability
//
//	@summary returns available episodes for a specific media across the entire server.
//	@desc This endpoint returns all locally available episodes for a media from the server-wide file mapping.
//	@desc Useful for checking what episodes are available when a user adds anime to their library.
//	@route /api/v1/library/media-availability/{id} [GET]
//	@returns []anime.LocalFile
func (h *Handler) HandleGetMediaAvailability(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	mId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the user's LocalFiles from database
	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Filter files for this specific media
	var mediaFiles []*anime.LocalFile
	for _, lf := range lfs {
		if lf.MediaId == mId {
			mediaFiles = append(mediaFiles, lf)
		}
	}

	return h.RespondWithData(c, mediaFiles)
}
