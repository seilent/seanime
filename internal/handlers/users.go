package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"seanime/internal/api/anilist"
	"seanime/internal/database/models"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// User authentication and management handlers

// AniListLoginRequest represents the AniList OAuth login request payload
type AniListLoginRequest struct {
    Token  string `json:"token" validate:"required"` // AniList access token
    DryRun bool   `json:"dryRun"`                     // Validate only, no side effects
}

// LoginResponse represents the login response
type LoginResponse struct {
	User    *models.User `json:"user"`
	Token   string       `json:"token"`
	Message string       `json:"message"`
}

// CreateUserRequest represents the create user request payload
type CreateUserRequest struct {
	Username    string `json:"username" validate:"required"` // AniList username
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

// UpdateUserRequest represents the update user request payload
type UpdateUserRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Role        *string `json:"role,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=4"`
}

// ResetPasswordRequest represents the reset password request payload (admin only)
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required,min=4"`
}

// HandleUserLogin handles AniList OAuth login
//
//	@summary AniList OAuth login
//	@desc Authenticates a user via AniList OAuth and creates a session
//	@route /api/v1/users/login [POST]
//	@returns LoginResponse
func (h *Handler) HandleUserLogin(c echo.Context) error {
    var req AniListLoginRequest
    if err := c.Bind(&req); err != nil {
        return h.RespondWithError(c, err)
    }

    // Create AniList client with provided token
    authenticatedClient := anilist.NewAnilistClient(req.Token)
    getViewer, err := authenticatedClient.GetViewer(context.Background())
    if err != nil {
        h.App.Logger.Error().Err(err).Msg("Failed to get AniList user info")
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "Failed to get user information from AniList",
        })
    }

    // DryRun: only validate token and return username. No whitelist/user/session writes.
    if req.DryRun {
        return h.RespondWithData(c, map[string]any{
            "valid":    true,
            "username": getViewer.Viewer.Name,
        })
    }

    // Check if user is in whitelist or if this is first setup
    globalSettings, err := h.App.Database.GetGlobalSettings()
    if err != nil {
        // If global settings don't exist, this is first setup
        globalSettings = nil
    }

	isWhitelisted := false
    // If setup not completed (no whitelist), block normal login
    isFirstSetup := globalSettings == nil || len(globalSettings.AnilistWhitelist) == 0
    if !isFirstSetup {
        // Check existing whitelist
        for _, whitelistedUsername := range globalSettings.AnilistWhitelist {
            if whitelistedUsername == getViewer.Viewer.Name {
                isWhitelisted = true
                break
            }
        }
    } else {
        h.App.Logger.Info().Str("username", getViewer.Viewer.Name).Msg("Login blocked: setup not completed (no whitelist)")
    }

	if !isWhitelisted {
		h.App.Logger.Warn().Str("username", getViewer.Viewer.Name).Msg("AniList user not in whitelist")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "User not authorized to access this server",
		})
	}

    // Create or update user
    user, err := h.App.Database.GetUserByUsername(getViewer.Viewer.Name)
    if err != nil {
        // User doesn't exist, create new one
        // First user during setup becomes admin, or first user in whitelist
        role := "user"
        if (!isFirstSetup) && (globalSettings != nil && len(globalSettings.AnilistWhitelist) > 0 && globalSettings.AnilistWhitelist[0] == getViewer.Viewer.Name) {
            role = "admin"
        }

		user = &models.User{
			Username:    getViewer.Viewer.Name,
			Role:        role,
			IsActive:    true,
			DisplayName: getViewer.Viewer.Name,
		}

		user, err = h.App.Database.CreateUser(user)
		if err != nil {
			return h.RespondWithError(c, err)
		}

		h.App.Logger.Info().Str("username", getViewer.Viewer.Name).Str("role", role).Msg("Created new user from AniList OAuth")
	}

	// Create or update AniList account record
	viewerBytes, _ := json.Marshal(getViewer.Viewer)
	account := &models.Account{
		UserID:   user.ID,
		Username: getViewer.Viewer.Name,
		Token:    req.Token,
		Viewer:   viewerBytes,
	}

	_, err = h.App.Database.UpsertAccount(account)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create user session (7 days expiration)
	session, err := h.App.Database.CreateUserSession(user.ID, 24*7)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     "seanime-session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	}
	c.SetCookie(cookie)

	h.App.Logger.Info().Str("username", user.Username).Msg("User logged in")

	return h.RespondWithData(c, LoginResponse{
		User:    user,
		Token:   session.Token,
		Message: "Login successful",
	})
}

