package models

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
	"time"
)

type BaseModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Token struct {
	BaseModel
	Value string `json:"value"`
}

type Account struct {
	BaseModel
	UserID   uint   `gorm:"column:user_id;unique" json:"userId"` // Foreign key to User
	Username string `gorm:"column:username" json:"username"`
	Token    string `gorm:"column:token" json:"token"`
	Viewer   []byte `gorm:"column:viewer" json:"viewer"`
}

// +---------------------+
// |     LocalFiles      |
// +---------------------+

type LocalFiles struct {
	BaseModel
	UserID uint   `gorm:"column:user_id;index" json:"userId"` // Foreign key to User
	Value  []byte `gorm:"column:value" json:"value"`
}

// +---------------------+
// |       Settings      |
// +---------------------+

// GlobalSettings - server-wide settings stored in separate table
type GlobalSettings struct {
	BaseModel
	SetupCompleted   bool                    `gorm:"column:setup_completed" json:"setupCompleted"`
	AnilistWhitelist StringSlice             `gorm:"column:anilist_whitelist;type:text" json:"anilistWhitelist"`
	Library          *LibrarySettings        `gorm:"embedded" json:"library"`
	Torrent          *TorrentSettings        `gorm:"embedded" json:"torrent"`
	AutoDownloader   *AutoDownloaderSettings `gorm:"embedded" json:"autoDownloader"`
}

// Settings - per-user settings
type Settings struct {
	BaseModel
	UserID                  uint                   `gorm:"column:user_id;index" json:"userId"` // Foreign key to User
	AutoPlayNextEpisode     bool                   `gorm:"column:auto_play_next_episode" json:"autoPlayNextEpisode"`
	AutoUpdateProgress      bool                   `gorm:"column:auto_update_progress" json:"autoUpdateProgress"`
	MediaPlayer             *MediaPlayerSettings   `gorm:"embedded" json:"mediaPlayer"`
	Anilist                 *AnilistSettings       `gorm:"embedded" json:"anilist"`
	ListSync                *ListSyncSettings      `gorm:"embedded" json:"listSync"`
	Discord                 *DiscordSettings       `gorm:"embedded" json:"discord"`
	Notifications           *NotificationSettings  `gorm:"embedded" json:"notifications"`
	Manga                   *MangaSettings         `gorm:"embedded" json:"manga"`
	// Virtual fields populated from GlobalSettings for frontend compatibility
	Library        *LibrarySettings        `gorm:"-" json:"library"`        // Populated from GlobalSettings
	Torrent        *TorrentSettings        `gorm:"-" json:"torrent"`        // Populated from GlobalSettings
	AutoDownloader *AutoDownloaderSettings `gorm:"-" json:"autoDownloader"` // Populated from GlobalSettings
}

type AnilistSettings struct {
	//AnilistClientId    string `gorm:"column:anilist_client_id" json:"anilistClientId"`
	HideAudienceScore  bool `gorm:"column:hide_audience_score" json:"hideAudienceScore"`
	EnableAdultContent bool `gorm:"column:enable_adult_content" json:"enableAdultContent"`
	BlurAdultContent   bool `gorm:"column:blur_adult_content" json:"blurAdultContent"`
}

type LibrarySettings struct {
	LibraryPath                     string `gorm:"column:library_path" json:"libraryPath"`
	DisableUpdateCheck              bool   `gorm:"column:disable_update_check" json:"disableUpdateCheck"`
	TorrentProvider                 string `gorm:"column:torrent_provider" json:"torrentProvider"`
	AutoScan                        bool   `gorm:"column:auto_scan" json:"autoScan"`
	EnableOnlinestream              bool   `gorm:"column:enable_onlinestream" json:"enableOnlinestream"`
	IncludeOnlineStreamingInLibrary bool   `gorm:"column:include_online_streaming_in_library" json:"includeOnlineStreamingInLibrary"`
	DisableAnimeCardTrailers        bool   `gorm:"column:disable_anime_card_trailers" json:"disableAnimeCardTrailers"`
	EnableManga                     bool   `gorm:"column:enable_manga" json:"enableManga"`
	DOHProvider                     string `gorm:"column:doh_provider" json:"dohProvider"`
	OpenTorrentClientOnStart        bool   `gorm:"column:open_torrent_client_on_start" json:"openTorrentClientOnStart"`
	OpenWebURLOnStart               bool   `gorm:"column:open_web_url_on_start" json:"openWebURLOnStart"`
	RefreshLibraryOnStart           bool   `gorm:"column:refresh_library_on_start" json:"refreshLibraryOnStart"`
	// v2.2+
	EnableWatchContinuity    bool         `gorm:"column:enable_watch_continuity" json:"enableWatchContinuity"`
	LibraryPaths             LibraryPaths `gorm:"column:library_paths;type:text" json:"libraryPaths"`
	// v2.6+
	ScannerMatchingThreshold float64 `gorm:"column:scanner_matching_threshold" json:"scannerMatchingThreshold"`
	ScannerMatchingAlgorithm string  `gorm:"column:scanner_matching_algorithm" json:"scannerMatchingAlgorithm"`
	// v2.9+
	AutoSyncToLocalAccount      bool `gorm:"column:auto_sync_to_local_account" json:"autoSyncToLocalAccount"`
}

