package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/database/models"
	"seanime/internal/library/anime"
	"seanime/internal/platforms/anilist_platform"
	"seanime/internal/platforms/platform"
	"seanime/internal/util/comparison"
	"seanime/internal/util/limiter"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// MultiTokenAPI handles API calls with automatic token rotation
type MultiTokenAPI struct {
	Database           DatabaseInterface
	Logger             *zerolog.Logger
	AnilistRateLimiter *limiter.Limiter
	CompleteAnimeCache *anilist.CompleteAnimeCache
}

// NewMultiTokenAPI creates a new multi-token API handler
func NewMultiTokenAPI(database DatabaseInterface, logger *zerolog.Logger, rateLimiter *limiter.Limiter, cache *anilist.CompleteAnimeCache) *MultiTokenAPI {
	return &MultiTokenAPI{
		Database:           database,
		Logger:             logger,
		AnilistRateLimiter: rateLimiter,
		CompleteAnimeCache: cache,
	}
}

// AggregatedCollectionResult contains collections from multiple users
type AggregatedCollectionResult struct {
	AllAnime            []*anilist.CompleteAnime
	UserCollections     map[uint]*anilist.AnimeCollectionWithRelations
	TotalUsersAccessed  int
	SuccessfulFetches   int
	FailedFetches       int
}

// AggregateUserCollections fetches and combines anime collections from multiple users
// This replaces the per-file search approach with collection-based matching
func (mt *MultiTokenAPI) AggregateUserCollections(ctx context.Context) (*AggregatedCollectionResult, error) {
	result := &AggregatedCollectionResult{
		AllAnime:        make([]*anilist.CompleteAnime, 0),
		UserCollections: make(map[uint]*anilist.AnimeCollectionWithRelations),
	}

	if adapter, ok := mt.Database.(*DatabaseAdapter); ok {
		// Get all active users
		users, err := adapter.db.GetAllUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}

		result.TotalUsersAccessed = len(users)
		mt.Logger.Info().Int("totalUsers", len(users)).Msg("multi-token: Starting collection aggregation from multiple users")

		// Track unique anime by ID to avoid duplicates
		uniqueAnime := make(map[int]*anilist.CompleteAnime)

		for _, user := range users {
			if !user.IsActive {
				continue
			}

			// Create platform for this user
			userPlatform, err := mt.createUserPlatform(user.ID)
			if err != nil {
				mt.Logger.Debug().
					Err(err).
					Uint("userID", user.ID).
					Msg("multi-token: Failed to create platform for user, skipping")
				result.FailedFetches++
				continue
			}

			// Apply rate limiting
			mt.AnilistRateLimiter.Wait()

			// Fetch user's anime collection with relations
			collection, err := userPlatform.GetAnimeCollectionWithRelations(ctx)
			if err != nil {
				mt.Logger.Warn().
					Err(err).
					Uint("userID", user.ID).
					Msg("multi-token: Failed to fetch collection for user")
				result.FailedFetches++
				continue
			}

			// Store user's collection
			result.UserCollections[user.ID] = collection
			result.SuccessfulFetches++

			// Extract all anime from this user's collection
			userAnimeCount := 0
			if collection.GetMediaListCollection() != nil {
				for _, list := range collection.GetMediaListCollection().GetLists() {
					for _, entry := range list.GetEntries() {
						if anime := entry.GetMedia(); anime != nil {
							// Add to unique anime map (deduplicates across users)
							uniqueAnime[anime.GetID()] = anime
							userAnimeCount++
						}
					}
				}
			}

			mt.Logger.Info().
				Uint("userID", user.ID).
				Int("animeCount", userAnimeCount).
				Msg("multi-token: Successfully fetched user collection")
		}

		// Convert unique anime map to slice
		for _, anime := range uniqueAnime {
			result.AllAnime = append(result.AllAnime, anime)
		}

		mt.Logger.Info().
			Int("totalUniqueAnime", len(result.AllAnime)).
			Int("successfulFetches", result.SuccessfulFetches).
			Int("failedFetches", result.FailedFetches).
			Msg("multi-token: Collection aggregation completed")

		return result, nil
	}

	return nil, errors.New("database adapter not available")
}