// HandleUserLogout handles user logout
//
//	@summary User logout
//	@desc Logs out the current user and destroys the session
//	@route /api/v1/users/logout [POST]
//	@returns map[string]string
func (h *Handler) HandleUserLogout(c echo.Context) error {
	// Get session token from cookie or header
	token := h.getSessionToken(c)
	if token != "" {
		// Delete session from database
		h.App.Database.DeleteUserSession(token)
	}

	// Clear session cookie
	cookie := &http.Cookie{
		Name:     "seanime-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	}
	c.SetCookie(cookie)

	return h.RespondWithData(c, map[string]string{
		"message": "Logout successful",
	})
}

// HandleGetUserProfile gets the current user's profile
//
//	@summary Get user profile
//	@desc Gets the current authenticated user's profile
//	@route /api/v1/users/profile [GET]
//	@returns models.User
func (h *Handler) HandleGetUserProfile(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	return h.RespondWithData(c, user)
}

// HandleGetUserViewer gets the current user's AniList viewer data including avatar
//
//	@summary Get user AniList viewer data
//	@desc Gets the current authenticated user's AniList viewer data including avatar information
//	@route /api/v1/users/viewer [GET]
//	@returns anilist.GetViewer_Viewer
func (h *Handler) HandleGetUserViewer(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	// Get user's AniList account
	account, err := h.App.Database.GetAccountForUser(user.ID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "AniList account not found",
		})
	}

	if account == nil || len(account.Viewer) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "No AniList viewer data available",
		})
	}

	// Unmarshal viewer data
	var viewerData anilist.GetViewer_Viewer
	if err := json.Unmarshal(account.Viewer, &viewerData); err != nil {
		h.App.Logger.Err(err).Msg("Could not unmarshal viewer data")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to parse viewer data",
		})
	}

	return h.RespondWithData(c, viewerData)
}

// HandleUpdateUserProfile updates the current user's profile
//
//	@summary Update user profile
//	@desc Updates the current authenticated user's profile
//	@route /api/v1/users/profile [PATCH]
//	@returns models.User
func (h *Handler) HandleUpdateUserProfile(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Prepare updates
	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}

	// Only allow role changes for admins
	if req.Role != nil && user.IsAdmin() {
		updates["role"] = *req.Role
	}

	// Update user
	if len(updates) > 0 {
		if err := h.App.Database.UpdateUser(user.ID, updates); err != nil {
			return h.RespondWithError(c, err)
		}
	}

	// Get updated user
	updatedUser, err := h.App.Database.GetUserByID(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, updatedUser)
}

// HandleChangePassword changes the current user's password
//
//	@summary Change password
//	@desc Changes the current authenticated user's password
//	@route /api/v1/users/change-password [POST]
//	@returns map[string]string
func (h *Handler) HandleChangePassword(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Validate current password
	_, err := h.App.Database.ValidateUserPassword(user.Username, req.CurrentPassword)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Current password is incorrect",
		})
	}

	// Update password
	if err := h.App.Database.UpdateUserPassword(user.ID, req.NewPassword); err != nil {
		return h.RespondWithError(c, err)
	}

	// Invalidate all sessions for this user (force re-login)
	h.App.Database.DeleteUserSessions(user.ID)

	h.App.Logger.Info().Str("username", user.Username).Msg("User changed password")

	return h.RespondWithData(c, map[string]string{
		"message": "Password changed successfully",
	})
}

// Admin-only handlers

// HandleGetAllUsers gets all users (admin only)
//
//	@summary Get all users
//	@desc Gets all users in the system (admin only)
//	@route /api/v1/admin/users [GET]
//	@returns []models.User
func (h *Handler) HandleGetAllUsers(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	users, err := h.App.Database.GetAllUsers()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, users)
}

// HandleCreateUser creates a new user (admin only)
//
//	@summary Create user
//	@desc Creates a new user with AniList username (admin only)
//	@route /api/v1/admin/users [POST]
//	@returns models.User
func (h *Handler) HandleCreateUser(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = "user"
	}

	// Validate role
	if req.Role != "user" && req.Role != "admin" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid role. Must be 'user' or 'admin'",
		})
	}

	// Create user (AniList username only, no password)
	newUser := &models.User{
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		IsActive:    true,
	}
	
	newUser, err := h.App.Database.CreateUser(newUser)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Str("username", req.Username).Str("createdBy", user.Username).Msg("User created")

	return h.RespondWithData(c, newUser)
}