func (o *LibrarySettings) GetLibraryPaths() (ret []string) {
	ret = make([]string, len(o.LibraryPaths)+1)
	ret[0] = o.LibraryPath
	if len(o.LibraryPaths) > 0 {
		copy(ret[1:], o.LibraryPaths)
	}
	return
}

type LibraryPaths []string

func (o *LibraryPaths) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return errors.New("src value cannot cast to string")
	}
	*o = strings.Split(str, ",")
	return nil
}
func (o LibraryPaths) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	return strings.Join(o, ","), nil
}


type IntSlice []int

func (o *IntSlice) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return errors.New("src value cannot cast to string")
	}
	ids := strings.Split(str, ",")
	*o = make(IntSlice, len(ids))
	for i, id := range ids {
		(*o)[i], _ = strconv.Atoi(id)
	}
	return nil
}
func (o IntSlice) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	strs := make([]string, len(o))
	for i, id := range o {
		strs[i] = strconv.Itoa(id)
	}
	return strings.Join(strs, ","), nil
}

type MangaSettings struct {
	DefaultProvider      string `gorm:"column:default_manga_provider" json:"defaultMangaProvider"`
	AutoUpdateProgress   bool   `gorm:"column:manga_auto_update_progress" json:"mangaAutoUpdateProgress"`
	LocalSourceDirectory string `gorm:"column:manga_local_source_directory" json:"mangaLocalSourceDirectory"`
}

type MediaPlayerSettings struct {
	Default     string `gorm:"column:default_player" json:"defaultPlayer"` // "vlc" or "mpc-hc"
	Host        string `gorm:"column:player_host" json:"host"`
	VlcUsername string `gorm:"column:vlc_username" json:"vlcUsername"`
	VlcPassword string `gorm:"column:vlc_password" json:"vlcPassword"`
	VlcPort     int    `gorm:"column:vlc_port" json:"vlcPort"`
	VlcPath     string `gorm:"column:vlc_path" json:"vlcPath"`
	MpcPort     int    `gorm:"column:mpc_port" json:"mpcPort"`
	MpcPath     string `gorm:"column:mpc_path" json:"mpcPath"`
	MpvSocket   string `gorm:"column:mpv_socket" json:"mpvSocket"`
	MpvPath     string `gorm:"column:mpv_path" json:"mpvPath"`
	MpvArgs     string `gorm:"column:mpv_args" json:"mpvArgs"`
	IinaSocket  string `gorm:"column:iina_socket" json:"iinaSocket"`
	IinaPath    string `gorm:"column:iina_path" json:"iinaPath"`
	IinaArgs    string `gorm:"column:iina_args" json:"iinaArgs"`
}

