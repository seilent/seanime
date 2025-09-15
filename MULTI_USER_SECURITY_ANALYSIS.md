# 🚨 CRITICAL MULTI-USER SECURITY ANALYSIS - COMPLETE ISOLATION FAILURE

**Date:** 2025-09-15
**Status:** CRITICAL - IMMEDIATE ACTION REQUIRED
**Impact:** Complete user isolation breakdown in Seanime multi-user system

## EXECUTIVE SUMMARY

Seanime's multi-user implementation has **COMPLETE USER ISOLATION FAILURE**. Users can access each other's data, control each other's playback, and see each other's activity in real-time. This analysis identified 30+ critical vulnerabilities across authentication, data access, and event broadcasting.

## CRITICAL VULNERABILITIES IDENTIFIED

### 1. **GLOBAL SINGLETON MANAGERS** ⚠️ CRITICAL

#### PlaybackManager - Single User Context
**File:** `internal/library/playbackmanager/playback_manager.go:71`
```go
currentUserID uint // User ID for the current playback session
```
**Issue:** Single field gets overwritten by each user. When User B starts playback, User A's session is lost.

**Impact:**
- User B's playback overwrites User A's active session
- Progress tracking uses wrong user ID (line 206, 416)
- All users see whoever's currently playing

#### DirectStreamManager - Global Stream State
**File:** `internal/directstream/manager.go:42`
```go
currentStream mo.Option[Stream] // The current stream being played
```
**Issue:** Single stream slot for entire server. Starting new stream terminates others.

**Impact:**
- User A's stream gets cancelled when User B starts streaming
- No isolation between users' streaming sessions

#### SystemScanService - Global Platform Usage
**File:** `internal/core/system_scan_service.go:24,188`
```go
platform platform.Platform // For AniList API access
// Later used as:
systemScanner := scanner.NewSystemScanner(sss.platform)
```
**Issue:** Uses global platform instead of user-specific platforms for system scans.

**Impact:**
- System scans use wrong user's AniList metadata
- File matching and metadata updates affect wrong user's data

### 2. **AUTHENTICATION BYPASS** ⚠️ CRITICAL - 9 Handlers Missing Auth

**File:** `internal/handlers/playback_manager.go`

| Handler | Line | Status |
|---------|------|--------|
| `HandlePlaybackPlayRandomVideo` | 53-64 | ❌ NO AUTH |
| `HandlePlaybackSyncCurrentProgress` | 73-83 | ❌ NO AUTH |
| `HandlePlaybackPlayNextEpisode` | 92-100 | ❌ NO AUTH |
| `HandlePlaybackGetNextEpisode` | 108-112 | ❌ NO AUTH |
| `HandlePlaybackAutoPlayNextEpisode` | 120-128 | ❌ NO AUTH |
| `HandlePlaybackCancelCurrentPlaylist` | 175-183 | ❌ NO AUTH |
| `HandlePlaybackPlaylistNext` | 191-199 | ❌ NO AUTH |
| `HandlePlaybackStartManualTracking` | 211-232 | ❌ NO AUTH |
| `HandlePlaybackCancelManualTracking` | 240-245 | ❌ NO AUTH |

**Working Examples (for comparison):**
- `HandlePlaybackPlayVideo` (lines 27-30): ✅ HAS AUTH
- `HandlePlaybackStartPlaylist` (lines 150-153): ✅ HAS AUTH

### 3. **GLOBAL DATA ACCESS** ⚠️ CRITICAL

#### Random Video - Wrong Data Sources
**File:** `internal/library/playbackmanager/play_random_episode.go`

```go
// Line 25: Uses GLOBAL platform
animeCollection, err := pm.platform.GetAnimeCollection(context.Background(), false)

// Line 35: Uses ALL users' files
lfs, _, err := db_bridge.GetLocalFiles(pm.Database)
```

**Should use:**
- User-specific platform: `GetUserPlatform(c)`
- User-specific files: `db_bridge.GetLocalFilesForUser(pm.Database, userID)`

#### Widespread GetLocalFiles() Misuse
**Pattern:** Most handlers use global `GetLocalFiles()` instead of user-specific `GetLocalFilesForUser()`

