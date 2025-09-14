package handlers

import (
	"strings"
	"seanime/internal/database/models"

	"github.com/labstack/echo/v4"
)

// UserAuthMiddleware handles user authentication for the multi-user system
func (h *Handler) UserAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Request().URL.Path

		// Allow certain paths without user authentication
		if h.isPublicPath(path) {
			return next(c)
		}

		// Allow setup-related APIs during initial setup (when whitelist is empty)
		if h.isSetupPath(path) {
			settings, err := h.App.Database.GetSettings()
			if err != nil || len(settings.AnilistWhitelist) == 0 {
				// No whitelist configured, allow setup APIs
				return next(c)
			}
		}

		// Check user session
		token := h.getSessionToken(c)
		if token != "" {
			user, err := h.validateUserSession(token)
			if err == nil {
				c.Set("user", user)
				// User context is now handled per-request in individual handlers
				return next(c)
			}
		}

		// Handle special paths that need different treatment
		if path == "/api/v1/status" {
			// Allow status requests but mark as unauthenticated
			c.Set("unauthenticated", true)
			return next(c)
		}


		return c.JSON(401, map[string]string{"error": "Authentication required"})
	}
}

// isPublicPath checks if a path should be accessible without user authentication
func (h *Handler) isPublicPath(path string) bool {
	publicPaths := []string{
		"/api/v1/users/login",
		"/api/v1/users/logout",
		"/events",
		"/api/v1/image-proxy",
		"/api/v1/proxy",
		"/api/v1/manga/local-page",
	}

	// Check exact matches
	for _, publicPath := range publicPaths {
		if path == publicPath {
			return true
		}
	}

	// Check prefixes for streaming endpoints (used by media players)
	publicPrefixes := []string{
		"/api/v1/directstream",
		"/api/v1/mediastream/att/",
		"/api/v1/mediastream/direct",
		"/api/v1/mediastream/transcode/",
		"/api/v1/mediastream/subs/",
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// isSetupPath checks if a path should be accessible during initial setup (when no users exist)
func (h *Handler) isSetupPath(path string) bool {
	setupPaths := []string{
		"/api/v1/directory-selector",    // Needed for library path selection
		"/api/v1/start",                 // Getting started API endpoint
	}

	// Check exact matches
	for _, setupPath := range setupPaths {
		if path == setupPath {
			return true
		}
	}

	return false
}

// RequireAdmin middleware ensures the current user is an admin
func (h *Handler) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := h.getCurrentUser(c)
		if user == nil {
			return c.JSON(401, map[string]string{"error": "Authentication required"})
		}

		if !user.IsAdmin() {
			return c.JSON(403, map[string]string{"error": "Admin access required"})
		}

		return next(c)
	}
}

// OptionalAuth middleware allows both authenticated and unauthenticated access
func (h *Handler) OptionalAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Try to get user from session, but don't require it
		token := h.getSessionToken(c)
		if token != "" {
			user, err := h.validateUserSession(token)
			if err == nil {
				c.Set("user", user)
			}
		}

		return next(c)
	}
}

// getCurrentUser extracts the current user from the Echo context
func (h *Handler) getCurrentUser(c echo.Context) *models.User {
	user, ok := c.Get("user").(*models.User)
	if !ok {
		return nil
	}
	return user
}

