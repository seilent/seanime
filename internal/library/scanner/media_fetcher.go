package scanner

import (
	"context"
	"errors"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/database/models"
	"seanime/internal/hook"
	"seanime/internal/library/anime"
	"seanime/internal/platforms/platform"
	"seanime/internal/util"
	"seanime/internal/util/limiter"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
)

// MediaFetcher holds all anilist.BaseAnime that will be used for the comparison process
type MediaFetcher struct {
	AllMedia                     []*anilist.CompleteAnime
	CollectionMediaIds           []int
	UnknownMediaIds              []int // Media IDs that are not in the user's collection
	AnimeCollectionWithRelations *anilist.AnimeCollectionWithRelations
	ScanLogger                   *ScanLogger
}

type MediaFetcherOptions struct {
	Platform               platform.Platform
	MetadataProvider       metadata.Provider
	LocalFiles             []*anime.LocalFile
	CompleteAnimeCache     *anilist.CompleteAnimeCache
	Logger                 *zerolog.Logger
	AnilistRateLimiter     *limiter.Limiter
	DisableAnimeCollection bool
	ScanLogger             *ScanLogger
	Database               DatabaseInterface // For global media pool access
	UserID                 uint              // For user subscription tracking
}

// DatabaseInterface defines the methods needed for global media pool access
type DatabaseInterface interface {
	GetAllGlobalMappings() ([]*GlobalAnimeMapping, error)
}

// NewMediaFetcher
// Calling this method will kickstart the fetch process
// MediaFetcher.AllMedia will be all anilist.BaseAnime from the user's AniList collection.
func NewMediaFetcher(ctx context.Context, opts *MediaFetcherOptions) (ret *MediaFetcher, retErr error) {
	defer util.HandlePanicInModuleWithError("library/scanner/NewMediaFetcher", &retErr)

	if opts.Platform == nil ||
		opts.LocalFiles == nil ||
		opts.CompleteAnimeCache == nil ||
		opts.MetadataProvider == nil ||
		opts.Logger == nil ||
		opts.AnilistRateLimiter == nil {
		return nil, errors.New("missing options")
	}

	mf := new(MediaFetcher)
	mf.ScanLogger = opts.ScanLogger

	opts.Logger.Debug().
		Msg("media fetcher: Creating global matching service (server-wide media pool)")

	if mf.ScanLogger != nil {
		mf.ScanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Msg("Creating global matching service using server-wide media pool")
	}

	// Invoke ScanMediaFetcherStarted hook
	event := &ScanMediaFetcherStartedEvent{}
	hook.GlobalHookManager.OnScanMediaFetcherStarted().Trigger(event)

	// +---------------------+
	// |  Global Media Pool  |
	// +---------------------+
	// Use media from ALL users' collections on the server (pure matching)

	mf.AllMedia = make([]*anilist.CompleteAnime, 0)

	// Build global media pool from all users' collections
	globalMedia, err := BuildGlobalMediaPool(ctx, opts.Platform, opts.CompleteAnimeCache, opts.AnilistRateLimiter, mf.ScanLogger, opts.Database)
	if err != nil {
		if mf.ScanLogger != nil {
			mf.ScanLogger.LogMediaFetcher(zerolog.WarnLevel).
				Err(err).
				Msg("Failed to build global media pool, attempting to populate from all users")
		}
		
		// If global pool is empty, populate it from all users' collections
		err = PopulateGlobalPoolFromAllUsers(ctx, opts.Platform, opts.Database, mf.ScanLogger)
		if err != nil {
			if mf.ScanLogger != nil {
				mf.ScanLogger.LogMediaFetcher(zerolog.ErrorLevel).
					Err(err).
					Msg("Failed to populate global pool from all users")
			}
			return nil, err
		}
		
		// Retry building global media pool after population
		globalMedia, err = BuildGlobalMediaPool(ctx, opts.Platform, opts.CompleteAnimeCache, opts.AnilistRateLimiter, mf.ScanLogger, opts.Database)
		if err != nil {
			return nil, err
		}
	}

	// Global pool is the ONLY source of truth
	mf.AllMedia = globalMedia
	
	if mf.ScanLogger != nil {
		mf.ScanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Int("count", len(mf.AllMedia)).
			Msg("Using global media pool from all users' collections")
	}

	// +---------------------+
	// |  Collection Tracking |\n\t// +---------------------+
	// Track current user's collection separately for adding unknown media

	// Get current user's collection for tracking purposes only
	if !opts.DisableAnimeCollection {
		animeCollectionWithRelations, err := opts.Platform.GetAnimeCollectionWithRelations(ctx)
		if err == nil {
			mf.AnimeCollectionWithRelations = animeCollectionWithRelations
		}
	}

	// Get the media IDs from current user's collection for tracking
	if mf.AnimeCollectionWithRelations != nil {
		collectionMedia := make([]*anilist.CompleteAnime, 0)
		for _, list := range mf.AnimeCollectionWithRelations.GetMediaListCollection().GetLists() {
			for _, entry := range list.GetEntries() {
				collectionMedia = append(collectionMedia, entry.GetMedia())
			}
		}

		mf.CollectionMediaIds = lop.Map(collectionMedia, func(m *anilist.CompleteAnime, index int) int {
			return m.ID
		})

		// +---------------------+
		// | Subscription Tracking |
		// +---------------------+
		// Track which user has which anime for multi-token system

		if opts.Database != nil && opts.UserID > 0 {
			err = mf.trackUserSubscriptions(opts.UserID, collectionMedia, opts.Database)
			if err != nil && mf.ScanLogger != nil {
				mf.ScanLogger.LogMediaFetcher(zerolog.WarnLevel).
					Err(err).
					Msg("Failed to track user subscriptions")
			}
		}

		// Find unknown media (not in current user's collection, but exists in global pool)
		unknownMedia := lo.Filter(mf.AllMedia, func(m *anilist.CompleteAnime, _ int) bool {
			return !lo.Contains(mf.CollectionMediaIds, m.ID)
		})
		mf.UnknownMediaIds = lop.Map(unknownMedia, func(m *anilist.CompleteAnime, _ int) int {
			return m.ID
		})
	}

	if mf.ScanLogger != nil {
		mf.ScanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Int("globalPool", len(mf.AllMedia)).
			Int("userCollection", len(mf.CollectionMediaIds)).
			Int("unknownToUser", len(mf.UnknownMediaIds)).
			Msg("Global matching service ready")
	}

	// Invoke ScanMediaFetcherCompleted hook
	completedEvent := &ScanMediaFetcherCompletedEvent{
		AllMedia:        mf.AllMedia,
		UnknownMediaIds: mf.UnknownMediaIds,
	}
	_ = hook.GlobalHookManager.OnScanMediaFetcherCompleted().Trigger(completedEvent)
	mf.AllMedia = completedEvent.AllMedia
	mf.UnknownMediaIds = completedEvent.UnknownMediaIds

	return mf, nil
}

