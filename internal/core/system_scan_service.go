package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/database/db"
	"seanime/internal/events"
	"seanime/internal/library/scanner"
	"seanime/internal/platforms/platform"
	"seanime/internal/util"
	"seanime/internal/util/limiter"
)

type SystemScanService struct {
	logger           *zerolog.Logger
	database         *db.Database
	platform         platform.Platform // For AniList API access
	metadataProvider metadata.Provider // For metadata operations
	wsEventManager   events.WSEventManagerInterface
	sseManager       *events.SSEManager
	rateLimiter      *limiter.Limiter
	animeCache       *anilist.CompleteAnimeCache

	// File change notification handling
	fileChangeCh chan struct{}
	mu           sync.Mutex
	debouncing   bool
	enabled      bool

	// Debounce settings
	debounceDelay time.Duration

	// Cooldown settings
	cooldownDuration  time.Duration
	lastScanCompleted time.Time
}

type SystemScanServiceOptions struct {
	Logger           *zerolog.Logger
	Database         *db.Database
	Platform         platform.Platform // For AniList API access
	MetadataProvider metadata.Provider // For metadata operations
	WSEventManager   events.WSEventManagerInterface
	SSEManager       *events.SSEManager
	RateLimiter      *limiter.Limiter
	AnimeCache       *anilist.CompleteAnimeCache
	DebounceDelay    time.Duration // Optional, defaults to 5 seconds
	CooldownDuration time.Duration // Optional, defaults to 1 minute
}

func NewSystemScanService(opts *SystemScanServiceOptions) *SystemScanService {
	debounceDelay := opts.DebounceDelay
	if debounceDelay == 0 {
		debounceDelay = 5 * time.Second // Default debounce delay
	}

	cooldownDuration := opts.CooldownDuration
	if cooldownDuration == 0 {
		cooldownDuration = 1 * time.Minute // Default cooldown duration
	}

	service := &SystemScanService{
		logger:            opts.Logger,
		database:          opts.Database,
		platform:          opts.Platform,
		metadataProvider:  opts.MetadataProvider,
		wsEventManager:    opts.WSEventManager,
		sseManager:        opts.SSEManager,
		rateLimiter:       opts.RateLimiter,
		animeCache:        opts.AnimeCache,
		fileChangeCh:      make(chan struct{}, 1),
		enabled:           true,
		debounceDelay:     debounceDelay,
		cooldownDuration:  cooldownDuration,
		lastScanCompleted: time.Time{}, // Zero time initially
	}

	// Start the file change watcher goroutine
	go service.watchFileChanges()

	return service
}

// NotifyFileChange is called when a file change is detected
// It debounces multiple rapid file changes to avoid excessive system scans
func (sss *SystemScanService) NotifyFileChange() {
	if sss == nil || !sss.enabled {
		return
	}

	defer util.HandlePanicInModuleThen("core/system_scan_service/NotifyFileChange", func() {
		sss.logger.Error().Msg("system_scan_service: recovered from panic in NotifyFileChange")
	})

	sss.mu.Lock()
	defer sss.mu.Unlock()

	// If we are currently debouncing, don't send another signal
	if sss.debouncing {
		return
	}

	// Check if we're still in cooldown period after last scan completion
	if !sss.lastScanCompleted.IsZero() {
		timeSinceLastScan := time.Since(sss.lastScanCompleted)
		if timeSinceLastScan < sss.cooldownDuration {
			remainingCooldown := sss.cooldownDuration - timeSinceLastScan
			sss.logger.Debug().
				Dur("remaining_cooldown", remainingCooldown).
				Msg("system_scan_service: File change ignored due to cooldown period")
			return
		}
	}

	// Send signal to file change channel (non-blocking)
	select {
	case sss.fileChangeCh <- struct{}{}:
		sss.logger.Debug().Msg("system_scan_service: File change notification sent")
	default:
		// Channel is full, ignore this notification
		sss.logger.Debug().Msg("system_scan_service: File change channel full, ignoring notification")
	}
}

// watchFileChanges runs in a goroutine and handles debounced file change notifications
func (sss *SystemScanService) watchFileChanges() {
	defer util.HandlePanicInModuleThen("core/system_scan_service/watchFileChanges", func() {
		sss.logger.Error().Msg("system_scan_service: recovered from panic in watchFileChanges")
	})

	for {
		select {
		case <-sss.fileChangeCh:
			sss.handleFileChangeDebounced()
		}
	}
}

