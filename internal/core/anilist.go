package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"seanime/internal/api/anilist"
	"seanime/internal/database/models"
	"seanime/internal/platforms/anilist_platform"
	"seanime/internal/platforms/platform"
	"seanime/internal/user"
)

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
	
	return platform.GetAnimeCollection(context.Background(), bypassCache)
}

// GetRawAnimeCollection is the same as GetAnimeCollection but returns the raw collection that includes custom lists
// Removed global GetRawAnimeCollection - use GetRawAnimeCollectionForUser() directly

// GetRawAnimeCollectionForUser returns the raw AniList anime collection for a specific user, ensuring proper isolation.
func (a *App) GetRawAnimeCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.AnimeCollection, error) {
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
	
	return platform.GetRawAnimeCollection(context.Background(), bypassCache)
}

// Removed global RefreshAnimeCollection - use RefreshAnimeCollectionForUser() directly

// RefreshAnimeCollectionForUser refreshes the AniList anime collection for a specific user, ensuring proper isolation.
func (a *App) RefreshAnimeCollectionForUser(dbUser *models.User) (*anilist.AnimeCollection, error) {
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
	
	ret, err := platform.RefreshAnimeCollection(context.Background())
	if err != nil {
		return nil, err
	}

	// Call refresh hooks for this user's collection
	go func() {
		for _, fn := range a.OnRefreshAnilistCollectionFuncs.Values() {
			fn()
		}
	}()

	return ret, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Global GetMangaCollection removed - use GetMangaCollectionForUser() directly

// GetMangaCollectionForUser returns the AniList manga collection for a specific user, ensuring proper isolation.
func (a *App) GetMangaCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.MangaCollection, error) {
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
	
	return platform.GetMangaCollection(context.Background(), bypassCache)
}

// Global GetRawMangaCollection removed - use GetRawMangaCollectionForUser() directly

// GetRawMangaCollectionForUser returns the raw AniList manga collection for a specific user, ensuring proper isolation.
func (a *App) GetRawMangaCollectionForUser(dbUser *models.User, bypassCache bool) (*anilist.MangaCollection, error) {
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
	
	return platform.GetRawMangaCollection(context.Background(), bypassCache)
}

// Global RefreshMangaCollection removed - use RefreshMangaCollectionForUser() directly

// RefreshMangaCollectionForUser refreshes the AniList manga collection for a specific user, ensuring proper isolation.
func (a *App) RefreshMangaCollectionForUser(dbUser *models.User) (*anilist.MangaCollection, error) {
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
	
	mc, err := platform.RefreshMangaCollection(context.Background())
	if err != nil {
		return nil, err
	}

	// Pure multiuser system - no global cache

	return mc, nil
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
	if token == "" || token == user.SimulatedUserToken {
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