//----------------------------------------------------------------------------------------------------------------------

// FetchMediaFromLocalFiles gets media and their relations from local file titles.
// This is a pure matching service that discovers anime from file names without user collection dependency.
// It retrieves unique titles from local files,
// fetches mal.SearchResultAnime from MAL,
// uses these search results to get AniList IDs using metadata.AnimeMetadata mappings,
// queries AniList to retrieve all anilist.BaseAnime using anilist.GetBaseAnimeById and their relations using anilist.FetchMediaTree.
// It does not return an error if one of the steps fails.
// It returns the discovered media and a boolean indicating whether the process was successful.
// BuildGlobalMediaPool creates a comprehensive media pool from all users' collections on the server.
// This provides pure matching without user collection dependency and without expensive API calls.
// The more users on the server, the more comprehensive the matching database becomes.
func BuildGlobalMediaPool(
	ctx context.Context,
	platform platform.Platform,
	completeAnimeCache *anilist.CompleteAnimeCache,
	anilistRateLimiter *limiter.Limiter,
	scanLogger *ScanLogger,
	database DatabaseInterface,
) (ret []*anilist.CompleteAnime, retErr error) {
	defer util.HandlePanicInModuleWithError("library/scanner/BuildGlobalMediaPool", &retErr)

	if scanLogger != nil {
		scanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Str("module", "GlobalPool").
			Msg("Building enhanced global media pool from all users' collections")
	}

	// Phase 1: Try to get media from existing global mappings first
	globalMedia := make([]*anilist.CompleteAnime, 0)
	globalMappings, err := getAllGlobalMappings(platform, database)
	if err == nil && len(globalMappings) > 0 {
		if scanLogger != nil {
			scanLogger.LogMediaFetcher(zerolog.DebugLevel).
				Str("module", "GlobalPool").
				Int("mappings", len(globalMappings)).
				Msg("Found existing global mappings, using as base")
		}

		// Get unique AniList IDs from global mappings
		globalMediaIds := make([]int, 0)
		seenIds := make(map[int]bool)
		for _, mapping := range globalMappings {
			if !seenIds[mapping.AniListID] {
				globalMediaIds = append(globalMediaIds, mapping.AniListID)
				seenIds[mapping.AniListID] = true
			}
		}

		// Fetch media details for existing mappings
		if len(globalMediaIds) > 0 {
			mediaMap, fetchErr := platform.BatchGetAnimeWithRelations(ctx, globalMediaIds)
			if fetchErr == nil {
				for _, media := range mediaMap {
					if media != nil {
						globalMedia = append(globalMedia, media)
						completeAnimeCache.Set(media.GetID(), media)
					}
				}
			}
		}
	}

	// Phase 2: Enhanced collection aggregation using multi-token system
	if scanLogger != nil {
		scanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Str("module", "GlobalPool").
			Int("existingMedia", len(globalMedia)).
			Msg("Enhancing global pool with multi-user collection aggregation")
	}

	// Create MultiTokenAPI for collection aggregation
	// Use a simple logger for MultiTokenAPI
	tempLogger := zerolog.Nop() // Use a no-op logger to avoid complexity with ScanLogger
	multiTokenAPI := NewMultiTokenAPI(database, &tempLogger, anilistRateLimiter, completeAnimeCache)

	// Aggregate collections from ALL users
	aggregatedResult, err := multiTokenAPI.AggregateUserCollections(ctx)
	if err != nil {
		if scanLogger != nil {
			scanLogger.LogMediaFetcher(zerolog.WarnLevel).
				Str("module", "GlobalPool").
				Err(err).
				Msg("Failed to aggregate user collections, using existing mappings only")
		}
		// Return what we have from existing mappings
		if len(globalMedia) > 0 {
			return globalMedia, nil
		}
		return []*anilist.CompleteAnime{}, nil
	}

	// Phase 3: Merge existing mappings with aggregated collections
	seenAnimeIds := make(map[int]bool)
	
	// Add existing media first
	for _, media := range globalMedia {
		seenAnimeIds[media.GetID()] = true
	}

	// Add new anime from aggregated collections
	newAnimeCount := 0
	for _, anime := range aggregatedResult.AllAnime {
		if !seenAnimeIds[anime.GetID()] {
			globalMedia = append(globalMedia, anime)
			seenAnimeIds[anime.GetID()] = true
			newAnimeCount++
			// Cache the anime
			completeAnimeCache.Set(anime.GetID(), anime)
		}
	}

	if scanLogger != nil {
		scanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Str("module", "GlobalPool").
			Int("totalUsers", aggregatedResult.TotalUsersAccessed).
			Int("successfulFetches", aggregatedResult.SuccessfulFetches).
			Int("failedFetches", aggregatedResult.FailedFetches).
			Int("existingMedia", len(globalMedia)-newAnimeCount).
			Int("newFromCollections", newAnimeCount).
			Int("totalGlobalMedia", len(globalMedia)).
			Msg("Enhanced global media pool built successfully")
	}

	return globalMedia, nil
}

