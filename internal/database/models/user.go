package models

import (
	"time"
)

// User represents a user account in the multi-user system
type User struct {
	BaseModel
	Username    string `gorm:"unique;not null" json:"username"` // AniList username
	Role        string `gorm:"default:'user'" json:"role"`      // "admin", "user"
	IsActive    bool   `gorm:"default:true" json:"isActive"`
	DisplayName string `json:"displayName"`
}

// UserSession represents an active user session
type UserSession struct {
	BaseModel
	UserID    uint      `gorm:"not null;index" json:"userId"`
	Token     string    `gorm:"unique;not null;index" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// IsExpired checks if the session has expired
func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// TableName returns the table name for UserSession
func (UserSession) TableName() string {
	return "user_sessions"
}

// UserPreference represents user-specific preferences/settings
type UserPreference struct {
	BaseModel
	UserID uint   `gorm:"not null;index" json:"userId"`
	Key    string `gorm:"not null;index" json:"key"`
	Value  string `gorm:"type:text" json:"value"`
	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for UserPreference
func (UserPreference) TableName() string {
	return "user_preferences"
}