**Affected Files:**
- `internal/handlers/anime.go:37`
- `internal/handlers/playlist.go:38,115,185`
- `internal/handlers/localfiles.go:27,37,114,166,211,275,344`
- `internal/handlers/anime_entries.go:124,189,266,355,498`
- And 15+ more locations...

**Correct Usage (examples):**
- `internal/handlers/directstream.go:35` ✅ Uses `GetLocalFilesForUser()`
- `internal/handlers/scan.go:51` ✅ Uses `GetLocalFilesForUser()`

### 4. **WEBSOCKET EVENT LEAKAGE** ⚠️ CRITICAL

#### Global Broadcasting
**File:** `internal/events/sse.go:64-79`
```go
// BroadcastEvent sends an event to all connected clients
func (m *SSEManager) BroadcastEvent(eventType string, data interface{}) {
    // Sends to ALL connections without user filtering
    for _, conn := range connections {
        m.sendEventToConnection(conn, eventType, data)
    }
}
```

#### Affected Events (Broadcast to All Users)
**File:** `internal/library/playbackmanager/progress_tracking.go` and others

- `PlaybackManagerProgressTrackingStarted` (line 78)
- `PlaybackManagerProgressVideoCompleted` (line 167)
- `PlaybackManagerProgressTrackingStopped` (line 182)
- `PlaybackManagerProgressPlaybackState` (line 246)
- `ErrorToast` messages (line 98)
- `InfoToast` messages (line 639, 651)
- Playlist state changes (line 631)

**Impact:** All users see everyone's playback status, progress updates, and error messages.

### 5. **PROGRESS TRACKING CROSS-CONTAMINATION** ⚠️ HIGH

**File:** `internal/library/playbackmanager/progress_tracking.go:206,416`
```go
// Uses stored pm.currentUserID for continuity tracking
pm.continuityManager.UpdateExternalPlayerEpisodeWatchHistoryItemForUser(
    pm.currentUserID, // This is whoever was last playing!
    pm.currentMediaPlaybackStatus.CurrentTimeInSeconds,
    pm.currentMediaPlaybackStatus.DurationInSeconds
)
```

**Issue:** Progress updates use the stored `currentUserID`, which belongs to whoever was last playing, not necessarily the current user.

## ATTACK SCENARIOS

### Scenario 1: Session Hijacking via Random Video
1. User A logs in and has a large anime collection
2. Attacker (no login) calls `/api/v1/playback-manager/play-random`
3. System plays random video from User A's collection
4. Attacker gets access to User A's media

### Scenario 2: Playback Interference
1. User A starts watching Episode 1 of Anime X
2. User B starts watching Episode 5 of Anime Y
3. User A's session is terminated, progress lost
4. All users see User B's playback in their UI

### Scenario 3: Data Leak via Events
1. User A marks episode as completed (private action)
2. All connected users receive completion event
3. Users can see what others are watching in real-time

### Scenario 4: Progress Corruption
1. User A watches Episode 1, pauses at 50%
2. User B starts different video
3. When User A resumes, progress is saved to User B's account
4. User A's progress is lost, User B gets corrupted data

## TECHNICAL IMPACT DETAILS

### Authentication Bypass Impact
- **9 critical endpoints** accessible without login
- Remote code execution via media player control
- Unauthorized access to user's anime collections
- Ability to manipulate other users' playlists

### Data Isolation Failure Impact
- Users can see/access files from ALL users
- Cross-user metadata pollution
- System scans affect wrong user accounts
- Background services operate on wrong data

### Event Broadcasting Impact
- Complete loss of user privacy
- Real-time surveillance of user activity
- Error messages leak sensitive paths/info
- UI shows other users' states

## ROOT CAUSE ANALYSIS

### Architectural Problem
The system was originally designed as **single-user**, then multi-user authentication was **bolted on top** without refactoring the core singleton managers.

### Core Issues:
1. **Singleton Pattern Misuse** - One manager instance for all users
2. **Missing User Context** - Operations don't carry user identity
3. **Global State Mutation** - Shared state gets overwritten
4. **Broadcast-Only Events** - No user-specific event routing

## REMEDIATION STRATEGY

### Phase 1: IMMEDIATE CRITICAL FIXES (Priority 1)