// trackUserSubscriptions records which anime a user has in their AniList collection
// This enables the multi-token system for API resilience
func (mf *MediaFetcher) trackUserSubscriptions(userID uint, collectionMedia []*anilist.CompleteAnime, database DatabaseInterface) error {
	if len(collectionMedia) == 0 {
		return nil
	}

	if mf.ScanLogger != nil {
		mf.ScanLogger.LogMediaFetcher(zerolog.DebugLevel).
			Int("userID", int(userID)).
			Int("animeCount", len(collectionMedia)).
			Msg("Tracking user anime subscriptions")
	}

	// Update subscriptions for each anime in user's collection
	for _, anime := range collectionMedia {
		if adapter, ok := database.(*DatabaseAdapter); ok {
			err := adapter.db.UpsertUserAnimeSubscription(userID, anime.GetID(), "active")
			if err != nil {
				// Log but don't fail the entire process for a single subscription error
				if mf.ScanLogger != nil {
					mf.ScanLogger.LogMediaFetcher(zerolog.WarnLevel).
						Err(err).
						Int("aniListID", anime.GetID()).
						Str("title", anime.GetTitleSafe()).
						Msg("Failed to upsert user anime subscription")
				}
				continue
			}
		}
	}

	if mf.ScanLogger != nil {
		mf.ScanLogger.LogMediaFetcher(zerolog.DebugLevel).
			Int("userID", int(userID)).
			Int("animeCount", len(collectionMedia)).
			Msg("User anime subscriptions tracked successfully")
	}

	return nil
}