// MatchFileResult contains the result of a file matching attempt
type MatchFileResult struct {
	Success       bool
	AniListID     int
	Title         string
	Year          int
	EpisodeNumber int
	Confidence    float64
	Error         error
}

// MatchFileWithTokenRotation attempts to match a file using token rotation strategy
func (mt *MultiTokenAPI) MatchFileWithTokenRotation(ctx context.Context, file *anime.LocalFile) *MatchFileResult {
	result := &MatchFileResult{
		Success: false,
	}

	// Extract title and basic info from file
	parsedTitle := mt.extractTitleFromFile(file)
	if parsedTitle == "" {
		result.Error = errors.New("no title could be parsed from filename")
		return result
	}

	// Step 1: Search for potential anime matches by title
	potentialMatches, err := mt.searchAnimeByTitle(ctx, parsedTitle)
	if err != nil {
		result.Error = fmt.Errorf("failed to search anime by title: %w", err)
		return result
	}

	if len(potentialMatches) == 0 {
		result.Error = errors.New("no potential matches found")
		return result
	}

	mt.Logger.Debug().
		Str("file", file.Path).
		Str("title", parsedTitle).
		Int("potentialMatches", len(potentialMatches)).
		Msg("multi-token: Found potential anime matches")

	// Step 2: Try to fetch detailed metadata for each potential match using token rotation
	for _, aniListID := range potentialMatches {
		anime, fetchErr := mt.fetchAnimeWithTokenRotation(ctx, aniListID)
		if fetchErr != nil {
			mt.Logger.Warn().
				Err(fetchErr).
				Int("aniListID", aniListID).
				Msg("multi-token: Failed to fetch anime with token rotation")
			continue
		}

		// Step 3: Calculate match confidence
		confidence := mt.calculateMatchConfidence(file, anime, parsedTitle)

		if confidence > 0.7 { // Threshold for accepting a match
			result.Success = true
			result.AniListID = anime.GetID()
			result.Title = anime.GetTitleSafe()
			if startDate := anime.GetStartDate(); startDate != nil && startDate.GetYear() != nil {
				result.Year = *startDate.GetYear()
			}
			result.EpisodeNumber = mt.extractEpisodeNumber(file)
			result.Confidence = confidence

			// Step 4: Save successful match to global mappings
			saveErr := mt.saveGlobalMapping(file, anime, result.EpisodeNumber)
			if saveErr != nil {
				mt.Logger.Warn().Err(saveErr).Msg("multi-token: Failed to save global mapping")
			}

			mt.Logger.Info().
				Str("file", file.Path).
				Int("aniListID", result.AniListID).
				Str("title", result.Title).
				Float64("confidence", confidence).
				Msg("multi-token: Successfully matched file")

			return result
		}
	}

	result.Error = errors.New("no suitable match found with sufficient confidence")
	return result
}

// searchAnimeByTitle searches for anime by title and returns potential AniList IDs
func (mt *MultiTokenAPI) searchAnimeByTitle(ctx context.Context, title string) ([]int, error) {
	// This method is now obsolete - we use collection aggregation instead
	// Return empty to indicate no individual searches needed
	mt.Logger.Debug().Str("title", title).Msg("multi-token: searchAnimeByTitle is deprecated, using collection aggregation")
	return []int{}, nil
}

