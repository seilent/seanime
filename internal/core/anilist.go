package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"seanime/internal/api/anilist"
	"seanime/internal/database/models"
	"seanime/internal/platforms/anilist_platform"
	"seanime/internal/platforms/platform"
)

// collectionCacheTTL is the stale-while-revalidate threshold for cached collections.
const collectionCacheTTL = 10 * time.Minute

// swrInflight deduplicates background revalidation goroutines keyed by "userID:variant".
var swrInflight sync.Map

// Global user methods removed - use GetUserPlatform() in handlers instead

// GetUserAnilistTokenForUser returns the AniList token for a specific database user.
// This is used in multi-user mode when we have user context from middleware.
func (a *App) GetUserAnilistTokenForUser(dbUser *models.User) string {
	if dbUser == nil {
		return ""
	}
	
	// Get the account (AniList connection) for this user
	account, err := a.Database.GetAccountForUser(dbUser.ID)
	if err != nil || account == nil {
		return ""
	}
	
	return account.Token
}

// SetUserFromContext removed - pure multiuser system uses GetUserPlatform() in handlers

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// UpdatePlatform changes the current platform to the provided one.
func (a *App) UpdatePlatform(platform platform.Platform) {
	a.AnilistPlatform = platform
}

// UpdateAnilistClientToken will update the Anilist Client Wrapper token.
// This function should be called when a user logs in
func (a *App) UpdateAnilistClientToken(token string) {
	a.AnilistClient = anilist.NewAnilistClient(token)
	a.AnilistPlatform.SetAnilistClient(a.AnilistClient) // Update Anilist Client Wrapper in Platform
}

// Global collection methods removed - use GetAnimeCollectionForUser() directly

// GetAnimeCollectionForUser returns the AniList anime collection for a specific user, ensuring proper isolation.
func (a *App) GetAnimeCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.AnimeCollection, error) {
	// bypassCache == true: sync refresh with stale fallback (same as Refresh semantics)
	if bypassCache {
		coll, err := a.fetchAnimeCollectionForUser(dbUser)
		if err != nil {
			// Stale fallback
			if cached, ok := a.reassembleAnimeCollection(dbUser.ID, "anime"); ok {
				return cached, nil
			}
			return nil, err
		}
		a.persistAnimeCollection(coll, dbUser.ID, "anime")
		return coll, nil
	}

	// SWR: try cache first
	if cached, ok := a.reassembleAnimeCollection(dbUser.ID, "anime"); ok {
		// Check staleness and trigger background revalidation if needed
		row, found, _ := a.Database.GetCachedUserMediaList(dbUser.ID, "anime")
		if found && time.Since(row.UpdatedAt) >= collectionCacheTTL {
			a.spawnRevalidation(dbUser, "anime", func() (*anilist.AnimeCollection, error) {
				return a.fetchAnimeCollectionForUser(dbUser)
			})
		}
		return cached, nil
	}

	// Cold miss: live fetch
	coll, err := a.fetchAnimeCollectionForUser(dbUser)
	if err != nil {
		return nil, err
	}
	a.persistAnimeCollection(coll, dbUser.ID, "anime")
	return coll, nil
}

// fetchAnimeCollectionForUser performs the live AniList fetch for anime collection.
func (a *App) fetchAnimeCollectionForUser(dbUser *models.User) (*anilist.AnimeCollection, error) {
	// Create user-specific AniList client
	token, err := a.getUserTokenForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	username, err := a.getUsernameForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	client := anilist.NewAnilistClient(token)
	platform := anilist_platform.NewAnilistPlatform(client, a.Logger)
	platform.SetUsername(username)

	return platform.GetAnimeCollection(context.Background(), false)
}

// GetRawAnimeCollection is the same as GetAnimeCollection but returns the raw collection that includes custom lists
// Removed global GetRawAnimeCollection - use GetRawAnimeCollectionForUser() directly