type TorrentSettings struct {
	Default              string `gorm:"column:default_torrent_client" json:"defaultTorrentClient"`
	QBittorrentPath      string `gorm:"column:qbittorrent_path" json:"qbittorrentPath"`
	QBittorrentHost      string `gorm:"column:qbittorrent_host" json:"qbittorrentHost"`
	QBittorrentPort      int    `gorm:"column:qbittorrent_port" json:"qbittorrentPort"`
	QBittorrentUsername  string `gorm:"column:qbittorrent_username" json:"qbittorrentUsername"`
	QBittorrentPassword  string `gorm:"column:qbittorrent_password" json:"qbittorrentPassword"`
	QBittorrentTags      string `gorm:"column:qbittorrent_tags" json:"qbittorrentTags"`
	TransmissionPath     string `gorm:"column:transmission_path" json:"transmissionPath"`
	TransmissionHost     string `gorm:"column:transmission_host" json:"transmissionHost"`
	TransmissionPort     int    `gorm:"column:transmission_port" json:"transmissionPort"`
	TransmissionUsername string `gorm:"column:transmission_username" json:"transmissionUsername"`
	TransmissionPassword string `gorm:"column:transmission_password" json:"transmissionPassword"`
	// v2.1+
	ShowActiveTorrentCount bool `gorm:"column:show_active_torrent_count" json:"showActiveTorrentCount"`
	// v2.2+
	HideTorrentList bool `gorm:"column:hide_torrent_list" json:"hideTorrentList"`
}

type ListSyncSettings struct {
	Automatic bool   `gorm:"column:automatic_sync" json:"automatic"`
	Origin    string `gorm:"column:sync_origin" json:"origin"`
}

type DiscordSettings struct {
	EnableRichPresence                      bool `gorm:"column:enable_rich_presence" json:"enableRichPresence"`
	EnableAnimeRichPresence                 bool `gorm:"column:enable_anime_rich_presence" json:"enableAnimeRichPresence"`
	EnableMangaRichPresence                 bool `gorm:"column:enable_manga_rich_presence" json:"enableMangaRichPresence"`
	RichPresenceHideSeanimeRepositoryButton bool `gorm:"column:rich_presence_hide_seanime_repository_button" json:"richPresenceHideSeanimeRepositoryButton"`
	RichPresenceShowAniListMediaButton      bool `gorm:"column:rich_presence_show_anilist_media_button" json:"richPresenceShowAniListMediaButton"`
	RichPresenceShowAniListProfileButton    bool `gorm:"column:rich_presence_show_anilist_profile_button" json:"richPresenceShowAniListProfileButton"`
	RichPresenceUseMediaTitleStatus         bool `gorm:"column:rich_presence_use_media_title_status;default:true" json:"richPresenceUseMediaTitleStatus"`
}

type NotificationSettings struct {
	DisableNotifications               bool `gorm:"column:disable_notifications" json:"disableNotifications"`
	DisableAutoDownloaderNotifications bool `gorm:"column:disable_auto_downloader_notifications" json:"disableAutoDownloaderNotifications"`
	DisableAutoScannerNotifications    bool `gorm:"column:disable_auto_scanner_notifications" json:"disableAutoScannerNotifications"`
}

// +---------------------+
// |         MAL         |
// +---------------------+

type Mal struct {
	BaseModel
	Username       string    `gorm:"column:username" json:"username"`
	AccessToken    string    `gorm:"column:access_token" json:"accessToken"`
	RefreshToken   string    `gorm:"column:refresh_token" json:"refreshToken"`
	TokenExpiresAt time.Time `gorm:"column:token_expires_at" json:"tokenExpiresAt"`
}

// +---------------------+
// |    Scan Summary     |
// +---------------------+

type ScanSummary struct {
	BaseModel
	Value []byte `gorm:"column:value" json:"value"`
}

// +---------------------+
// |   Auto downloader   |
// +---------------------+

type AutoDownloaderRule struct {
	BaseModel
	Value []byte `gorm:"column:value" json:"value"`
}

type AutoDownloaderItem struct {
	BaseModel
	RuleID      uint   `gorm:"column:rule_id" json:"ruleId"`
	MediaID     int    `gorm:"column:media_id" json:"mediaId"`
	Episode     int    `gorm:"column:episode" json:"episode"`
	Link        string `gorm:"column:link" json:"link"`
	Hash        string `gorm:"column:hash" json:"hash"`
	Magnet      string `gorm:"column:magnet" json:"magnet"`
	TorrentName string `gorm:"column:torrent_name" json:"torrentName"`
	Downloaded  bool   `gorm:"column:downloaded" json:"downloaded"`
}

