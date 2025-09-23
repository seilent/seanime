package db

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"seanime/internal/database/models"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User management operations

// CreateUserWithPassword creates a new user with hashed password (legacy method)
func (db *Database) CreateUserWithPassword(username, password, displayName, role string) (*models.User, error) {
	// Check if username already exists
	var existingUser models.User
	if err := db.gormdb.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	// Password hashing removed for AniList OAuth - this method is deprecated
	_ = password // Silence unused parameter warning

	// Set default role if not provided
	if role == "" {
		role = "user"
	}

	// Set default display name if not provided
	if displayName == "" {
		displayName = username
	}

	user := &models.User{
		Username:    username,
		DisplayName: displayName,
		Role:        role,
		IsActive:    true,
	}

	if err := db.gormdb.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// CreateUser creates a new user from a User struct (for AniList OAuth)
func (db *Database) CreateUser(user *models.User) (*models.User, error) {
	// Check if username already exists
	var existingUser models.User
	if err := db.gormdb.Where("username = ?", user.Username).First(&existingUser).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	// Set default values if not provided
	if user.Role == "" {
		user.Role = "user"
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	user.IsActive = true

	if err := db.gormdb.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (db *Database) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := db.gormdb.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (db *Database) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := db.gormdb.Where("id = ? AND is_active = ?", id, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ValidateUserPassword is no longer used (AniList OAuth replaces password auth)
// Kept for backwards compatibility but will always return error
func (db *Database) ValidateUserPassword(username, password string) (*models.User, error) {
	return nil, errors.New("password authentication is no longer supported - use AniList OAuth")
}

// UpdateUserPassword updates a user's password
func (db *Database) UpdateUserPassword(userID uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return db.gormdb.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", string(hashedPassword)).Error
}

// UpdateUser updates user information
func (db *Database) UpdateUser(userID uint, updates map[string]interface{}) error {
	// Don't allow updating password through this method
	delete(updates, "password_hash")
	delete(updates, "id")

	return db.gormdb.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

// GetAllUsers retrieves all users (admin only)
func (db *Database) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := db.gormdb.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteUser soft deletes a user by setting IsActive to false
func (db *Database) DeleteUser(userID uint) error {
	return db.gormdb.Model(&models.User{}).Where("id = ?", userID).Update("is_active", false).Error
}

// Session management operations

// generateSessionToken generates a random session token
func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateUserSession creates a new user session
func (db *Database) CreateUserSession(userID uint, expirationHours int) (*models.UserSession, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	if expirationHours <= 0 {
		expirationHours = 24 * 7 // Default to 7 days
	}

	session := &models.UserSession{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(expirationHours) * time.Hour),
	}

	if err := db.gormdb.Create(session).Error; err != nil {
		return nil, err
	}

	// Load user data
	if err := db.gormdb.Preload("User").First(session, session.ID).Error; err != nil {
		return nil, err
	}

	return session, nil
}

// GetUserSession retrieves a session by token
func (db *Database) GetUserSession(token string) (*models.UserSession, error) {
	var session models.UserSession
	if err := db.gormdb.Preload("User").Where("token = ?", token).First(&session).Error; err != nil {
		return nil, err
	}

	// Check if session is expired
	if session.IsExpired() {
		// Delete expired session
		db.gormdb.Delete(&session)
		return nil, errors.New("session expired")
	}

	// Check if user is still active
	if !session.User.IsActive {
		return nil, errors.New("user account is inactive")
	}

	return &session, nil
}

// DeleteUserSession deletes a session (logout)
func (db *Database) DeleteUserSession(token string) error {
	return db.gormdb.Where("token = ?", token).Delete(&models.UserSession{}).Error
}

// DeleteUserSessions deletes all sessions for a user
func (db *Database) DeleteUserSessions(userID uint) error {
	return db.gormdb.Where("user_id = ?", userID).Delete(&models.UserSession{}).Error
}

// CleanupExpiredSessions removes expired sessions
func (db *Database) CleanupExpiredSessions() error {
	return db.gormdb.Where("expires_at < ?", time.Now()).Delete(&models.UserSession{}).Error
}

// User-specific data operations

// GetSettingsForUser retrieves settings for a specific user
func (db *Database) GetSettingsForUser(userID uint) (*models.Settings, error) {
	var settings models.Settings
	if err := db.gormdb.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default settings for user
			return db.CreateDefaultSettingsForUser(userID)
		}
		return nil, err
	}
	return &settings, nil
}

// CreateDefaultSettingsForUser creates default settings for a new user
func (db *Database) CreateDefaultSettingsForUser(userID uint) (*models.Settings, error) {
	settings := &models.Settings{
		UserID:              userID,
		AutoPlayNextEpisode: false,
		AutoUpdateProgress:  true,
		MediaPlayer: &models.MediaPlayerSettings{
			Default:     "vlc",
			Host:        "127.0.0.1",
			VlcUsername: "",
			VlcPassword: "",
			VlcPort:     8080,
			VlcPath:     "",
			MpcPort:     13579,
			MpcPath:     "",
			MpvSocket:   "",
			MpvPath:     "",
			MpvArgs:     "",
			IinaSocket:  "",
			IinaPath:    "",
			IinaArgs:    "",
		},
		Manga: &models.MangaSettings{
			DefaultProvider:      "mangadex",
			AutoUpdateProgress:   true,
			LocalSourceDirectory: "",
		},
		Anilist: &models.AnilistSettings{
			HideAudienceScore:  false,
			EnableAdultContent: false,
			BlurAdultContent:   true,
		},
		ListSync: &models.ListSyncSettings{
			Automatic: false,
			Origin:    "anilist",
		},
		Discord: &models.DiscordSettings{
			EnableRichPresence:                      false,
			EnableAnimeRichPresence:                 true,
			EnableMangaRichPresence:                 true,
			RichPresenceHideSeanimeRepositoryButton: false,
			RichPresenceShowAniListMediaButton:      true,
			RichPresenceShowAniListProfileButton:    true,
			RichPresenceUseMediaTitleStatus:         true,
		},
		Notifications: &models.NotificationSettings{
			DisableNotifications:               false,
			DisableAutoDownloaderNotifications: false,
			DisableAutoScannerNotifications:    false,
		},
		ClientMedia: &models.ClientMediaSettings{
			DirectPlayOnly: false,
		},
	}

	if err := db.gormdb.Create(settings).Error; err != nil {
		return nil, err
	}

	return settings, nil
}

// SaveSettingsForUser saves settings for a specific user
func (db *Database) SaveSettingsForUser(userID uint, settings *models.Settings) error {
	settings.UserID = userID
	return db.gormdb.Save(settings).Error
}

// GetAccountForUser retrieves account for a specific user
func (db *Database) GetAccountForUser(userID uint) (*models.Account, error) {
	var account models.Account
	if err := db.gormdb.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create empty account for user
			account = models.Account{
				UserID:   userID,
				Username: "",
				Token:    "",
				Viewer:   nil,
			}
			if err := db.gormdb.Create(&account).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &account, nil
}

// UpsertAccountForUser creates or updates account for a specific user
func (db *Database) UpsertAccountForUser(userID uint, account *models.Account) (*models.Account, error) {
	account.UserID = userID

	var existingAccount models.Account
	if err := db.gormdb.Where("user_id = ?", userID).First(&existingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new account
			if err := db.gormdb.Create(account).Error; err != nil {
				return nil, err
			}
			return account, nil
		}
		return nil, err
	}

	// Update existing account
	account.ID = existingAccount.ID
	if err := db.gormdb.Save(account).Error; err != nil {
		return nil, err
	}

	return account, nil
}

// CreateFirstTimeSetup creates the first admin user during initial setup
func (db *Database) CreateFirstTimeSetup(username, password, displayName string) (*models.User, error) {
	// Check if any users already exist
	var count int64
	if err := db.gormdb.Model(&models.User{}).Count(&count).Error; err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, errors.New("users already exist, first-time setup not allowed")
	}

	// Create the first admin user
	user, err := db.CreateUserWithPassword(username, password, displayName, "admin")
	if err != nil {
		return nil, err
	}

	return user, nil
}