// GetRawAnimeCollectionForUser returns the raw AniList anime collection for a specific user, ensuring proper isolation.
func (a *App) GetRawAnimeCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.AnimeCollection, error) {
	if bypassCache {
		coll, err := a.fetchRawAnimeCollectionForUser(dbUser)
		if err != nil {
			if cached, ok := a.reassembleAnimeCollection(dbUser.ID, "anime_raw"); ok {
				return cached, nil
			}
			return nil, err
		}
		a.persistAnimeCollection(coll, dbUser.ID, "anime_raw")
		return coll, nil
	}

	// SWR: try cache first
	if cached, ok := a.reassembleAnimeCollection(dbUser.ID, "anime_raw"); ok {
		row, found, _ := a.Database.GetCachedUserMediaList(dbUser.ID, "anime_raw")
		if found && time.Since(row.UpdatedAt) >= collectionCacheTTL {
			a.spawnRevalidation(dbUser, "anime_raw", func() (*anilist.AnimeCollection, error) {
				return a.fetchRawAnimeCollectionForUser(dbUser)
			})
		}
		return cached, nil
	}

	// Cold miss
	coll, err := a.fetchRawAnimeCollectionForUser(dbUser)
	if err != nil {
		return nil, err
	}
	a.persistAnimeCollection(coll, dbUser.ID, "anime_raw")
	return coll, nil
}

// fetchRawAnimeCollectionForUser performs the live AniList fetch for raw anime collection.
func (a *App) fetchRawAnimeCollectionForUser(dbUser *models.User) (*anilist.AnimeCollection, error) {
	token, err := a.getUserTokenForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	username, err := a.getUsernameForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	client := anilist.NewAnilistClient(token)
	platform := anilist_platform.NewAnilistPlatform(client, a.Logger)
	platform.SetUsername(username)

	return platform.GetRawAnimeCollection(context.Background(), false)
}

// Removed global RefreshAnimeCollection - use RefreshAnimeCollectionForUser() directly

// RefreshAnimeCollectionForUser refreshes the AniList anime collection for a specific user, ensuring proper isolation.
func (a *App) RefreshAnimeCollectionForUser(dbUser *models.User) (*anilist.AnimeCollection, error) {
	coll, err := a.fetchAnimeCollectionForUser(dbUser)
	if err != nil {
		// On error, try returning stale cache
		if cached, ok := a.reassembleAnimeCollection(dbUser.ID, "anime"); ok {
			return cached, nil
		}
		return nil, err
	}

	// Persist fresh data
	a.persistAnimeCollection(coll, dbUser.ID, "anime")

	// Call refresh hooks for this user's collection
	go func() {
		for _, fn := range a.OnRefreshAnilistCollectionFuncs.Values() {
			fn()
		}
	}()

	return coll, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Global GetMangaCollection removed - use GetMangaCollectionForUser() directly

// GetMangaCollectionForUser returns the AniList manga collection for a specific user, ensuring proper isolation.
func (a *App) GetMangaCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.MangaCollection, error) {
	if bypassCache {
		coll, err := a.fetchMangaCollectionForUser(dbUser)
		if err != nil {
			if cached, ok := a.reassembleMangaCollection(dbUser.ID, "manga"); ok {
				return cached, nil
			}
			return nil, err
		}
		a.persistMangaCollection(coll, dbUser.ID, "manga")
		return coll, nil
	}

	// SWR: try cache first
	if cached, ok := a.reassembleMangaCollection(dbUser.ID, "manga"); ok {
		row, found, _ := a.Database.GetCachedUserMediaList(dbUser.ID, "manga")
		if found && time.Since(row.UpdatedAt) >= collectionCacheTTL {
			a.spawnMangaRevalidation(dbUser, "manga", func() (*anilist.MangaCollection, error) {
				return a.fetchMangaCollectionForUser(dbUser)
			})
		}
		return cached, nil
	}

	// Cold miss
	coll, err := a.fetchMangaCollectionForUser(dbUser)
	if err != nil {
		return nil, err
	}
	a.persistMangaCollection(coll, dbUser.ID, "manga")
	return coll, nil
}

// fetchMangaCollectionForUser performs the live AniList fetch for manga collection.
func (a *App) fetchMangaCollectionForUser(dbUser *models.User) (*anilist.MangaCollection, error) {
	token, err := a.getUserTokenForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	username, err := a.getUsernameForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	client := anilist.NewAnilistClient(token)
	platform := anilist_platform.NewAnilistPlatform(client, a.Logger)
	platform.SetUsername(username)

	return platform.GetMangaCollection(context.Background(), false)
}

// Global GetRawMangaCollection removed - use GetRawMangaCollectionForUser() directly