type AutoDownloaderSettings struct {
	Provider              string `gorm:"column:auto_downloader_provider" json:"provider"`
	Interval              int    `gorm:"column:auto_downloader_interval" json:"interval"`
	Enabled               bool   `gorm:"column:auto_downloader_enabled" json:"enabled"`
	DownloadAutomatically bool   `gorm:"column:auto_downloader_download_automatically" json:"downloadAutomatically"`
	EnableEnhancedQueries bool   `gorm:"column:auto_downloader_enable_enhanced_queries" json:"enableEnhancedQueries"`
	EnableSeasonCheck     bool   `gorm:"column:auto_downloader_enable_season_check" json:"enableSeasonCheck"`
	UseDebrid             bool   `gorm:"column:auto_downloader_use_debrid" json:"useDebrid"`
}

// +---------------------+
// |     Media Entry     |
// +---------------------+

type SilencedMediaEntry struct {
	BaseModel
}

// +---------------------+
// |        Theme        |
// +---------------------+

type Theme struct {
	BaseModel
	UserID uint `gorm:"column:user_id;index" json:"userId"` // Foreign key to User
	// Main
	EnableColorSettings              bool   `gorm:"column:enable_color_settings" json:"enableColorSettings"`
	BackgroundColor                  string `gorm:"column:background_color" json:"backgroundColor"`
	AccentColor                      string `gorm:"column:accent_color" json:"accentColor"`
	SidebarBackgroundColor           string `gorm:"column:sidebar_background_color" json:"sidebarBackgroundColor"`  // DEPRECATED
	AnimeEntryScreenLayout           string `gorm:"column:anime_entry_screen_layout" json:"animeEntryScreenLayout"` // DEPRECATED
	ExpandSidebarOnHover             bool   `gorm:"column:expand_sidebar_on_hover" json:"expandSidebarOnHover"`
	HideTopNavbar                    bool   `gorm:"column:hide_top_navbar" json:"hideTopNavbar"`
	EnableMediaCardBlurredBackground bool   `gorm:"column:enable_media_card_blurred_background" json:"enableMediaCardBlurredBackground"`
	// Note: These are named "libraryScreen" but are used on all pages
	LibraryScreenCustomBackgroundImage   string `gorm:"column:library_screen_custom_background_image" json:"libraryScreenCustomBackgroundImage"`
	LibraryScreenCustomBackgroundOpacity int    `gorm:"column:library_screen_custom_background_opacity" json:"libraryScreenCustomBackgroundOpacity"`
	// Anime
	SmallerEpisodeCarouselSize bool `gorm:"column:smaller_episode_carousel_size" json:"smallerEpisodeCarouselSize"`
	// Library Screen (Anime & Manga)
	// LibraryScreenBannerType: "dynamic", "custom"
	LibraryScreenBannerType           string `gorm:"column:library_screen_banner_type" json:"libraryScreenBannerType"`
	LibraryScreenCustomBannerImage    string `gorm:"column:library_screen_custom_banner_image" json:"libraryScreenCustomBannerImage"`
	LibraryScreenCustomBannerPosition string `gorm:"column:library_screen_custom_banner_position" json:"libraryScreenCustomBannerPosition"`
	LibraryScreenCustomBannerOpacity  int    `gorm:"column:library_screen_custom_banner_opacity" json:"libraryScreenCustomBannerOpacity"`
	DisableLibraryScreenGenreSelector bool   `gorm:"column:disable_library_screen_genre_selector" json:"disableLibraryScreenGenreSelector"`

	LibraryScreenCustomBackgroundBlur string `gorm:"column:library_screen_custom_background_blur" json:"libraryScreenCustomBackgroundBlur"`
	EnableMediaPageBlurredBackground  bool   `gorm:"column:enable_media_page_blurred_background" json:"enableMediaPageBlurredBackground"`
	DisableSidebarTransparency        bool   `gorm:"column:disable_sidebar_transparency" json:"disableSidebarTransparency"`
	UseLegacyEpisodeCard              bool   `gorm:"column:use_legacy_episode_card" json:"useLegacyEpisodeCard"` // DEPRECATED
	DisableCarouselAutoScroll         bool   `gorm:"column:disable_carousel_auto_scroll" json:"disableCarouselAutoScroll"`

	// v2.6+
	MediaPageBannerType        string `gorm:"column:media_page_banner_type" json:"mediaPageBannerType"`
	MediaPageBannerSize        string `gorm:"column:media_page_banner_size" json:"mediaPageBannerSize"`
	MediaPageBannerInfoBoxSize string `gorm:"column:media_page_banner_info_box_size" json:"mediaPageBannerInfoBoxSize"`

	// v2.7+
	ShowEpisodeCardAnimeInfo             bool   `gorm:"column:show_episode_card_anime_info" json:"showEpisodeCardAnimeInfo"`
	ContinueWatchingDefaultSorting       string `gorm:"column:continue_watching_default_sorting" json:"continueWatchingDefaultSorting"`
	AnimeLibraryCollectionDefaultSorting string `gorm:"column:anime_library_collection_default_sorting" json:"animeLibraryCollectionDefaultSorting"`
	MangaLibraryCollectionDefaultSorting string `gorm:"column:manga_library_collection_default_sorting" json:"mangaLibraryCollectionDefaultSorting"`
	ShowAnimeUnwatchedCount              bool   `gorm:"column:show_anime_unwatched_count" json:"showAnimeUnwatchedCount"`
	ShowMangaUnreadCount                 bool   `gorm:"column:show_manga_unread_count" json:"showMangaUnreadCount"`

	// v2.8+
	HideEpisodeCardDescription        bool   `gorm:"column:hide_episode_card_description" json:"hideEpisodeCardDescription"`
	HideDownloadedEpisodeCardFilename bool   `gorm:"column:hide_downloaded_episode_card_filename" json:"hideDownloadedEpisodeCardFilename"`
	CustomCSS                         string `gorm:"column:custom_css" json:"customCSS"`
	MobileCustomCSS                   string `gorm:"column:mobile_custom_css" json:"mobileCustomCSS"`

	// v2.9+
	UnpinnedMenuItems StringSlice `gorm:"column:unpinned_menu_items;type:text" json:"unpinnedMenuItems"`
}

