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

		// Check user session
		token := h.getSessionToken(c)
		if token != "" {
			user, err := h.validateUserSession(token)
			if err == nil {
				c.Set("user", user)
				// Set user context in App for AniList operations
				h.App.SetUserFromContext(user)
				return next(c)
			}
		}

		// Handle special paths that need different treatment
		if path == "/api/v1/status" {
			// Allow status requests but mark as unauthenticated
			c.Set("unauthenticated", true)
			return next(c)
		}

		// Handle Nakama client connections
		if h.App.Settings.GetNakama().Enabled && h.App.Settings.GetNakama().IsHost {
			nakamaPasswordHeader := c.Request().Header.Get("X-Seanime-Nakama-Token")

			if path == "/api/v1/nakama/ws" {
				if nakamaPasswordHeader == h.App.Settings.GetNakama().HostPassword {
					c.Response().Header().Set("X-Seanime-Nakama-Is-Client", "true")
					return next(c)
				}
			}

			if strings.HasPrefix(path, "/api/v1/nakama/host/") {
				if nakamaPasswordHeader == h.App.Settings.GetNakama().HostPassword {
					c.Response().Header().Set("X-Seanime-Nakama-Is-Client", "true")
					return next(c)
				}
			}
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
		"/api/v1/torrentstream/stream/",
		"/api/v1/nakama/stream",
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
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