// handleFileChangeDebounced handles the debounced file change notification
func (sss *SystemScanService) handleFileChangeDebounced() {
	sss.mu.Lock()
	sss.debouncing = true
	sss.mu.Unlock()

	sss.logger.Info().Msgf("system_scan_service: File change detected, starting debounced system scan in %v", sss.debounceDelay)

	// Wait for the debounce period
	time.Sleep(sss.debounceDelay)

	// Perform system scan
	sss.performSystemScan()

	sss.mu.Lock()
	sss.debouncing = false
	sss.mu.Unlock()
}

// performSystemScan executes a system-wide scan with default options
func (sss *SystemScanService) performSystemScan() {
	defer util.HandlePanicInModuleThen("core/system_scan_service/performSystemScan", func() {
		sss.logger.Error().Msg("system_scan_service: recovered from panic in performSystemScan")
	})

	sss.logger.Info().Msg("system_scan_service: Starting automatic system scan due to file changes")

	// Get all library paths from the database
	libraryPaths, err := sss.getAllLibraryPaths()
	if err != nil {
		sss.logger.Error().Err(err).Msg("system_scan_service: Failed to get library paths for system scan")
		return
	}

	if len(libraryPaths) == 0 {
		sss.logger.Debug().Msg("system_scan_service: No library paths found, skipping system scan")
		return
	}

	// Create system scanner
	systemScanner := scanner.NewSystemScanner(
		sss.database,
		sss.platform,
		sss.metadataProvider,
		sss.logger,
		sss.wsEventManager,
		sss.rateLimiter,
		sss.animeCache,
	)

	// Default scan options for file change triggered scans
	scanOptions := &scanner.SystemScanOptions{
		LibraryPaths:      libraryPaths,
		SkipExistingFiles: true, // Skip files already mapped to avoid unnecessary work
		ForceRescan:       false,
	}

	// Perform the scan
	result, err := systemScanner.ScanSystem(context.Background(), scanOptions)
	if err != nil {
		sss.logger.Error().Err(err).Msg("system_scan_service: System scan failed")
		return
	}

	sss.logger.Info().
		Int("filesProcessed", result.FilesProcessed).
		Int("newMappings", result.NewMappings).
		Int("updatedMappings", result.UpdatedMappings).
		Msg("system_scan_service: Automatic system scan completed")

	// Set the last scan completion time to start cooldown period
	sss.mu.Lock()
	sss.lastScanCompleted = time.Now()
	sss.mu.Unlock()

	sss.logger.Debug().
		Dur("cooldown_duration", sss.cooldownDuration).
		Msg("system_scan_service: Scan completed, cooldown period started")

	// Send SSE event for scan completion to trigger frontend cache refresh
	if sss.sseManager != nil && result.NewMappings > 0 {
		sss.sseManager.SendEventToSSE("system-scan-completed", map[string]interface{}{
			"filesProcessed":  result.FilesProcessed,
			"newMappings":     result.NewMappings,
			"updatedMappings": result.UpdatedMappings,
			"duration":        result.Duration,
			"timestamp":       time.Now().Unix(),
			"mappedAnime":     result.MappedAnime,
		})
		sss.logger.Debug().
			Int("newMappings", result.NewMappings).
			Int("animeCount", len(result.MappedAnime)).
			Msg("system_scan_service: SSE scan completion event sent")
	}
}

// getAllLibraryPaths retrieves all library paths from settings
func (sss *SystemScanService) getAllLibraryPaths() ([]string, error) {
	// Get library path from settings
	libraryPath, err := sss.database.GetLibraryPathFromSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get library path from settings: %w", err)
	}

	// Get additional library paths from settings
	additionalPaths, err := sss.database.GetAdditionalLibraryPathsFromSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get additional library paths from settings: %w", err)
	}

	allPaths := []string{libraryPath}
	allPaths = append(allPaths, additionalPaths...)

	return allPaths, nil
}

// SetEnabled allows enabling/disabling the service
func (sss *SystemScanService) SetEnabled(enabled bool) {
	if sss == nil {
		return
	}

	sss.mu.Lock()
	defer sss.mu.Unlock()
	sss.enabled = enabled

	if enabled {
		sss.logger.Info().Msg("system_scan_service: Auto system scan on file changes enabled")
	} else {
		sss.logger.Info().Msg("system_scan_service: Auto system scan on file changes disabled")
	}
}

// IsEnabled returns whether the service is enabled
func (sss *SystemScanService) IsEnabled() bool {
	if sss == nil {
		return false
	}

	sss.mu.Lock()
	defer sss.mu.Unlock()
	return sss.enabled
}

// IsDebouncing returns whether the service is currently debouncing
func (sss *SystemScanService) IsDebouncing() bool {
	if sss == nil {
		return false
	}

	sss.mu.Lock()
	defer sss.mu.Unlock()
	return sss.debouncing
}