// +---------------------+
// |      Playlist       |
// +---------------------+

type PlaylistEntry struct {
	BaseModel
	UserID uint   `gorm:"column:user_id;index" json:"userId"` // Foreign key to User
	Name   string `gorm:"column:name" json:"name"`
	Value  []byte `gorm:"column:value" json:"value"`
}

// +------------------------+
// | Chapter Download Queue |
// +------------------------+

type ChapterDownloadQueueItem struct {
	BaseModel
	UserID        uint   `gorm:"column:user_id;index" json:"userId"` // Foreign key to User
	Provider      string `gorm:"column:provider" json:"provider"`
	MediaID       int    `gorm:"column:media_id" json:"mediaId"`
	ChapterID     string `gorm:"column:chapter_id" json:"chapterId"`
	ChapterNumber string `gorm:"column:chapter_number" json:"chapterNumber"`
	PageData      []byte `gorm:"column:page_data" json:"pageData"` // Contains map of page index to page details
	Status        string `gorm:"column:status" json:"status"`
}

// +---------------------+
// |     MediaStream     |
// +---------------------+

type MediastreamSettings struct {
	BaseModel
	// DEVNOTE: Should really be "Enabled"
	TranscodeEnabled              bool   `gorm:"column:transcode_enabled" json:"transcodeEnabled"`
	TranscodeHwAccel              string `gorm:"column:transcode_hw_accel" json:"transcodeHwAccel"`
	TranscodeThreads              int    `gorm:"column:transcode_threads" json:"transcodeThreads"`
	TranscodePreset               string `gorm:"column:transcode_preset" json:"transcodePreset"`
	DisableAutoSwitchToDirectPlay bool   `gorm:"column:disable_auto_switch_to_direct_play" json:"disableAutoSwitchToDirectPlay"`
	DirectPlayOnly                bool   `gorm:"column:direct_play_only" json:"directPlayOnly"`
	PreTranscodeEnabled           bool   `gorm:"column:pre_transcode_enabled" json:"preTranscodeEnabled"`
	PreTranscodeLibraryDir        string `gorm:"column:pre_transcode_library_dir" json:"preTranscodeLibraryDir"`
	FfmpegPath                    string `gorm:"column:ffmpeg_path" json:"ffmpegPath"`
	FfprobePath                   string `gorm:"column:ffprobe_path" json:"ffprobePath"`
	// v2.2+
	TranscodeHwAccelCustomSettings string `gorm:"column:transcode_hw_accel_custom_settings" json:"transcodeHwAccelCustomSettings"`

	//TranscodeTempDir              string `gorm:"column:transcode_temp_dir" json:"transcodeTempDir"` // DEPRECATED
}

