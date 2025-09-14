package handlers

import (
	"errors"
	"seanime/internal/api/anilist"
	"seanime/internal/library/scanner"
	"seanime/internal/util/limiter"
	"time"

	"github.com/labstack/echo/v4"
)

// +---------------------+
// |   Admin System Scan  |
// +---------------------+

//	@summary Start system-wide library scan
//	@desc Initiates a system-wide scan of all library paths using multi-token strategy
//	@desc Requires admin privileges
//	@route /api/v1/admin/system-scan/start [POST]
//	@returns scanner.SystemScanResult
func (h *Handler) HandleStartSystemScan(c echo.Context) error {
	type body struct {
		LibraryPaths      []string `json:"libraryPaths"`      // Override library paths
		SkipExistingFiles bool     `json:"skipExistingFiles"` // Skip files already in global mappings
		ForceRescan       bool     `json:"forceRescan"`       // Force rescan all files
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Verify admin access
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Verify admin privileges
	if !user.IsAdmin() {
		return h.RespondWithError(c, errors.New("admin privileges required"))
	}

	// Get library paths if not provided
	libraryPaths := b.LibraryPaths
	if len(libraryPaths) == 0 {
		// Get all library paths from all users (system-wide)
		paths, err := h.getAllLibraryPaths()
		if err != nil {
			return h.RespondWithError(c, err)
		}
		libraryPaths = paths
	}

	// Create system scanner
	systemScanner := scanner.NewSystemScanner(
		h.App.Database,
		h.App.AnilistPlatform,
		h.App.MetadataProvider,
		h.App.Logger,
		h.App.WSEventManager,
		limiter.NewLimiter(time.Second, 10), // Rate limit for AniList API
		anilist.NewCompleteAnimeCache(),
	)

	// Start system scan
	scanOptions := &scanner.SystemScanOptions{
		LibraryPaths:      libraryPaths,
		SkipExistingFiles: b.SkipExistingFiles,
		ForceRescan:       b.ForceRescan,
	}

	result, err := systemScanner.ScanSystem(c.Request().Context(), scanOptions)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, result)
}

//	@summary Get system scan status
//	@desc Returns the current status of system scanning operations
//	@route /api/v1/admin/system-scan/status [GET]
//	@returns map[string]interface{}
func (h *Handler) HandleGetSystemScanStatus(c echo.Context) error {
	// Verify admin access
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Verify admin privileges
	if !user.IsAdmin() {
		return h.RespondWithError(c, errors.New("admin privileges required"))
	}

	// Get system scan status from SystemScanService
	status := map[string]interface{}{
		"isScanning":         false, // TODO: Track running scans
		"autoScanEnabled":    h.App.SystemScanService.IsEnabled(),
		"isDebouncing":       h.App.SystemScanService.IsDebouncing(),
		"lastScan":           nil,   // TODO: Track last scan time
		"totalFiles":         0,     // TODO: Get from global mappings
		"mappedFiles":        0,     // TODO: Get from global mappings
		"unmappedFiles":      0,     // TODO: Get from global mappings
	}

	return h.RespondWithData(c, status)
}

// Note: Unmapped file management handlers are available in global_mapping.go
// This file focuses only on system scan functionality

// Helper functions

// getAllLibraryPaths gets all library paths from all users in the system
func (h *Handler) getAllLibraryPaths() ([]string, error) {
	// This is a simplified implementation
	// In a full system, you would query all users' library settings

	// For now, get from global settings
	libraryPath, err := h.App.Database.GetLibraryPathFromSettings()
	if err != nil {
		return nil, err
	}

	additionalPaths, err := h.App.Database.GetAdditionalLibraryPathsFromSettings()
	if err != nil {
		return nil, err
	}

	allPaths := []string{libraryPath}
	allPaths = append(allPaths, additionalPaths...)

	return allPaths, nil
}