// GetRawMangaCollectionForUser returns the raw AniList manga collection for a specific user, ensuring proper isolation.
func (a *App) GetRawMangaCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.MangaCollection, error) {
	if bypassCache {
		coll, err := a.fetchRawMangaCollectionForUser(dbUser)
		if err != nil {
			if cached, ok := a.reassembleMangaCollection(dbUser.ID, "manga_raw"); ok {
				return cached, nil
			}
			return nil, err
		}
		a.persistMangaCollection(coll, dbUser.ID, "manga_raw")
		return coll, nil
	}

	// SWR: try cache first
	if cached, ok := a.reassembleMangaCollection(dbUser.ID, "manga_raw"); ok {
		row, found, _ := a.Database.GetCachedUserMediaList(dbUser.ID, "manga_raw")
		if found && time.Since(row.UpdatedAt) >= collectionCacheTTL {
			a.spawnMangaRevalidation(dbUser, "manga_raw", func() (*anilist.MangaCollection, error) {
				return a.fetchRawMangaCollectionForUser(dbUser)
			})
		}
		return cached, nil
	}

	// Cold miss
	coll, err := a.fetchRawMangaCollectionForUser(dbUser)
	if err != nil {
		return nil, err
	}
	a.persistMangaCollection(coll, dbUser.ID, "manga_raw")
	return coll, nil
}

// fetchRawMangaCollectionForUser performs the live AniList fetch for raw manga collection.
func (a *App) fetchRawMangaCollectionForUser(dbUser *models.User) (*anilist.MangaCollection, error) {
	token, err := a.getUserTokenForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token: %w", err)
	}

	username, err := a.getUsernameForDBUser(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	client := anilist.NewAnilistClient(token)
	platform := anilist_platform.NewAnilistPlatform(client, a.Logger)
	platform.SetUsername(username)

	return platform.GetRawMangaCollection(context.Background(), false)
}

// Global RefreshMangaCollection removed - use RefreshMangaCollectionForUser() directly

// RefreshMangaCollectionForUser refreshes the AniList manga collection for a specific user, ensuring proper isolation.
func (a *App) RefreshMangaCollectionForUser(dbUser *models.User) (*anilist.MangaCollection, error) {
	coll, err := a.fetchMangaCollectionForUser(dbUser)
	if err != nil {
		// On error, try returning stale cache
		if cached, ok := a.reassembleMangaCollection(dbUser.ID, "manga"); ok {
			return cached, nil
		}
		return nil, err
	}

	// Persist fresh data
	a.persistMangaCollection(coll, dbUser.ID, "manga")

	// Pure multiuser system - no global cache

	return coll, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SWR Cache Helpers — Anime
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// persistAnimeCollection clones the collection, extracts media into CachedMedia rows,
// strips media from the clone to an id-only stub, and stores the skeleton. Errors are logged only.
func (a *App) persistAnimeCollection(coll *anilist.AnimeCollection, userID uint, variant string) {
	if coll == nil || coll.MediaListCollection == nil {
		return
	}

	// Clone via JSON round-trip to avoid mutating the caller's object
	raw, err := json.Marshal(coll)
	if err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to marshal anime collection for persist")
		return
	}
	var clone anilist.AnimeCollection
	if err := json.Unmarshal(raw, &clone); err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to unmarshal anime collection clone")
		return
	}

	var mediaItems []*models.CachedMedia
	now := time.Now()

	// Walk clone: extract media, build CachedMedia, then strip to id-only stub
	for _, list := range clone.MediaListCollection.Lists {
		if list == nil {
			continue
		}
		for _, entry := range list.Entries {
			if entry == nil || entry.Media == nil {
				continue
			}
			media := entry.Media
			mediaJSON, err := json.Marshal(media)
			if err != nil {
				continue
			}

			// Extract scalar fields best-effort
			cm := &models.CachedMedia{
				AnilistID: media.ID,
				Type:      "anime",
				Data:      mediaJSON,
				UpdatedAt: now,
			}
			if media.Format != nil {
				cm.Format = string(*media.Format)
			}
			if media.Status != nil {
				cm.Status = string(*media.Status)
			}
			if media.Season != nil {
				cm.Season = string(*media.Season)
			}
			if media.SeasonYear != nil {
				cm.SeasonYear = *media.SeasonYear
			}
			if media.Title != nil && media.Title.Romaji != nil {
				cm.TitleRomaji = *media.Title.Romaji
			}
			mediaItems = append(mediaItems, cm)

			// Strip media to id-only stub in the clone
			entry.Media = &anilist.BaseAnime{ID: media.ID}
		}
	}

	// Batch upsert media
	if err := a.Database.UpsertCachedMediaBatch(mediaItems); err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to upsert cached anime media")
	}

	// Marshal stripped skeleton and store
	skeletonJSON, err := json.Marshal(&clone)
	if err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to marshal anime skeleton")
		return
	}
	if err := a.Database.UpsertCachedUserMediaList(userID, variant, skeletonJSON); err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to upsert anime collection skeleton")
	}
}

