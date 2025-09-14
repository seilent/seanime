package scanner

import (
	"context"
	"fmt"
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/library/anime"
	"seanime/internal/library/filesystem"
	"seanime/internal/platforms/platform"
	"seanime/internal/util"
	"seanime/internal/util/limiter"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// SystemScanner handles system-wide library scanning using multi-token strategy
type SystemScanner struct {
	Database           DatabaseBackend         // Changed from DatabaseInterface to DatabaseBackend
	Platform           platform.Platform       // For AniList API access
	MetadataProvider   metadata.Provider       // For metadata operations
	Logger             *zerolog.Logger
	WSEventManager     events.WSEventManagerInterface
	AnilistRateLimiter *limiter.Limiter
	CompleteAnimeCache *anilist.CompleteAnimeCache
	MultiTokenAPI      *MultiTokenAPI // For token rotation strategy
}

// SystemScanOptions contains options for system-level scanning
type SystemScanOptions struct {
	LibraryPaths       []string // All library paths from all users
	SkipExistingFiles  bool     // Skip files already in global mappings
	ForceRescan        bool     // Force rescan all files
}

// MappedAnimeInfo contains information about a newly mapped anime
type MappedAnimeInfo struct {
	AniListID    int    `json:"anilistId"`
	Title        string `json:"title"`
	EpisodeCount int    `json:"episodeCount"`
	Episodes     []int  `json:"episodes"`
}

// SystemScanResult contains the results of a system scan
type SystemScanResult struct {
	FilesProcessed    int                `json:"filesProcessed"`
	NewMappings       int                `json:"newMappings"`
	UpdatedMappings   int                `json:"updatedMappings"`
	Errors            []string           `json:"errors"`
	Duration          string             `json:"duration"`
	UnmappedFilePaths []string           `json:"unmappedFilePaths"` // For admin queue
	MappedAnime       []*MappedAnimeInfo `json:"mappedAnime"`       // Detailed info about newly mapped anime
}

// NewSystemScanner creates a new system scanner
func NewSystemScanner(database DatabaseBackend, platform platform.Platform, metadataProvider metadata.Provider, logger *zerolog.Logger, wsEventManager events.WSEventManagerInterface, rateLimiter *limiter.Limiter, cache *anilist.CompleteAnimeCache) *SystemScanner {
	multiTokenAPI := NewMultiTokenAPI(NewDatabaseAdapter(database), logger, rateLimiter, cache)

	return &SystemScanner{
		Database:           database,
		Platform:           platform,
		MetadataProvider:   metadataProvider,
		Logger:             logger,
		WSEventManager:     wsEventManager,
		AnilistRateLimiter: rateLimiter,
		CompleteAnimeCache: cache,
		MultiTokenAPI:      multiTokenAPI,
	}
}

// ScanSystem performs a system-wide scan of all library paths
func (ss *SystemScanner) ScanSystem(ctx context.Context, options *SystemScanOptions) (*SystemScanResult, error) {
	startTime := time.Now()

	ss.Logger.Info().
		Strs("libraryPaths", options.LibraryPaths).
		Bool("skipExisting", options.SkipExistingFiles).
		Msg("system-scanner: Starting system-wide library scan")

	if ss.WSEventManager != nil {
		ss.WSEventManager.SendEvent("system-scan-started", map[string]interface{}{
			"libraryPaths": options.LibraryPaths,
			"timestamp":   startTime,
		})
	}

	result := &SystemScanResult{
		UnmappedFilePaths: make([]string, 0),
	}

	// Step 1: Discover all media files across all library paths
	allFiles, err := ss.discoverFiles(options.LibraryPaths)
	if err != nil {
		return nil, err
	}

	result.FilesProcessed = len(allFiles)
	ss.Logger.Info().Int("totalFiles", result.FilesProcessed).Msg("system-scanner: File discovery completed")

	if ss.WSEventManager != nil {
		ss.WSEventManager.SendEvent("system-scan-progress", map[string]interface{}{
			"phase":       "discovery",
			"totalFiles":  result.FilesProcessed,
			"progress":    20,
		})
	}

	// Step 2: Filter out already mapped files (if not forcing rescan)
	filesToProcess := allFiles
	if !options.ForceRescan && options.SkipExistingFiles {
		filesToProcess = ss.filterUnmappedFiles(allFiles)
		result.NewMappings = len(filesToProcess)
		ss.Logger.Info().Int("newFiles", result.NewMappings).Msg("system-scanner: Filtered new files")
	} else {
		result.NewMappings = len(allFiles)
	}

	if ss.WSEventManager != nil {
		ss.WSEventManager.SendEvent("system-scan-progress", map[string]interface{}{
			"phase":      "filtering",
			"newFiles":   result.NewMappings,
			"progress":   40,
		})
	}

	// Step 3: Match files using multi-token strategy
	mappedCount, unmappedFiles, errorCount, mappedAnimeList, err := ss.matchFilesWithMultiToken(ctx, filesToProcess)
	if err != nil {
		return nil, err
	}

	result.UpdatedMappings = mappedCount
	result.UnmappedFilePaths = unmappedFiles
	result.MappedAnime = mappedAnimeList

	// Collect error messages
	var errorMessages []string
	if errorCount > 0 {
		errorMessages = append(errorMessages, fmt.Sprintf("%d files failed to process", errorCount))
	}
	result.Errors = errorMessages

	// Step 4: Add unmapped files to admin queue
	if len(unmappedFiles) > 0 {
		err = ss.addUnmappedFilesToQueue(unmappedFiles)
		if err != nil {
			ss.Logger.Warn().Err(err).Msg("system-scanner: Failed to add unmapped files to admin queue")
		}
	}

	result.Duration = time.Since(startTime).String()

	ss.Logger.Info().
		Int("filesProcessed", result.FilesProcessed).
		Int("newMappings", result.NewMappings).
		Int("updatedMappings", result.UpdatedMappings).
		Int("unmappedFiles", len(result.UnmappedFilePaths)).
		Int("errorCount", len(result.Errors)).
		Str("duration", result.Duration).
		Msg("system-scanner: System scan completed")

	if ss.WSEventManager != nil {
		ss.WSEventManager.SendEvent("system-scan-completed", map[string]interface{}{
			"result":    result,
			"timestamp": time.Now(),
		})
	}

	return result, nil
}

// discoverFiles finds all media files in the given library paths
func (ss *SystemScanner) discoverFiles(libraryPaths []string) ([]*anime.LocalFile, error) {
	var allFiles []*anime.LocalFile

	for _, libraryPath := range libraryPaths {
		if _, err := os.Stat(libraryPath); os.IsNotExist(err) {
			ss.Logger.Warn().Str("path", libraryPath).Msg("system-scanner: Library path does not exist, skipping")
			continue
		}

		// Get media file paths from directory
		filePaths, err := filesystem.GetMediaFilePathsFromDir(libraryPath)
		if err != nil {
			ss.Logger.Error().Err(err).Str("path", libraryPath).Msg("system-scanner: Failed to scan directory")
			continue
		}

		// Convert paths to LocalFile objects
		for _, filePath := range filePaths {
			localFile := anime.NewLocalFile(filePath, libraryPath)
			if localFile != nil {
				allFiles = append(allFiles, localFile)
			}
		}

		ss.Logger.Debug().Str("path", libraryPath).Int("files", len(filePaths)).Msg("system-scanner: Scanned directory")
	}

	return allFiles, nil
}

// filterUnmappedFiles removes files that are already in global mappings
func (ss *SystemScanner) filterUnmappedFiles(files []*anime.LocalFile) []*anime.LocalFile {
	if len(files) == 0 {
		return files
	}

	// Get all existing global mappings
	existingMappings, err := ss.Database.GetAllGlobalMappings()
	if err != nil {
		ss.Logger.Warn().Err(err).Msg("system-scanner: Failed to get existing mappings, processing all files")
		return files
	}

	// Create a map of existing file paths for fast lookup
	existingPaths := make(map[string]bool)
	for _, mapping := range existingMappings {
		existingPaths[util.NormalizePath(mapping.LocalFilePath)] = true
	}

	// Filter out files that already exist in mappings
	var unmappedFiles []*anime.LocalFile
	for _, file := range files {
		if !existingPaths[util.NormalizePath(file.Path)] {
			unmappedFiles = append(unmappedFiles, file)
		}
	}

	ss.Logger.Debug().
		Int("total", len(files)).
		Int("existing", len(existingMappings)).
		Int("unmapped", len(unmappedFiles)).
		Msg("system-scanner: Filtered unmapped files")

	return unmappedFiles
}

// matchFilesWithMultiToken matches files using the multi-token strategy
func (ss *SystemScanner) matchFilesWithMultiToken(ctx context.Context, files []*anime.LocalFile) (int, []string, int, []*MappedAnimeInfo, error) {
	if len(files) == 0 {
		return 0, []string{}, 0, []*MappedAnimeInfo{}, nil
	}

	ss.Logger.Info().Int("files", len(files)).Msg("system-scanner: Starting enhanced multi-token collection-based matching")

	// Step 1: Use Enhanced MediaFetcher with multi-token collection aggregation
	// This replaces the direct collection aggregation with the sophisticated MediaFetcher pipeline
	
	// Create database adapter for MediaFetcher
	dbAdapter := NewDatabaseAdapter(ss.Database)
	
	// Create MediaFetcher options with multi-token support
	mediaFetcherOpts := &MediaFetcherOptions{
		Platform:               ss.Platform,
		MetadataProvider:       ss.MetadataProvider,
		LocalFiles:             files,
		CompleteAnimeCache:     ss.CompleteAnimeCache,
		Logger:                 ss.Logger,
		AnilistRateLimiter:     ss.AnilistRateLimiter,
		DisableAnimeCollection: true, // We want pure global matching, not user-specific collections
		ScanLogger:             nil,  // Use regular logger for now
		Database:               dbAdapter,
		UserID:                 0, // No specific user for system scan
	}

	// Create Enhanced MediaFetcher - this will aggregate collections from ALL users
	mediaFetcher, err := NewMediaFetcher(ctx, mediaFetcherOpts)
	if err != nil {
		return 0, []string{}, 0, []*MappedAnimeInfo{}, fmt.Errorf("failed to create enhanced media fetcher: %w", err)
	}

	if len(mediaFetcher.AllMedia) == 0 {
		ss.Logger.Warn().Msg("system-scanner: No anime found in enhanced global media pool")
		// Return all files as unmapped
		unmappedFiles := make([]string, len(files))
		for i, file := range files {
			unmappedFiles[i] = file.Path
		}
		return 0, unmappedFiles, 0, []*MappedAnimeInfo{}, nil
	}

	ss.Logger.Info().
		Int("globalMedia", len(mediaFetcher.AllMedia)).
		Msg("system-scanner: Using enhanced global media pool for matching")

	// Step 2: Create MediaContainer from enhanced MediaFetcher
	// The MediaFetcher has already done the sophisticated collection aggregation
	mediaContainer := NewMediaContainer(&MediaContainerOptions{
		AllMedia:   mediaFetcher.AllMedia,
		ScanLogger: nil, // We'll use our own logger
	})

	ss.Logger.Info().
		Int("normalizedMedia", len(mediaContainer.NormalizedMedia)).
		Msg("system-scanner: Created media container from enhanced global pool")

	// Step 3: Use existing Matcher with the enhanced media pool
	// This preserves ALL sophisticated title normalization and episode mapping logic
	matcher := &Matcher{
		LocalFiles:         files,
		MediaContainer:     mediaContainer,
		CompleteAnimeCache: ss.CompleteAnimeCache,
		Logger:             ss.Logger,
		Algorithm:          "sorensen-dice", // Use proven algorithm
		Threshold:          0.6,             // Reasonable threshold
	}

	// Step 4: Run the existing matching logic with sophisticated title normalization
	err = matcher.MatchLocalFilesWithMedia()
	if err != nil {
		ss.Logger.Warn().Err(err).Msg("system-scanner: Matching process encountered errors, continuing")
	}

	// Step 5: Count results and save successful matches to global mappings
	mappedCount := 0
	errorCount := 0
	var unmappedFiles []string
	mappedAnimeMap := make(map[int]*MappedAnimeInfo) // Track anime details by AniList ID

	for i, file := range files {
		if file.MediaId != 0 {
			// File was successfully matched - save to global mappings with rich metadata
			anime, exists := ss.CompleteAnimeCache.Get(file.MediaId)
			if exists && anime != nil {
				episodeNumber := 0
				if file.Metadata != nil && file.Metadata.Episode > 0 {
					episodeNumber = file.Metadata.Episode
				} else if file.ParsedData != nil && file.ParsedData.Episode != "" {
					// Parse episode from filename since Metadata.Episode is not populated in system scan
					if ep, err := strconv.Atoi(file.ParsedData.Episode); err == nil {
						episodeNumber = ep
					}
				}
				
				// Save to global mappings using MultiTokenAPI (which has the saveGlobalMapping method)
				saveErr := ss.MultiTokenAPI.saveGlobalMapping(file, anime, episodeNumber)
				if saveErr != nil {
					ss.Logger.Warn().
						Err(saveErr).
						Str("file", file.Path).
						Int("mediaId", file.MediaId).
						Msg("system-scanner: Failed to save global mapping")
					errorCount++
				} else {
					mappedCount++

					// Collect anime details for SSE event payload
					if mappedInfo, exists := mappedAnimeMap[file.MediaId]; exists {
						// Add episode to existing anime
						mappedInfo.EpisodeCount++
						mappedInfo.Episodes = append(mappedInfo.Episodes, episodeNumber)
					} else {
						// Create new anime info
						mappedAnimeMap[file.MediaId] = &MappedAnimeInfo{
							AniListID:    file.MediaId,
							Title:        anime.GetTitleSafe(),
							EpisodeCount: 1,
							Episodes:     []int{episodeNumber},
						}
					}

					ss.Logger.Debug().
						Str("file", file.Path).
						Int("mediaId", file.MediaId).
						Str("title", anime.GetTitleSafe()).
						Int("episode", episodeNumber).
						Msg("system-scanner: Successfully mapped file to global database")
				}
			} else {
				ss.Logger.Warn().
					Str("file", file.Path).
					Int("mediaId", file.MediaId).
					Msg("system-scanner: Matched file but anime not in cache")
				errorCount++
			}
		} else {
			// File was not matched
			unmappedFiles = append(unmappedFiles, file.Path)
		}

		// Report progress periodically
		if (i+1)%50 == 0 || i == len(files)-1 {
			progress := int((float64(i+1) / float64(len(files))) * 40) + 40 // 40-80% range
			if ss.WSEventManager != nil {
				ss.WSEventManager.SendEvent("system-scan-progress", map[string]interface{}{
					"phase":       "matching",
					"processed":   i + 1,
					"totalFiles":  len(files),
					"mapped":      mappedCount,
					"unmapped":    len(unmappedFiles),
					"progress":    progress,
				})
			}
		}
	}

	ss.Logger.Info().
		Int("mapped", mappedCount).
		Int("unmapped", len(unmappedFiles)).
		Int("errors", errorCount).
		Msg("system-scanner: Enhanced collection-based matching completed")

	// Convert mapped anime map to slice for return
	mappedAnimeList := make([]*MappedAnimeInfo, 0, len(mappedAnimeMap))
	for _, animeInfo := range mappedAnimeMap {
		mappedAnimeList = append(mappedAnimeList, animeInfo)
	}

	return mappedCount, unmappedFiles, errorCount, mappedAnimeList, nil
}

// matchSingleFileWithMultiToken attempts to match a single file using multi-token strategy
func (ss *SystemScanner) matchSingleFileWithMultiToken(ctx context.Context, file *anime.LocalFile) (bool, error) {
	// This method is obsolete - we now use collection aggregation instead of per-file matching
	ss.Logger.Debug().Str("file", file.Path).Msg("system-scanner: matchSingleFileWithMultiToken is deprecated, using collection aggregation")
	return false, nil
}

// extractTitle extracts anime title from filename
func (ss *SystemScanner) extractTitle(file *anime.LocalFile) string {
	if file.ParsedData != nil && file.ParsedData.Title != "" {
		return file.ParsedData.Title
	}

	// Fallback to basic filename parsing if ParsedData is not available
	// This is a simplified version - you might want to use the existing parsing logic
	return file.Name
}

// addUnmappedFilesToQueue adds unmapped files to the admin queue
func (ss *SystemScanner) addUnmappedFilesToQueue(unmappedFiles []string) error {
	if len(unmappedFiles) == 0 {
		return nil
	}

	ss.Logger.Info().Int("count", len(unmappedFiles)).Msg("system-scanner: Adding unmapped files to admin queue")

	for _, filePath := range unmappedFiles {
		// Extract basic info about the file
		parsedTitle := ""
		fileSize := int64(0)

		// Get file size
		if info, err := os.Stat(filePath); err == nil {
			fileSize = info.Size()
		}

		// Create LocalFile to extract parsed title
		if localFile := anime.NewLocalFile(filePath, ""); localFile != nil {
			parsedTitle = ss.extractTitle(localFile)
		}

		unmappedFile := &models.UnmappedFile{
			LocalFilePath: filePath,
			ParsedTitle:   parsedTitle,
			FileSize:      fileSize,
			LastDetected:  time.Now(),
			Status:        "PENDING",
		}

		err := ss.Database.CreateUnmappedFile(unmappedFile)
		if err != nil {
			ss.Logger.Warn().Err(err).Str("file", filePath).Msg("system-scanner: Failed to add unmapped file to queue")
		}
	}

	return nil
}