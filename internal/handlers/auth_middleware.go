package handlers

import (
	"errors"
	"strings"

	"github.com/labstack/echo/v4"
)

// UserAuthMiddleware handles user authentication for the multi-user system
func (h *Handler) UserAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Request().URL.Path

		// Check if multi-user is enabled
		if !h.App.Database.IsMultiUserEnabled() {
			// Fall back to legacy server password authentication
			return h.legacyServerPasswordAuth(next)(c)
		}

		// Allow certain paths without user authentication
		if h.isPublicPath(path) {
			return next(c)
		}

		// Check server password first (if set)
		if h.App.Config.Server.Password != "" {
			passwordHash := c.Request().Header.Get("X-Seanime-Token")
			if passwordHash != h.App.ServerPasswordHash {
				// Check HMAC token in query parameter
				token := c.Request().URL.Query().Get("token")
				if token != "" {
					hmacAuth := h.App.GetServerPasswordHMACAuth()
					_, err := hmacAuth.ValidateToken(token, path)
					if err != nil {
						h.App.Logger.Debug().Err(err).Str("path", path).Msg("server auth: HMAC token validation failed")
						return h.RespondWithError(c, errors.New("UNAUTHENTICATED"))
					}
				} else {
					return h.RespondWithError(c, errors.New("UNAUTHENTICATED"))
				}
			}
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

// legacyServerPasswordAuth provides backward compatibility for single-user mode
func (h *Handler) legacyServerPasswordAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.App.Config.Server.Password == "" {
			return next(c)
		}

		path := c.Request().URL.Path
		passwordHash := c.Request().Header.Get("X-Seanime-Token")

		// Allow the following paths to be accessed by anyone
		if path == "/api/v1/auth/login" || // for auth
			path == "/api/v1/auth/logout" || // for auth
			path == "/api/v1/status" || // for interface
			path == "/events" || // for server events
			strings.HasPrefix(path, "/api/v1/directstream") || // used by media players
			strings.HasPrefix(path, "/api/v1/mediastream/att/") || // used by media players
			strings.HasPrefix(path, "/api/v1/mediastream/direct") || // used by media players
			strings.HasPrefix(path, "/api/v1/mediastream/transcode/") || // used by media players
			strings.HasPrefix(path, "/api/v1/mediastream/subs/") || // used by media players
			strings.HasPrefix(path, "/api/v1/image-proxy") || // used by img tag
			strings.HasPrefix(path, "/api/v1/proxy") || // used by video players
			strings.HasPrefix(path, "/api/v1/manga/local-page") || // used by img tag
			strings.HasPrefix(path, "/api/v1/torrentstream/stream/") || // accessible by media players
			strings.HasPrefix(path, "/api/v1/nakama/stream") { // accessible by media players

			if path == "/api/v1/status" {
				// allow status requests by anyone but mark as unauthenticated
				// so we can filter out critical info like settings
				if passwordHash != h.App.ServerPasswordHash {
					c.Set("unauthenticated", true)
				}
			}

			return next(c)
		}

		if passwordHash == h.App.ServerPasswordHash {
			return next(c)
		}

		// Check HMAC token in query parameter
		token := c.Request().URL.Query().Get("token")
		if token != "" {
			hmacAuth := h.App.GetServerPasswordHMACAuth()
			_, err := hmacAuth.ValidateToken(token, path)
			if err == nil {
				return next(c)
			} else {
				h.App.Logger.Debug().Err(err).Str("path", path).Msg("server auth: HMAC token validation failed")
			}
		}

		// Handle Nakama client connections
		if h.App.Settings.GetNakama().Enabled && h.App.Settings.GetNakama().IsHost {
			// Verify the Nakama host password in the client request
			nakamaPasswordHeader := c.Request().Header.Get("X-Seanime-Nakama-Token")

			// Allow WebSocket connections for peer-to-host communication
			if path == "/api/v1/nakama/ws" {
				if nakamaPasswordHeader == h.App.Settings.GetNakama().HostPassword {
					c.Response().Header().Set("X-Seanime-Nakama-Is-Client", "true")
					return next(c)
				}
			}

			// Only allow the following paths to be accessed by Nakama clients
			if strings.HasPrefix(path, "/api/v1/nakama/host/") {
				if nakamaPasswordHeader == h.App.Settings.GetNakama().HostPassword {
					c.Response().Header().Set("X-Seanime-Nakama-Is-Client", "true")
					return next(c)
				}
			}
		}

		return h.RespondWithError(c, errors.New("UNAUTHENTICATED"))
	}
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