// +---------------------+
// |    TorrentStream    |
// +---------------------+

type TorrentstreamSettings struct {
	BaseModel
	Enabled             bool   `gorm:"column:enabled" json:"enabled"`
	AutoSelect          bool   `gorm:"column:auto_select" json:"autoSelect"`
	PreferredResolution string `gorm:"column:preferred_resolution" json:"preferredResolution"`
	DisableIPV6         bool   `gorm:"column:disable_ipv6" json:"disableIPV6"`
	DownloadDir         string `gorm:"column:download_dir" json:"downloadDir"`
	AddToLibrary        bool   `gorm:"column:add_to_library" json:"addToLibrary"`
	TorrentClientHost   string `gorm:"column:torrent_client_host" json:"torrentClientHost"`
	TorrentClientPort   int    `gorm:"column:torrent_client_port" json:"torrentClientPort"`
	StreamingServerHost string `gorm:"column:streaming_server_host" json:"streamingServerHost"`
	StreamingServerPort int    `gorm:"column:streaming_server_port" json:"streamingServerPort"`
	//FallbackToTorrentStreamingView bool   `gorm:"column:fallback_to_torrent_streaming_view" json:"fallbackToTorrentStreamingView"` // DEPRECATED
	IncludeInLibrary bool `gorm:"column:include_in_library" json:"includeInLibrary"`
	// v2.6+
	StreamUrlAddress string `gorm:"column:stream_url_address" json:"streamUrlAddress"`
	// v2.7+
	SlowSeeding bool `gorm:"column:slow_seeding" json:"slowSeeding"`
}

type TorrentstreamHistory struct {
	BaseModel
	MediaId int    `gorm:"column:media_id" json:"mediaId"`
	Torrent []byte `gorm:"column:torrent" json:"torrent"`
}

// +---------------------+
// |        Filler       |
// +---------------------+

type MediaFiller struct {
	BaseModel
	Provider      string    `gorm:"column:provider" json:"provider"`
	Slug          string    `gorm:"column:slug" json:"slug"`
	MediaID       int       `gorm:"column:media_id" json:"mediaId"`
	LastFetchedAt time.Time `gorm:"column:last_fetched_at" json:"lastFetchedAt"`
	Data          []byte    `gorm:"column:data" json:"data"`
}

// +---------------------+
// |        Manga        |
// +---------------------+

type MangaMapping struct {
	BaseModel
	Provider string `gorm:"column:provider" json:"provider"`
	MediaID  int    `gorm:"column:media_id" json:"mediaId"`
	MangaID  string `gorm:"column:manga_id" json:"mangaId"` // ID from search result, used to fetch chapters
}

type MangaChapterContainer struct {
	BaseModel
	Provider  string `gorm:"column:provider" json:"provider"`
	MediaID   int    `gorm:"column:media_id" json:"mediaId"`
	ChapterID string `gorm:"column:chapter_id" json:"chapterId"`
	Data      []byte `gorm:"column:data" json:"data"`
}

// +---------------------+
// |  Online streaming   |
// +---------------------+

type OnlinestreamMapping struct {
	BaseModel
	Provider string `gorm:"column:provider" json:"provider"`
	MediaID  int    `gorm:"column:media_id" json:"mediaId"`
	AnimeID  string `gorm:"column:anime_id" json:"anime_id"` // ID from search result, used to fetch episodes
}

// +---------------------+
// |       Debrid        |
// +---------------------+

type DebridSettings struct {
	BaseModel
	Enabled  bool   `gorm:"column:enabled" json:"enabled"`
	Provider string `gorm:"column:provider" json:"provider"`
	ApiKey   string `gorm:"column:api_key" json:"apiKey"`
	//FallbackToDebridStreamingView bool   `gorm:"column:fallback_to_debrid_streaming_view" json:"fallbackToDebridStreamingView"` // DEPRECATED
	IncludeDebridStreamInLibrary bool   `gorm:"column:include_debrid_stream_in_library" json:"includeDebridStreamInLibrary"`
	StreamAutoSelect             bool   `gorm:"column:stream_auto_select" json:"streamAutoSelect"`
	StreamPreferredResolution    string `gorm:"column:stream_preferred_resolution" json:"streamPreferredResolution"`
}

