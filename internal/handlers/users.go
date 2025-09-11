package handlers

import (
	"errors"
	"net/http"
	"seanime/internal/database/models"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// User authentication and management handlers

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	User    *models.User `json:"user"`
	Token   string       `json:"token"`
	Message string       `json:"message"`
}

// CreateUserRequest represents the create user request payload
type CreateUserRequest struct {
	Username    string `json:"username" validate:"required"`
	Password    string `json:"password" validate:"required,min=4"`
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

// HandleUserLogin handles user login
//
//	@summary User login
//	@desc Authenticates a user and creates a session
//	@route /api/v1/users/login [POST]
//	@returns LoginResponse
func (h *Handler) HandleUserLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return h.RespondWithError(c, err)
	}

	// Validate user credentials
	user, err := h.App.Database.ValidateUserPassword(req.Username, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid username or password",
		})
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
//	@desc Creates a new user (admin only)
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

	// Create user
	newUser, err := h.App.Database.CreateUser(req.Username, req.Password, req.DisplayName, req.Role)
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

// Setup-related handlers

// HandleSetupRequired checks if first-time setup is required
//
//	@summary Check if setup is required
//	@desc Checks if any users exist in the system
//	@route /api/v1/setup/required [GET]
//	@returns map[string]bool
func (h *Handler) HandleSetupRequired(c echo.Context) error {
	isRequired := !h.App.Database.IsMultiUserEnabled()
	
	return h.RespondWithData(c, map[string]bool{
		"required": isRequired,
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

// getCurrentUser gets the current authenticated user from context
func (h *Handler) getCurrentUser(c echo.Context) *models.User {
	if user := c.Get("user"); user != nil {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
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