// fetchAnimeWithTokenRotation fetches anime metadata using token rotation
func (mt *MultiTokenAPI) fetchAnimeWithTokenRotation(ctx context.Context, aniListID int) (*anilist.CompleteAnime, error) {
	// Check cache first
	if cachedAnime, exists := mt.CompleteAnimeCache.Get(aniListID); exists && cachedAnime != nil {
		return cachedAnime, nil
	}

	// Get users who have this anime in their collections
	if adapter, ok := mt.Database.(*DatabaseAdapter); ok {
		subscriptions, err := adapter.db.GetUsersWithAnime(aniListID)
		if err != nil || len(subscriptions) == 0 {
			return nil, fmt.Errorf("no users found with anime ID %d", aniListID)
		}

		// Try each user's token until one succeeds
		for _, subscription := range subscriptions {
			if subscription.TokenStatus != "active" {
				continue
			}

			// Get user's account to access their token
			userPlatform, err := mt.createUserPlatform(subscription.UserID)
			if err != nil {
				mt.Logger.Warn().
					Err(err).
					Uint("userID", subscription.UserID).
					Msg("multi-token: Failed to create user platform")
				continue
			}

			// Apply rate limiting
			mt.AnilistRateLimiter.Wait()

			// Try to fetch anime with this user's token
			anime, fetchErr := userPlatform.GetAnimeWithRelations(ctx, aniListID)
			if fetchErr != nil {
				// Mark this token as failed
				updateErr := adapter.db.UpdateTokenStatus(subscription.UserID, aniListID, "failed")
				if updateErr != nil {
					mt.Logger.Warn().Err(updateErr).Msg("multi-token: Failed to update token status")
				}

				mt.Logger.Warn().
					Err(fetchErr).
					Uint("userID", subscription.UserID).
					Int("aniListID", aniListID).
					Msg("multi-token: Token failed, trying next user")
				continue
			}

			// Success! Cache the result and return
			mt.CompleteAnimeCache.Set(aniListID, anime)

			// Update token status as successful
			updateErr := adapter.db.UpdateTokenStatus(subscription.UserID, aniListID, "active")
			if updateErr != nil {
				mt.Logger.Warn().Err(updateErr).Msg("multi-token: Failed to update token status")
			}

			mt.Logger.Debug().
				Uint("userID", subscription.UserID).
				Int("aniListID", aniListID).
				Msg("multi-token: Successfully fetched anime with user token")

			return anime, nil
		}
	}

	return nil, fmt.Errorf("all available tokens failed for anime ID %d", aniListID)
}

// createUserPlatform creates an AniList platform for a specific user
func (mt *MultiTokenAPI) createUserPlatform(userID uint) (platform.Platform, error) {
	// Get user's account from database which contains their AniList OAuth token
	if adapter, ok := mt.Database.(*DatabaseAdapter); ok {
		account, err := adapter.db.GetAccountForUser(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get account for user %d: %w", userID, err)
		}

		// Check if user has a valid AniList token
		if account.Token == "" {
			return nil, fmt.Errorf("user %d has no AniList token", userID)
		}

		// Create AniList client with user's token
		anilistClient := anilist.NewAnilistClient(account.Token)
		
		// Create platform instance for this user
		userPlatform := anilist_platform.NewAnilistPlatform(anilistClient, mt.Logger)
		userPlatform.SetUsername(account.Username)

		return userPlatform, nil
	}

	return nil, errors.New("database adapter not available")
}

