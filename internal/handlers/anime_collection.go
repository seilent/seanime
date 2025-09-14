package handlers

import (
	"errors"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/library/anime"
	"seanime/internal/util"
	"seanime/internal/util/result"
	"time"

	"github.com/labstack/echo/v4"
)

// HandleGetLibraryCollection
//
//	@summary returns the main local anime collection.
//	@desc This creates a new LibraryCollection struct and returns it.
//	@desc This is used to get the main anime collection of the user.
//	@desc It uses the cached Anilist anime collection for the GET method.
//	@desc It refreshes the AniList anime collection if the POST method is used.
//	@route /api/v1/library/collection [GET,POST]
//	@returns anime.LibraryCollection
func (h *Handler) HandleGetLibraryCollection(c echo.Context) error {
	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	// Check if the user has an AniList account linked
	userToken := h.App.GetUserAnilistTokenForUser(user)
	if userToken == "" {
		// User doesn't have AniList linked, return empty library collection
		return h.RespondWithData(c, &anime.LibraryCollection{})
	}

	animeCollection, err := h.App.GetAnimeCollectionForUser(user, false)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Auto-populate user subscriptions for global media pool system
	if animeCollection != nil {
		err = h.trackUserSubscriptionsFromCollection(user.ID, animeCollection)
		if err != nil {
			// Log error but don't fail library request
			h.App.Logger.Error().Err(err).Msg("Failed to track user subscriptions from library collection")
		}
	}

	if animeCollection == nil {
		return h.RespondWithData(c, &anime.LibraryCollection{})
	}

	// Get user-specific local files
	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		// If no local files exist for this user, use empty slice
		lfs = []*anime.LocalFile{}
	}

	// Use existing GetUserPlatform method instead of global platform
	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	libraryCollection, err := anime.NewLibraryCollection(c.Request().Context(), &anime.NewLibraryCollectionOptions{
		AnimeCollection:  animeCollection,
		Platform:         userPlatform,
		LocalFiles:       lfs,
		MetadataProvider: h.App.MetadataProvider,
	})
	if err != nil {
		return h.RespondWithError(c, err)
	}



	// Hydrate total library size
	if libraryCollection != nil && libraryCollection.Stats != nil {
		libraryCollection.Stats.TotalSize = util.Bytes(h.App.TotalLibrarySize)
	}

	return h.RespondWithData(c, libraryCollection)
}

//----------------------------------------------------------------------------------------------------------------------------------------------------

var animeScheduleCache = result.NewCache[int, []*anime.ScheduleItem]()

// HandleGetAnimeCollectionSchedule
//
//	@summary returns anime collection schedule
//	@desc This is used by the "Schedule" page to display the anime schedule.
//	@route /api/v1/library/schedule [GET]
//	@returns []anime.ScheduleItem
func (h *Handler) HandleGetAnimeCollectionSchedule(c echo.Context) error {
	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	// Check if the user has an AniList account linked
	userToken := h.App.GetUserAnilistTokenForUser(user)
	if userToken == "" {
		// User doesn't have AniList linked, return empty schedule
		return h.RespondWithData(c, []*anime.ScheduleItem{})
	}

	// Invalidate the cache when the Anilist collection is refreshed
	h.App.AddOnRefreshAnilistCollectionFunc("HandleGetAnimeCollectionSchedule", func() {
		animeScheduleCache.Clear()
	})

	if ret, ok := animeScheduleCache.Get(1); ok {
		return h.RespondWithData(c, ret)
	}

	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}
	
	animeSchedule, err := userPlatform.GetAnimeAiringSchedule(c.Request().Context())
	if err != nil {
		return h.RespondWithError(c, err)
	}

	animeCollection, err := h.App.GetAnimeCollectionForUser(user, false)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	ret := anime.GetScheduleItems(animeSchedule, animeCollection)

	animeScheduleCache.SetT(1, ret, 1*time.Hour)

	return h.RespondWithData(c, ret)
}

// HandleAddUnknownMedia
//
//	@summary adds the given media to the user's AniList planning collections
//	@desc Since media not found in the user's AniList collection are not displayed in the library, this route is used to add them.
//	@desc The response is ignored in the frontend, the client should just refetch the entire library collection.
//	@route /api/v1/library/unknown-media [POST]
//	@returns anilist.AnimeCollection
func (h *Handler) HandleAddUnknownMedia(c echo.Context) error {

	type body struct {
		MediaIds []int `json:"mediaIds"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}
	
	// Add non-added media entries to AniList collection
	if err := userPlatform.AddMediaToCollection(c.Request().Context(), b.MediaIds); err != nil {
		return h.RespondWithError(c, errors.New("error: Anilist responded with an error, this is most likely a rate limit issue"))
	}

	// Bypass the cache
	animeCollection, err := h.App.GetAnimeCollectionForUser(user, true)
	if err != nil {
		return h.RespondWithError(c, errors.New("error: Anilist responded with an error, wait one minute before refreshing"))
	}

	return h.RespondWithData(c, animeCollection)

}

// trackUserSubscriptionsFromCollection records which anime a user has in their AniList collection
// This enables the global media pool system by populating subscription data when users view their library
func (h *Handler) trackUserSubscriptionsFromCollection(userID uint, animeCollection *anilist.AnimeCollection) error {
	if animeCollection == nil {
		return nil
	}

	// Extract anime media from collection - use BaseAnime since that's what AnimeCollection returns
	collectionMedia := make([]*anilist.BaseAnime, 0)
	for _, list := range animeCollection.GetMediaListCollection().GetLists() {
		for _, entry := range list.GetEntries() {
			collectionMedia = append(collectionMedia, entry.GetMedia())
		}
	}

	if len(collectionMedia) == 0 {
		return nil
	}

	h.App.Logger.Debug().
		Int("userID", int(userID)).
		Int("animeCount", len(collectionMedia)).
		Msg("Tracking user anime subscriptions from library collection")

	// Track subscriptions for each anime in user's collection
	for _, anime := range collectionMedia {
		err := h.App.Database.UpsertUserAnimeSubscription(userID, anime.GetID(), "active")
		if err != nil {
			// Log error but don't fail the entire process for a single subscription error
			h.App.Logger.Warn().
				Err(err).
				Int("aniListID", anime.GetID()).
				Str("title", anime.GetTitleSafe()).
				Msg("Failed to upsert user anime subscription")
			continue
		}
	}

	h.App.Logger.Debug().
		Int("userID", int(userID)).
		Int("animeCount", len(collectionMedia)).
		Msg("User anime subscriptions tracked successfully from library collection")

	return nil
}