#### 1.1 Authentication Bypass (Estimated: 2-4 hours)
```go
// Fix each missing auth handler:
func (h *Handler) HandlePlaybackPlayRandomVideo(c echo.Context) error {
    user := h.getCurrentUser(c)
    if user == nil {
        return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
    }
    // ... rest of handler
}
```

#### 1.2 Data Access Fixes (Estimated: 4-6 hours)
- Replace all `GetLocalFiles()` with `GetLocalFilesForUser(userID)`
- Fix random video to use user platform: `GetUserPlatform(c)`
- Audit all handlers for user-specific data access

#### 1.3 Event Broadcasting Fix (Estimated: 6-8 hours)
```go
// Add user-specific broadcasting
func (m *SSEManager) BroadcastEventToUser(userID uint, eventType string, data interface{})
func (m *SSEManager) BroadcastEventToUsers(userIDs []uint, eventType string, data interface{})
```

### Phase 2: ARCHITECTURAL REFACTOR (Priority 2)

#### Option A: Per-User Manager Instances (Estimated: 2-3 weeks)
```go
type UserManagerHub struct {
    playbackManagers    map[uint]*PlaybackManager
    directStreamManagers map[uint]*DirectStreamManager
    mu                  sync.RWMutex
}
```

#### Option B: User Context Propagation (Estimated: 1-2 weeks)
```go
type UserContext struct {
    UserID   uint
    Platform platform.Platform
}

// All manager methods accept UserContext
func (pm *PlaybackManager) StartRandomVideo(ctx UserContext, opts *StartRandomVideoOptions)
```

#### Option C: User-Scoped Service Layer (Estimated: 2-3 weeks)
```go
type UserScopedServices struct {
    UserID              uint
    Platform            platform.Platform
    PlaybackManager     *PlaybackManager
    DirectStreamManager *DirectStreamManager
}
```

### Phase 3: SYSTEMATIC AUDIT (Priority 3)

#### 3.1 Background Services Review
- AutoDownloader user isolation
- AutoScanner user context
- FileWatcher user filtering
- All cron jobs and background tasks

#### 3.2 Database Access Patterns
- Audit all database queries for user filtering
- Review all `db_bridge.*` functions
- Check cache implementations for user isolation

#### 3.3 Testing Strategy
- Multi-user integration tests
- Concurrent user scenario testing
- Security penetration testing
- Load testing with multiple users

## RECOMMENDED IMMEDIATE ACTIONS

### 1. **DISABLE VULNERABLE ENDPOINTS** (Immediate)
Comment out or disable the 9 handlers missing authentication until fixes are implemented.

### 2. **IMPLEMENT AUTHENTICATION FIXES** (Within 24 hours)
Add `getCurrentUser()` checks to all vulnerable handlers.

### 3. **FIX RANDOM VIDEO HANDLER** (Within 48 hours)
This is the highest impact vulnerability - completely bypasses authentication.

### 4. **IMPLEMENT USER-SPECIFIC EVENTS** (Within 1 week)
Stop broadcasting sensitive events to all users.

### 5. **DATA ACCESS AUDIT** (Within 1 week)
Replace global data access with user-specific access throughout the codebase.

## MONITORING AND DETECTION

### Immediate Monitoring Needs:
1. **Authentication bypass attempts** - Monitor 401/403 responses
2. **Cross-user data access** - Log user ID with all data operations
3. **Concurrent sessions** - Alert on multiple active sessions per user
4. **Event broadcasting** - Log event recipients for audit trails

### Security Logging:
```go
// Add to all fixed handlers
logger.Info().
    Uint("user_id", user.ID).
    Str("endpoint", c.Path()).
    Msg("User action authenticated")
```

## CONCLUSION

This analysis reveals **COMPLETE MULTI-USER SECURITY BREAKDOWN** in Seanime. The system currently provides **NO USER ISOLATION** and requires immediate action to prevent data breaches, privacy violations, and system interference between users.

**Severity: CRITICAL**
**Action Required: IMMEDIATE**
**Estimated Fix Time: 1-3 weeks for complete remediation**

The recommended approach is to implement Phase 1 fixes immediately, then proceed with architectural refactoring in Phase 2 to establish proper multi-user isolation from the ground up.