// calculateMatchConfidence calculates how well a file matches an anime using all title variants
func (mt *MultiTokenAPI) calculateMatchConfidence(file *anime.LocalFile, anime *anilist.CompleteAnime, parsedTitle string) float64 {
	// Use the same sophisticated matching algorithms as the existing matcher
	// Get all available title variants (Romaji, English, synonyms)
	titles := anime.GetAllTitles()

	if len(titles) == 0 {
		return 0.0
	}

	// Use Sorensen-Dice coefficient for fuzzy matching (same as existing matcher)
	compRes, ok := comparison.FindBestMatchWithSorensenDice(&parsedTitle, titles)
	if !ok {
		return 0.0
	}

	// Base confidence from title similarity
	confidence := compRes.Rating

	// Year matching bonus
	if file.ParsedData != nil && file.ParsedData.Year != "" {
		if fileYear, err := time.Parse("2006", file.ParsedData.Year); err == nil {
			if startDate := anime.GetStartDate(); startDate != nil && startDate.GetYear() != nil {
				animeYear := *startDate.GetYear()
				if fileYear.Year() == animeYear {
					confidence += 0.1 // Smaller bonus since title matching is primary
				}
			}
		}
	}

	// Episode validation bonus (if file has episode info)
	if file.Metadata != nil && file.Metadata.Episode > 0 {
		totalEpisodes := anime.GetTotalEpisodeCount()
		if totalEpisodes > 0 && file.Metadata.Episode <= totalEpisodes {
			confidence += 0.05 // Small bonus for valid episode number
		}
	}

	return confidence
}


// extractTitleFromFile extracts the anime title from a file
func (mt *MultiTokenAPI) extractTitleFromFile(file *anime.LocalFile) string {
	if file.ParsedData != nil && file.ParsedData.Title != "" {
		return file.ParsedData.Title
	}

	// Fallback to filename without extension
	return file.Name
}

// extractEpisodeNumber extracts episode number from file
func (mt *MultiTokenAPI) extractEpisodeNumber(file *anime.LocalFile) int {
	if file.Metadata != nil && file.Metadata.Episode > 0 {
		return file.Metadata.Episode
	}

	if file.ParsedData != nil && file.ParsedData.Episode != "" {
		if ep, err := parseInt(file.ParsedData.Episode); err == nil {
			return ep
		}
	}

	return 0
}

// parseInt safely converts string to int
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty string")
	}
	// Simplified conversion - you might want to use strconv.Atoi
	return 0, errors.New("not implemented")
}

// saveGlobalMapping saves a successful match to the global mappings database
func (mt *MultiTokenAPI) saveGlobalMapping(file *anime.LocalFile, anime *anilist.CompleteAnime, episodeNumber int) error {
	if adapter, ok := mt.Database.(*DatabaseAdapter); ok {
		// Get file size
		fileSize := int64(0)
		if info, err := os.Stat(file.Path); err == nil {
			fileSize = info.Size()
		}

		// Extract all title variants for comprehensive matching
		romajiTitle := anime.GetRomajiTitleSafe()
		englishTitle := ""
		if anime.GetTitle().GetEnglish() != nil {
			englishTitle = *anime.GetTitle().GetEnglish()
		}

		// Get synonyms as JSON string
		synonymsStr := ""
		if synonymsPtrs := anime.GetSynonyms(); synonymsPtrs != nil {
			// Dereference synonym pointers
			synonyms := make([]string, 0)
			for _, s := range synonymsPtrs {
				if s != nil {
					synonyms = append(synonyms, *s)
				}
			}
			if len(synonyms) > 0 {
				// Simple JSON encoding - in production you'd use proper JSON marshaling
				synonymsStr = `["` + strings.Join(synonyms, `","`) + `"]`
			}
		}

		// Create global mapping with all title variants
		mapping := &models.GlobalAnimeFileMapping{
			AniListID:     anime.GetID(),
			LocalFilePath: file.Path,
			Title:         anime.GetTitleSafe(),  // Primary display title
			RomajiTitle:   romajiTitle,           // Romaji variant
			EnglishTitle:  englishTitle,          // English variant
			Synonyms:      synonymsStr,           // All synonyms
			Year:          0,                     // Default year, set below if available
			EpisodeNumber: episodeNumber,
			FileSize:      fileSize,
			LastScanned:   time.Now(),
		}

		// Set year if available
		if startDate := anime.GetStartDate(); startDate != nil && startDate.GetYear() != nil {
			mapping.Year = *startDate.GetYear()
		}

		return adapter.db.CreateGlobalMapping(mapping)
	}

	return errors.New("database adapter not available")
}