// reassembleAnimeCollection loads the skeleton + media from cache and rehydrates.
// Returns (nil, false) on any miss or error.
func (a *App) reassembleAnimeCollection(userID uint, variant string) (*anilist.AnimeCollection, bool) {
	row, found, err := a.Database.GetCachedUserMediaList(userID, variant)
	if err != nil || !found {
		return nil, false
	}

	var coll anilist.AnimeCollection
	if err := json.Unmarshal(row.Data, &coll); err != nil {
		return nil, false
	}
	if coll.MediaListCollection == nil {
		return &coll, true
	}

	// Collect all media IDs from entries
	var ids []int
	for _, list := range coll.MediaListCollection.Lists {
		if list == nil {
			continue
		}
		for _, entry := range list.Entries {
			if entry != nil && entry.Media != nil {
				ids = append(ids, entry.Media.ID)
			}
		}
	}

	if len(ids) == 0 {
		return &coll, true
	}

	// Fetch media from cache
	mediaMap, err := a.Database.GetCachedMediaByIDs(ids)
	if err != nil {
		return nil, false
	}

	// Rehydrate: unmarshal full media into each entry
	for _, list := range coll.MediaListCollection.Lists {
		if list == nil {
			continue
		}
		for _, entry := range list.Entries {
			if entry == nil || entry.Media == nil {
				continue
			}
			cm, ok := mediaMap[entry.Media.ID]
			if !ok {
				// Partial cache miss — treat as full miss
				return nil, false
			}
			var fullMedia anilist.BaseAnime
			if err := json.Unmarshal(cm.Data, &fullMedia); err != nil {
				return nil, false
			}
			entry.Media = &fullMedia
		}
	}

	return &coll, true
}

// spawnRevalidation launches a deduped background goroutine to refresh an anime collection variant.
func (a *App) spawnRevalidation(dbUser *models.User, variant string, fetch func() (*anilist.AnimeCollection, error)) {
	key := fmt.Sprintf("%d:%s", dbUser.ID, variant)
	if _, loaded := swrInflight.LoadOrStore(key, struct{}{}); loaded {
		return // already in-flight
	}
	go func() {
		defer swrInflight.Delete(key)
		coll, err := fetch()
		if err != nil {
			a.Logger.Warn().Err(err).Str("variant", variant).Msg("core: [SWR] background revalidation failed")
			return
		}
		a.persistAnimeCollection(coll, dbUser.ID, variant)
	}()
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SWR Cache Helpers — Manga
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// persistMangaCollection clones the collection, extracts media into CachedMedia rows,
// strips media from the clone to an id-only stub, and stores the skeleton. Errors are logged only.
func (a *App) persistMangaCollection(coll *anilist.MangaCollection, userID uint, variant string) {
	if coll == nil || coll.MediaListCollection == nil {
		return
	}

	raw, err := json.Marshal(coll)
	if err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to marshal manga collection for persist")
		return
	}
	var clone anilist.MangaCollection
	if err := json.Unmarshal(raw, &clone); err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to unmarshal manga collection clone")
		return
	}

	var mediaItems []*models.CachedMedia
	now := time.Now()

	for _, list := range clone.MediaListCollection.Lists {
		if list == nil {
			continue
		}
		for _, entry := range list.Entries {
			if entry == nil || entry.Media == nil {
				continue
			}
			media := entry.Media
			mediaJSON, err := json.Marshal(media)
			if err != nil {
				continue
			}

			cm := &models.CachedMedia{
				AnilistID: media.ID,
				Type:      "manga",
				Data:      mediaJSON,
				UpdatedAt: now,
			}
			if media.Format != nil {
				cm.Format = string(*media.Format)
			}
			if media.Status != nil {
				cm.Status = string(*media.Status)
			}
			if media.Season != nil {
				cm.Season = string(*media.Season)
			}
			if media.Title != nil && media.Title.Romaji != nil {
				cm.TitleRomaji = *media.Title.Romaji
			}
			mediaItems = append(mediaItems, cm)

			// Strip media to id-only stub
			entry.Media = &anilist.BaseManga{ID: media.ID}
		}
	}

	if err := a.Database.UpsertCachedMediaBatch(mediaItems); err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to upsert cached manga media")
	}

	skeletonJSON, err := json.Marshal(&clone)
	if err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to marshal manga skeleton")
		return
	}
	if err := a.Database.UpsertCachedUserMediaList(userID, variant, skeletonJSON); err != nil {
		a.Logger.Error().Err(err).Msg("core: [SWR] failed to upsert manga collection skeleton")
	}
}