type DebridTorrentItem struct {
	BaseModel
	TorrentItemID string `gorm:"column:torrent_item_id" json:"torrentItemId"`
	Destination   string `gorm:"column:destination" json:"destination"`
	Provider      string `gorm:"column:provider" json:"provider"`
	MediaId       int    `gorm:"column:media_id" json:"mediaId"`
}

// +---------------------+
// |       Plugin        |
// +---------------------+

type PluginData struct {
	BaseModel
	PluginID string `gorm:"column:plugin_id;index" json:"pluginId"`
	Data     []byte `gorm:"column:data" json:"data"`
}

///////////////////////////////////////////////////////////////////////////

type StringSlice []string

func (o *StringSlice) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return errors.New("src value cannot cast to string")
	}
	*o = strings.Split(str, ",")
	return nil
}
func (o StringSlice) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	return strings.Join(o, ","), nil
}

// +---------------------+
// |   User Sync System  |
// +---------------------+

// UserLibraryEntry tracks which anime each user has in their library
type UserLibraryEntry struct {
	BaseModel
	UserID    uint      `gorm:"uniqueIndex:idx_user_media" json:"userId"`
	MediaID   int       `gorm:"uniqueIndex:idx_user_media" json:"mediaId"`
	AddedAt   time.Time `json:"addedAt"`
	IsActive  bool      `gorm:"default:true" json:"isActive"` // For filtering inactive entries
}

// UserEpisodeProgress tracks per-user episode watch progress with resume functionality
type UserEpisodeProgress struct {
	BaseModel
	UserID             uint      `gorm:"uniqueIndex:idx_user_episode" json:"userId"`
	MediaID            int       `gorm:"uniqueIndex:idx_user_episode" json:"mediaId"`
	EpisodeNumber      int       `gorm:"uniqueIndex:idx_user_episode" json:"episodeNumber"`
	AniDBEpisode       string    `json:"aniDbEpisode"`
	
	// Progress tracking
	WatchTimeSeconds   int       `json:"watchTimeSeconds"`   // Current watch position
	DurationSeconds    int       `json:"durationSeconds"`    // Total episode duration
	CompletionPercent  float64   `json:"completionPercent"`  // Calculated percentage
	IsCompleted        bool      `json:"isCompleted"`        // Marked as watched
	LastWatchedAt      time.Time `json:"lastWatchedAt"`
	
	// File association
	LocalFilePath      string    `json:"localFilePath"`      // Which file was being watched
	LocalFileHash      string    `json:"localFileHash"`      // For file integrity checking
	
	// Playback context
	MediaPlayerUsed    string    `json:"mediaPlayerUsed"`    // "mpv", "vlc", etc.
	DeviceInfo         string    `json:"deviceInfo"`         // Optional device identifier
}

// UserActivePlayback tracks active playback sessions for real-time sync
type UserActivePlayback struct {
	BaseModel
	UserID            uint      `gorm:"unique" json:"userId"`
	MediaID           int       `json:"mediaId"`
	EpisodeNumber     int       `json:"episodeNumber"`
	CurrentTimeSeconds int      `json:"currentTimeSeconds"`
	PlaybackState     string    `json:"playbackState"`     // "playing", "paused", "stopped"
	LastUpdateAt      time.Time `json:"lastUpdateAt"`
	SessionID         string    `json:"sessionId"`         // WebSocket session identifier
}