// HandleUpdateUser updates a user (admin only)
//
//	@summary Update user
//	@desc Updates a user (admin only)
//	@route /api/v1/admin/users/:id [PATCH]
//	@returns models.User
func (h *Handler) HandleUpdateUser(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Prepare updates
	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.Role != nil {
		// Validate role
		if *req.Role != "user" && *req.Role != "admin" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid role. Must be 'user' or 'admin'",
			})
		}
		updates["role"] = *req.Role
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// Update user
	if len(updates) > 0 {
		if err := h.App.Database.UpdateUser(uint(userID), updates); err != nil {
			return h.RespondWithError(c, err)
		}
	}

	// Get updated user
	updatedUser, err := h.App.Database.GetUserByID(uint(userID))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Uint("userID", uint(userID)).Str("updatedBy", user.Username).Msg("User updated")

	return h.RespondWithData(c, updatedUser)
}

// HandleDeleteUser deletes a user (admin only)
//
//	@summary Delete user
//	@desc Deletes a user (admin only)
//	@route /api/v1/admin/users/:id [DELETE]
//	@returns map[string]string
func (h *Handler) HandleDeleteUser(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	// Prevent deleting self
	if uint(userID) == user.ID {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot delete your own account",
		})
	}

	// Delete user (soft delete)
	if err := h.App.Database.DeleteUser(uint(userID)); err != nil {
		return h.RespondWithError(c, err)
	}

	// Delete all sessions for the user
	h.App.Database.DeleteUserSessions(uint(userID))

	h.App.Logger.Info().Uint("userID", uint(userID)).Str("deletedBy", user.Username).Msg("User deleted")

	return h.RespondWithData(c, map[string]string{
		"message": "User deleted successfully",
	})
}

// HandleResetUserPassword resets a user's password (admin only)
//
//	@summary Reset user password
//	@desc Resets a user's password (admin only)
//	@route /api/v1/admin/users/:id/reset-password [POST]
//	@returns map[string]string
func (h *Handler) HandleResetUserPassword(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Update password
	if err := h.App.Database.UpdateUserPassword(uint(userID), req.NewPassword); err != nil {
		return h.RespondWithError(c, err)
	}

	// Invalidate all sessions for the user
	h.App.Database.DeleteUserSessions(uint(userID))

	h.App.Logger.Info().Uint("userID", uint(userID)).Str("resetBy", user.Username).Msg("User password reset")

	return h.RespondWithData(c, map[string]string{
		"message": "Password reset successfully",
	})
}

// Whitelist management handlers

// HandleGetWhitelist gets the current AniList whitelist (admin only)
//
//	@summary Get AniList whitelist
//	@desc Gets the current AniList whitelist (admin only)
//	@route /api/v1/admin/whitelist [GET]
//	@returns []string
func (h *Handler) HandleGetWhitelist(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if globalSettings == nil || globalSettings.AnilistWhitelist == nil {
		return h.RespondWithData(c, []string{})
	}

	return h.RespondWithData(c, globalSettings.AnilistWhitelist)
}

// HandleAddToWhitelist adds a user to the AniList whitelist (admin only)
//
//	@summary Add user to whitelist
//	@desc Adds a user to the AniList whitelist (admin only)
//	@route /api/v1/admin/whitelist [POST]
//	@returns map[string]string
func (h *Handler) HandleAddToWhitelist(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username is required",
		})
	}

	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if globalSettings == nil {
		globalSettings = &models.GlobalSettings{
			BaseModel: models.BaseModel{
				ID:        1,
				UpdatedAt: time.Now(),
			},
		}
	}

	// Check if user is already in whitelist
	for _, whitelistedUser := range globalSettings.AnilistWhitelist {
		if whitelistedUser == req.Username {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "User already in whitelist",
			})
		}
	}

	// Add user to whitelist
	globalSettings.AnilistWhitelist = append(globalSettings.AnilistWhitelist, req.Username)
	globalSettings.UpdatedAt = time.Now()

	_, err = h.App.Database.UpsertGlobalSettings(globalSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Str("username", req.Username).Str("addedBy", user.Username).Msg("User added to AniList whitelist")

	return h.RespondWithData(c, map[string]string{
		"message": "User added to whitelist successfully",
	})
}