// reassembleMangaCollection loads the skeleton + media from cache and rehydrates.
func (a *App) reassembleMangaCollection(userID uint, variant string) (*anilist.MangaCollection, bool) {
	row, found, err := a.Database.GetCachedUserMediaList(userID, variant)
	if err != nil || !found {
		return nil, false
	}

	var coll anilist.MangaCollection
	if err := json.Unmarshal(row.Data, &coll); err != nil {
		return nil, false
	}
	if coll.MediaListCollection == nil {
		return &coll, true
	}

	var ids []int
	for _, list := range coll.MediaListCollection.Lists {
		if list == nil {
			continue
		}
		for _, entry := range list.Entries {
			if entry != nil && entry.Media != nil {
				ids = append(ids, entry.Media.ID)
			}
		}
	}

	if len(ids) == 0 {
		return &coll, true
	}

	mediaMap, err := a.Database.GetCachedMediaByIDs(ids)
	if err != nil {
		return nil, false
	}

	for _, list := range coll.MediaListCollection.Lists {
		if list == nil {
			continue
		}
		for _, entry := range list.Entries {
			if entry == nil || entry.Media == nil {
				continue
			}
			cm, ok := mediaMap[entry.Media.ID]
			if !ok {
				return nil, false
			}
			var fullMedia anilist.BaseManga
			if err := json.Unmarshal(cm.Data, &fullMedia); err != nil {
				return nil, false
			}
			entry.Media = &fullMedia
		}
	}

	return &coll, true
}

// spawnMangaRevalidation launches a deduped background goroutine to refresh a manga collection variant.
func (a *App) spawnMangaRevalidation(dbUser *models.User, variant string, fetch func() (*anilist.MangaCollection, error)) {
	key := fmt.Sprintf("%d:%s", dbUser.ID, variant)
	if _, loaded := swrInflight.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	go func() {
		defer swrInflight.Delete(key)
		coll, err := fetch()
		if err != nil {
			a.Logger.Warn().Err(err).Str("variant", variant).Msg("core: [SWR] background manga revalidation failed")
			return
		}
		a.persistMangaCollection(coll, dbUser.ID, variant)
	}()
}

// getUserTokenForDBUser gets the AniList token for a database user.
// This is a pure multiuser system - dbUser must not be nil.
func (a *App) getUserTokenForDBUser(dbUser *models.User) (string, error) {
	if dbUser == nil {
		return "", fmt.Errorf("dbUser cannot be nil in pure multiuser system")
	}

	// Get the account (AniList connection) for this user
	account, err := a.Database.GetAccountForUser(dbUser.ID)
	if err != nil {
		return "", fmt.Errorf("user has no AniList connection: %w", err)
	}

	if account.Token == "" {
		return "", fmt.Errorf("user has empty AniList token")
	}

	return account.Token, nil
}

// getUsernameForDBUser gets the AniList username for a database user.
// This is a pure multiuser system - dbUser must not be nil.
func (a *App) getUsernameForDBUser(dbUser *models.User) (string, error) {
	if dbUser == nil {
		return "", fmt.Errorf("dbUser cannot be nil in pure multiuser system")
	}

	// Get the account (AniList connection) for this user
	account, err := a.Database.GetAccountForUser(dbUser.ID)
	if err != nil {
		return "", fmt.Errorf("user has no AniList connection: %w", err)
	}

	// Parse the viewer JSON to get the username
	type ViewerData struct {
		Name string `json:"name"`
	}

	var viewer ViewerData
	if err := json.Unmarshal(account.Viewer, &viewer); err != nil {
		return "", fmt.Errorf("failed to parse viewer data: %w", err)
	}

	if viewer.Name == "" {
		return "", fmt.Errorf("user has no AniList username")
	}

	return viewer.Name, nil
}

// getUserIDFromToken extracts user ID from JWT token for comparison purposes.
// This is a simple implementation that decodes the JWT payload to get the "sub" field.
func (a *App) getUserIDFromToken(token string) string {
	if token == "" {
		return ""
	}

	// Simple JWT payload extraction (just for user ID comparison)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	// Parse JSON to get the "sub" field (user ID)
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	if sub, ok := claims["sub"].(string); ok {
		return sub
	}

	return ""
}
