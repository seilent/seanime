# Multi-User Implementation Plan for Seanime

## Current Architecture Analysis

Seanime is currently a single-user application with:
- Single `Account` record (ID=1) for AniList authentication
- Global settings shared across the application
- Server-level password authentication (optional)
- All data tied to the single user instance

## Implementation Strategy

### Phase 1: Database Schema Changes

#### New Tables
```go
// User management
type User struct {
    BaseModel
    Username     string `gorm:"unique;not null" json:"username"`
    PasswordHash string `gorm:"not null" json:"-"`
    Role         string `gorm:"default:'user'" json:"role"` // "admin", "user"
    IsActive     bool   `gorm:"default:true" json:"isActive"`
    DisplayName  string `json:"displayName"`
}

type UserSession struct {
    BaseModel
    UserID    uint      `gorm:"not null" json:"userId"`
    Token     string    `gorm:"unique;not null" json:"token"`
    ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
    User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

#### Modified Existing Tables
Add `UserID` foreign key to user-specific data:
- `Account` (AniList tokens)
- `Settings` (user preferences)
- `LocalFiles` (scan results)
- `ScanSummary`
- `AutoDownloaderRule`
- `PlaylistEntry`
- `ChapterDownloadQueueItem`
- `Theme`
- All other user-specific data

### Phase 2: Authentication System

#### New Handlers
- `POST /api/v1/users/login` - User login
- `POST /api/v1/users/logout` - User logout
- `GET /api/v1/users/profile` - Get current user profile
- `PATCH /api/v1/users/profile` - Update user profile

#### Admin Handlers
- `GET /api/v1/admin/users` - List all users
- `POST /api/v1/admin/users` - Create new user
- `PATCH /api/v1/admin/users/:id` - Update user
- `DELETE /api/v1/admin/users/:id` - Delete user
- `POST /api/v1/admin/users/:id/reset-password` - Reset user password

#### Authentication Middleware
Replace `OptionalAuthMiddleware` with user-aware authentication:
```go
func (h *Handler) UserAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Check server password first (if set)
        if h.App.Config.Server.Password != "" {
            // Server password logic
        }
        
        // Check user session
        token := c.Request().Header.Get("Authorization")
        if token == "" {
            cookie, _ := c.Cookie("seanime-session")
            if cookie != nil {
                token = cookie.Value
            }
        }
        
        if token != "" {
            user, err := h.validateUserSession(token)
            if err == nil {
                c.Set("user", user)
                return next(c)
            }
        }
        
        return c.JSON(401, map[string]string{"error": "Authentication required"})
    }
}
```

### Phase 3: User Context in Handlers

#### Modify Existing Handlers
Update all handlers to be user-aware:
```go
func (h *Handler) HandleGetSettings(c echo.Context) error {
    user := c.Get("user").(*models.User)
    settings, err := h.App.Database.GetSettingsForUser(user.ID)
    // ... rest of handler
}
```

#### Database Layer Changes
Add user-scoped queries:
```go
func (db *Database) GetSettingsForUser(userID uint) (*models.Settings, error) {
    var settings models.Settings
    err := db.gormdb.Where("user_id = ?", userID).First(&settings).Error
    return &settings, err
}
```

### Phase 4: Migration Strategy

#### Database Migration
```go
func MigrateToMultiUser(db *gorm.DB) error {
    // 1. Create new tables
    db.AutoMigrate(&models.User{}, &models.UserSession{})
    
    // 2. Add UserID columns to existing tables
    // 3. Create default admin user
    // 4. Assign existing data to admin user
    // 5. Add foreign key constraints
}
```

#### Backward Compatibility
- Existing single-user installations automatically get an admin user
- All existing data is assigned to the admin user
- Server password (if set) becomes admin password initially

### Phase 5: Frontend Changes

#### Login Interface
- Simple login form for username/password
- Session management with cookies
- User profile management

#### Admin Interface
- User management panel (admin only)
- Create/edit/delete users
- Password reset functionality

## Implementation Benefits

1. **Shared Media Files**: All users access the same media library
2. **Individual AniList Accounts**: Each user can connect their own AniList
3. **Personal Settings**: Each user has their own preferences and themes
4. **Individual Progress**: Separate watch/read progress per user
5. **Admin Control**: Admin can manage all users and system settings

## File Structure Changes

```
internal/
├── handlers/
│   ├── users.go          # New: User management handlers
│   ├── admin.go          # New: Admin-only handlers
│   └── auth.go           # Modified: Multi-user auth
├── database/
│   ├── models/
│   │   ├── user.go       # New: User models
│   │   └── models.go     # Modified: Add UserID fields
│   └── migrations/
│       └── multiuser.go  # New: Migration scripts
└── middleware/
    └── auth.go           # New: User authentication middleware
```

## Security Considerations

1. **Password Hashing**: Use bcrypt for password storage
2. **Session Management**: Secure session tokens with expiration
3. **Role-Based Access**: Admin vs regular user permissions
4. **Data Isolation**: Ensure users can only access their own data
5. **Server Password**: Optional server-level password for additional security

## Estimated Timeline

- **Phase 1** (Database): 2-3 days
- **Phase 2** (Auth System): 3-4 days
- **Phase 3** (Handler Updates): 4-5 days
- **Phase 4** (Migration): 2-3 days
- **Phase 5** (Frontend): 3-4 days
- **Testing & Polish**: 3-4 days

**Total**: ~3-4 weeks for complete implementation

## Next Steps

1. Start with database schema changes
2. Implement basic authentication system
3. Update core handlers one by one
4. Add admin interface
5. Test migration from single-user to multi-user