// HandleRemoveFromWhitelist removes a user from the AniList whitelist (admin only)
//
//	@summary Remove user from whitelist
//	@desc Removes a user from the AniList whitelist (admin only)
//	@route /api/v1/admin/whitelist/:username [DELETE]
//	@returns map[string]string
func (h *Handler) HandleRemoveFromWhitelist(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username is required",
		})
	}

	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if globalSettings == nil || len(globalSettings.AnilistWhitelist) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Whitelist is empty",
		})
	}

	// Find and remove user from whitelist
	found := false
	newWhitelist := make([]string, 0, len(globalSettings.AnilistWhitelist))
	for _, whitelistedUser := range globalSettings.AnilistWhitelist {
		if whitelistedUser != username {
			newWhitelist = append(newWhitelist, whitelistedUser)
		} else {
			found = true
		}
	}

	if !found {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found in whitelist",
		})
	}

	// Prevent removing the last admin (first user in whitelist)
	if len(globalSettings.AnilistWhitelist) > 0 && globalSettings.AnilistWhitelist[0] == username && len(newWhitelist) > 0 {
		// If removing the first user, make the next user admin by ensuring they're first
		// This maintains the "first user is admin" rule
		h.App.Logger.Warn().Str("removedAdmin", username).Str("newAdmin", newWhitelist[0]).Msg("Admin user removed from whitelist, promoting next user")
	}

	if len(newWhitelist) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot remove the last user from whitelist",
		})
	}

	globalSettings.AnilistWhitelist = newWhitelist
	globalSettings.UpdatedAt = time.Now()

	_, err = h.App.Database.UpsertGlobalSettings(globalSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Str("username", username).Str("removedBy", user.Username).Msg("User removed from AniList whitelist")

	return h.RespondWithData(c, map[string]string{
		"message": "User removed from whitelist successfully",
	})
}

// Setup-related handlers

// HandleSetupRequired checks if first-time setup is required
//
//	@summary Check if setup is required
//	@desc Checks if AniList whitelist is configured
//	@route /api/v1/setup/required [GET]
//	@returns map[string]bool
func (h *Handler) HandleSetupRequired(c echo.Context) error {
	// Check if AniList whitelist is configured - setup is required if whitelist is empty
	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		// If settings don't exist or error occurred, setup is required
		return h.RespondWithData(c, map[string]bool{
			"required": true,
		})
	}

	isRequired := globalSettings == nil || len(globalSettings.AnilistWhitelist) == 0

	return h.RespondWithData(c, map[string]bool{
		"required": isRequired,
	})
}

// UserPreferenceRequest represents a user preference request payload
type UserPreferenceRequest struct {
	Value interface{} `json:"value" validate:"required"`
}

// UserPreferenceResponse represents a user preference response
type UserPreferenceResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// HandleGetUserPreference handles getting a user preference
//	@desc Get a user preference by key
//	@route /api/v1/users/preferences/:key [GET]
//	@returns UserPreferenceResponse
func (h *Handler) HandleGetUserPreference(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return h.RespondWithError(c, errors.New("preference key is required"))
	}

	// Get user from session
	user, err := h.validateUserSession(h.getSessionToken(c))
	if err != nil {
		return h.RespondWithError(c, errors.New("invalid session"))
	}

	// Get preference from database
	preference, err := h.App.Database.GetUserPreference(user.ID, key)
	if err != nil {
		// If preference doesn't exist, return null
		return h.RespondWithData(c, UserPreferenceResponse{
			Key:   key,
			Value: nil,
		})
	}

	var value interface{}
	if err := json.Unmarshal([]byte(preference.Value), &value); err != nil {
		return h.RespondWithError(c, errors.New("failed to parse preference value"))
	}

	return h.RespondWithData(c, UserPreferenceResponse{
		Key:   key,
		Value: value,
	})
}

// HandleSetUserPreference handles setting a user preference
//	@desc Set a user preference by key
//	@route /api/v1/users/preferences/:key [PUT]
//	@returns UserPreferenceResponse
func (h *Handler) HandleSetUserPreference(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return h.RespondWithError(c, errors.New("preference key is required"))
	}

	var req UserPreferenceRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get user from session
	user, err := h.validateUserSession(h.getSessionToken(c))
	if err != nil {
		return h.RespondWithError(c, errors.New("invalid session"))
	}

	// Convert value to JSON string
	valueBytes, err := json.Marshal(req.Value)
	if err != nil {
		return h.RespondWithError(c, errors.New("failed to serialize preference value"))
	}

	// Save preference to database
	err = h.App.Database.SetUserPreference(user.ID, key, string(valueBytes))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, UserPreferenceResponse{
		Key:   key,
		Value: req.Value,
	})
}

// Helper methods

// getSessionToken gets the session token from cookie or Authorization header
func (h *Handler) getSessionToken(c echo.Context) string {
	// Try Authorization header first
	token := c.Request().Header.Get("Authorization")
	if token != "" {
		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			return token[7:]
		}
		return token
	}

	// Try session cookie
	cookie, err := c.Cookie("seanime-session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}


// validateUserSession validates a session token and returns the user
func (h *Handler) validateUserSession(token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("no session token provided")
	}

	session, err := h.App.Database.GetUserSession(token)
	if err != nil {
		return nil, err
	}

	return &session.User, nil
}