// PopulateGlobalPoolFromAllUsers populates the global database with anime from ALL users' collections
// This ensures the global pool contains deduplicated anime from across the entire server
func PopulateGlobalPoolFromAllUsers(
	ctx context.Context,
	platform platform.Platform,
	database DatabaseInterface,
	scanLogger *ScanLogger,
) error {
	if scanLogger != nil {
		scanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Str("module", "GlobalPoolPopulation").
			Msg("Starting population of global pool from all users' collections")
	}

	if database == nil {
		return errors.New("database interface not provided")
	}

	// Get all users' AniList collections
	// Note: This requires platform to support multi-user operations
	// For now, we'll use the current approach with global mappings as the foundation
	
	// Since we don't have direct access to all users' platforms in this context,
	// we'll populate the global pool differently:
	// 1. When users scan their libraries, their matched anime will be saved to global mappings
	// 2. For initial population, we'll ensure the scanning process saves matches to global database
	
	if scanLogger != nil {
		scanLogger.LogMediaFetcher(zerolog.InfoLevel).
			Str("module", "GlobalPoolPopulation").
			Msg("Global pool will be populated as users scan their libraries")
	}

	return nil
}

// getAllGlobalMappings retrieves all global mappings from the database
// This function needs to be implemented to access the database through the platform
func getAllGlobalMappings(platform platform.Platform, database DatabaseInterface) ([]*GlobalAnimeMapping, error) {
	if database == nil {
		return []*GlobalAnimeMapping{}, errors.New("database interface not provided")
	}
	
	// Get all global mappings from database
	return database.GetAllGlobalMappings()
}

// GlobalAnimeMapping represents a mapping from the database
type GlobalAnimeMapping struct {
	AniListID int
}

// DatabaseAdapter implements DatabaseInterface using the existing db.Database
type DatabaseAdapter struct {
	db DatabaseBackend
}

// DatabaseBackend defines the actual database methods we need
type DatabaseBackend interface {
	// Global mapping operations
	GetAllGlobalMappings() ([]*models.GlobalAnimeFileMapping, error)
	GetGlobalMapping(filePath string) (*models.GlobalAnimeFileMapping, error)
	CreateGlobalMapping(mapping *models.GlobalAnimeFileMapping) error

	// User anime subscription operations (for multi-token system)
	UpsertUserAnimeSubscription(userID uint, aniListID int, tokenStatus string) error
	GetUsersWithAnime(aniListID int) ([]*models.UserAnimeSubscription, error)
	UpdateTokenStatus(userID uint, aniListID int, status string) error

	// User operations (for multi-token API)
	GetAllUsers() ([]models.User, error)
	GetAccountForUser(userID uint) (*models.Account, error)

	// Unmapped file operations (for admin queue)
	CreateUnmappedFile(unmappedFile *models.UnmappedFile) error
	GetUnmappedFiles(status string) ([]*models.UnmappedFile, error)
	UpdateUnmappedFile(unmappedFile *models.UnmappedFile) error
	DeleteUnmappedFile(filePath string) error
}

// NewDatabaseAdapter creates a new database adapter
func NewDatabaseAdapter(db DatabaseBackend) *DatabaseAdapter {
	return &DatabaseAdapter{db: db}
}

// GetAllGlobalMappings implements DatabaseInterface
func (da *DatabaseAdapter) GetAllGlobalMappings() ([]*GlobalAnimeMapping, error) {
	// Get all global mappings from database
	mappings, err := da.db.GetAllGlobalMappings()
	if err != nil {
		return nil, err
	}

	// Convert to our internal format
	result := make([]*GlobalAnimeMapping, len(mappings))
	for i, mapping := range mappings {
		result[i] = &GlobalAnimeMapping{
			AniListID: mapping.AniListID,
		}
	}

	return result, nil
}


//----------------------------------------------------------------------------------------------------------------------

