package handlers

import (
	"errors"
	"seanime/internal/database/models"
	"strconv"

	"github.com/labstack/echo/v4"
)

// HandleGetUnmappedFiles
//
//	@summary returns all unmapped files
//	@desc This returns all files that could not be automatically matched to AniList entries.
//	@desc These files need manual mapping or can be marked as ignored.
//	@route /api/v1/global-mapping/unmapped-files [GET]
//	@returns []models.UnmappedFile
func (h *Handler) HandleGetUnmappedFiles(c echo.Context) error {

	unmappedFiles, err := h.App.Database.GetUnmappedFiles("UNMAPPED")
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, unmappedFiles)
}

// HandleGetIgnoredFiles
//
//	@summary returns all ignored files
//	@desc This returns all files that have been manually marked as ignored.
//	@route /api/v1/global-mapping/ignored-files [GET]
//	@returns []models.UnmappedFile
func (h *Handler) HandleGetIgnoredFiles(c echo.Context) error {

	ignoredFiles, err := h.App.Database.GetUnmappedFiles("IGNORED")
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, ignoredFiles)
}

// HandleGetGlobalMappings
//
//	@summary returns all global file mappings
//	@desc This returns all files that have been successfully mapped to AniList entries.
//	@route /api/v1/global-mapping/mappings [GET]
//	@returns []models.GlobalAnimeFileMapping
func (h *Handler) HandleGetGlobalMappings(c echo.Context) error {

	mappings, err := h.App.Database.GetAllGlobalMappings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, mappings)
}

type MapFileToAniListBody struct {
	FilePath      string `json:"filePath"`
	AniListID     int    `json:"anilistId"`
	Title         string `json:"title"`
	Year          int    `json:"year"`
	EpisodeNumber int    `json:"episodeNumber"`
}

// HandleMapFileToAniList
//
//	@summary maps a file to an AniList entry
//	@desc This manually maps a file to a specific AniList entry.
//	@desc The file will be moved from unmapped to mapped status.
//	@route /api/v1/global-mapping/map-file [POST]
//	@returns bool
func (h *Handler) HandleMapFileToAniList(c echo.Context) error {

	var body MapFileToAniListBody
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}
	userID := user.ID

	err := h.App.GlobalMappingService.MapFileToAniList(
		body.FilePath,
		body.AniListID,
		userID,
		body.Title,
		body.Year,
		body.EpisodeNumber,
	)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

type IgnoreFileBody struct {
	FilePath string `json:"filePath"`
}