// UserMediaSubscription tracks which media each user wants real-time updates for
type UserMediaSubscription struct {
	BaseModel
	UserID    uint      `gorm:"uniqueIndex:idx_user_media_sub" json:"userId"`
	MediaID   int       `gorm:"uniqueIndex:idx_user_media_sub" json:"mediaId"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

// +---------------------+
// |   Global Mapping    |
// +---------------------+

// GlobalAnimeFileMapping stores global AniList ID to local file path mappings
type GlobalAnimeFileMapping struct {
	BaseModel
	AniListID     int       `gorm:"column:anilist_id;index" json:"anilistId"`
	LocalFilePath string    `gorm:"column:local_file_path;unique" json:"localFilePath"`
	Title         string    `gorm:"column:title" json:"title"`                      // Primary title for display
	RomajiTitle   string    `gorm:"column:romaji_title" json:"romajiTitle"`         // Romaji title variant
	EnglishTitle  string    `gorm:"column:english_title" json:"englishTitle"`       // English title variant
	Synonyms      string    `gorm:"column:synonyms;type:text" json:"synonyms"`      // JSON array of synonyms
	Year          int       `gorm:"column:year" json:"year"`
	EpisodeNumber int       `gorm:"column:episode_number" json:"episodeNumber"`
	FileSize      int64     `gorm:"column:file_size" json:"fileSize"`
	LastScanned   time.Time `gorm:"column:last_scanned" json:"lastScanned"`
}

// UserAnimeSubscription tracks which users have which anime in their AniList collections
// This enables multi-token API strategy and user-specific catalog features
type UserAnimeSubscription struct {
	BaseModel
	UserID       uint      `gorm:"column:user_id;index" json:"userId"`                               // Foreign key to User
	AniListID    int       `gorm:"column:anilist_id;index" json:"anilistId"`                        // Which anime they have
	LastVerified time.Time `gorm:"column:last_verified" json:"lastVerified"`                        // When we last confirmed they have it
	TokenStatus  string    `gorm:"column:token_status;default:'active'" json:"tokenStatus"`         // active/failed/removed
	AddedAt      time.Time `gorm:"column:added_at;default:CURRENT_TIMESTAMP" json:"addedAt"`        // When first detected
}

// Index for efficient queries: unique user-anime combinations
func (UserAnimeSubscription) TableName() string {
	return "user_anime_subscriptions"
}

// UnmappedFile stores files that couldn't be automatically matched to AniList entries
type UnmappedFile struct {
	BaseModel
	LocalFilePath   string     `gorm:"column:local_file_path;unique" json:"localFilePath"`
	ParsedTitle     string     `gorm:"column:parsed_title" json:"parsedTitle"`       // Parsed anime title from filename
	DetectedTitle   string     `gorm:"column:detected_title" json:"detectedTitle"`   // Alternative detected title
	FileSize        int64      `gorm:"column:file_size" json:"fileSize"`
	LastDetected    time.Time  `gorm:"column:last_detected" json:"lastDetected"`
	Status          string     `gorm:"column:status;default:'PENDING'" json:"status"` // "PENDING", "IGNORED", "ASSIGNED"
	IgnoredByUserID uint       `gorm:"column:ignored_by_user_id" json:"ignoredByUserId"`
	IgnoredAt       *time.Time `gorm:"column:ignored_at" json:"ignoredAt"`
}

// UserLibrarySubscription tracks which anime each user has subscribed to in their library
type UserLibrarySubscription struct {
	BaseModel
	UserID    uint      `gorm:"column:user_id;index:idx_user_subscription" json:"userId"`
	AniListID int       `gorm:"column:anilist_id;index:idx_anilist_subscription" json:"anilistId"`
	AddedAt   time.Time `gorm:"column:added_at" json:"addedAt"`
}

// UserProgressSyncItem queues user progress updates for batch sync to AniList
type UserProgressSyncItem struct {
	BaseModel
	UserID         uint      `gorm:"column:user_id;index:idx_user_sync" json:"userId"`
	AniListID      int       `gorm:"column:anilist_id" json:"anilistId"`
	EpisodeNumber  int       `gorm:"column:episode_number" json:"episodeNumber"`
	Status         string    `gorm:"column:status" json:"status"`                     // "CURRENT", "COMPLETED", etc.
	Score          int       `gorm:"column:score" json:"score"`
	Progress       int       `gorm:"column:progress" json:"progress"`
	IsCompleted    bool      `gorm:"column:is_completed" json:"isCompleted"`
	LastUpdated    time.Time `gorm:"column:last_updated" json:"lastUpdated"`
	SyncStatus     string    `gorm:"column:sync_status;default:'PENDING'" json:"syncStatus"` // "PENDING", "SYNCED", "FAILED"
	RetryCount     int       `gorm:"column:retry_count;default:0" json:"retryCount"`
}