// HandleIgnoreFile
//
//	@summary marks a file as ignored
//	@desc This marks an unmapped file as ignored so it won't appear in the unmapped files list.
//	@route /api/v1/global-mapping/ignore-file [POST]
//	@returns bool
func (h *Handler) HandleIgnoreFile(c echo.Context) error {

	var body IgnoreFileBody
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}
	userID := user.ID

	err := h.App.GlobalMappingService.IgnoreFile(body.FilePath, userID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleUnignoreFile
//
//	@summary unmarks a file as ignored
//	@desc This moves an ignored file back to the unmapped files list.
//	@route /api/v1/global-mapping/unignore-file [POST]
//	@returns bool
func (h *Handler) HandleUnignoreFile(c echo.Context) error {

	var body IgnoreFileBody
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Check if user is admin or if they ignored the file themselves
	if !user.IsAdmin() {
		var unmappedFile models.UnmappedFile
		err := h.App.Database.Gorm().Where("local_file_path = ?", body.FilePath).First(&unmappedFile).Error
		if err != nil {
			return h.RespondWithError(c, errors.New("file not found"))
		}
		if unmappedFile.IgnoredByUserID == 0 || unmappedFile.IgnoredByUserID != user.ID {
			return h.RespondWithError(c, errors.New("permission denied"))
		}
	}

	// Update the unmapped file status back to UNMAPPED
	err := h.App.Database.Gorm().Model(&models.UnmappedFile{}).
		Where("local_file_path = ?", body.FilePath).
		Updates(map[string]interface{}{
			"status":            "UNMAPPED",
			"ignored_by_user_id": nil,
			"ignored_at":        nil,
		}).Error

	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

type RemoveMappingBody struct {
	FilePath string `json:"filePath"`
}

// HandleRemoveMapping
//
//	@summary removes a global file mapping
//	@desc This removes a global file mapping and optionally moves the file back to unmapped.
//	@route /api/v1/global-mapping/remove-mapping [POST]
//	@returns bool
func (h *Handler) HandleRemoveMapping(c echo.Context) error {

	var body RemoveMappingBody
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the existing mapping
	mapping, err := h.App.Database.GetGlobalMapping(body.FilePath)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Delete the global mapping
	err = h.App.Database.DeleteGlobalMapping(body.FilePath)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Update cache - remove from global mapping service cache
	h.App.GlobalMappingService.RemoveFromCache(body.FilePath, mapping.AniListID)

	// Optionally, add back to unmapped files for re-processing
	// (This is commented out - you might want to enable it)
	/*
		unmappedFile := models.UnmappedFile{
			LocalFilePath: body.FilePath,
			DetectedTitle: mapping.Title,
			FileSize:      mapping.FileSize,
			LastDetected:  time.Now(),
			Status:        "UNMAPPED",
		}
		h.App.Database.CreateUnmappedFile(&unmappedFile)
	*/

	// Send notification to subscribed users that mapping was removed
	subscribedUsers := h.App.UserSubscriptionService.GetSubscribedUsers(mapping.AniListID)
	for range subscribedUsers {
		// Broadcast file unmapped event via SSE
		h.App.SSEManager.BroadcastEvent("file_unmapped", map[string]interface{}{
			"anilist_id": mapping.AniListID,
			"file_path":  body.FilePath,
			"title":      mapping.Title,
		})
	}

	return h.RespondWithData(c, true)
}

// HandleGetProgressSyncStats
//
//	@summary returns progress sync statistics
//	@desc This returns statistics about the progress sync queue (pending, synced, failed items).
//	@route /api/v1/global-mapping/progress-sync-stats [GET]
//	@returns map[string]int64
func (h *Handler) HandleGetProgressSyncStats(c echo.Context) error {

	stats, err := h.App.Database.GetProgressSyncStats()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, stats)
}

// HandleRetryFailedSyncItems
//
//	@summary retries failed progress sync items
//	@desc This resets all failed sync items back to pending status for retry.
//	@route /api/v1/global-mapping/retry-failed-sync [POST]
//	@returns bool
func (h *Handler) HandleRetryFailedSyncItems(c echo.Context) error {

	err := h.App.ProgressSyncService.RetryFailedItems()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

type GetFilesForAnimeBody struct {
	AniListID string `param:"anilistId"`
}

// HandleGetFilesForAnime
//
//	@summary returns all files mapped to a specific AniList ID
//	@desc This returns all files that are mapped to a specific anime.
//	@route /api/v1/global-mapping/files/:anilistId [GET]
//	@returns []string
func (h *Handler) HandleGetFilesForAnime(c echo.Context) error {

	aniListIDStr := c.Param("anilistId")
	aniListID, err := strconv.Atoi(aniListIDStr)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	files := h.App.GlobalMappingService.GetFilesForAniList(aniListID)

	return h.RespondWithData(c, files)
}

type GetUserSubscriptionsResponse struct {
	UserID        uint `json:"userId"`
	SubscribedIds []int `json:"subscribedIds"`
}

// HandleGetUserSubscriptions
//
//	@summary returns current user's anime subscriptions
//	@desc This returns all AniList IDs that the current user is subscribed to.
//	@route /api/v1/global-mapping/user-subscriptions [GET]
//	@returns GetUserSubscriptionsResponse
func (h *Handler) HandleGetUserSubscriptions(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}
	userID := user.ID

	subscriptions, err := h.App.Database.GetUserLibrarySubscriptions(userID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	subscribedIds := make([]int, len(subscriptions))
	for i, sub := range subscriptions {
		subscribedIds[i] = sub.AniListID
	}

	response := GetUserSubscriptionsResponse{
		UserID:        userID,
		SubscribedIds: subscribedIds,
	}

	return h.RespondWithData(c, response)
}

type SubscribeToAnimeBody struct {
	AniListID int `json:"anilistId"`
}

// HandleSubscribeToAnime
//
//	@summary subscribes user to an anime
//	@desc This manually subscribes the current user to receive notifications for a specific anime.
//	@route /api/v1/global-mapping/subscribe [POST]
//	@returns bool
func (h *Handler) HandleSubscribeToAnime(c echo.Context) error {

	var body SubscribeToAnimeBody
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}
	userID := user.ID

	err := h.App.UserSubscriptionService.SubscribeUserToAnime(userID, body.AniListID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleUnsubscribeFromAnime
//
//	@summary unsubscribes user from an anime
//	@desc This removes the current user's subscription to a specific anime.
//	@route /api/v1/global-mapping/unsubscribe [POST]
//	@returns bool
func (h *Handler) HandleUnsubscribeFromAnime(c echo.Context) error {

	var body SubscribeToAnimeBody
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}
	userID := user.ID

	err := h.App.UserSubscriptionService.UnsubscribeUserFromAnime(userID, body.AniListID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}