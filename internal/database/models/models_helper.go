package models

func (s *Settings) GetMediaPlayer() *MediaPlayerSettings {
	if s == nil || s.MediaPlayer == nil {
		return &MediaPlayerSettings{}
	}
	return s.MediaPlayer
}


func (s *Settings) GetAnilist() *AnilistSettings {
	if s == nil || s.Anilist == nil {
		return &AnilistSettings{}
	}
	return s.Anilist
}

func (s *Settings) GetManga() *MangaSettings {
	if s == nil || s.Manga == nil {
		return &MangaSettings{}
	}
	return s.Manga
}


func (s *Settings) GetListSync() *ListSyncSettings {
	if s == nil || s.ListSync == nil {
		return &ListSyncSettings{}
	}
	return s.ListSync
}


func (s *Settings) GetDiscord() *DiscordSettings {
	if s == nil || s.Discord == nil {
		return &DiscordSettings{}
	}
	return s.Discord
}




///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (s *Settings) GetSensitiveValues() []string {
	if s == nil {
		return []string{}
	}
	return []string{
		s.GetMediaPlayer().VlcPassword,
	}
}

// GlobalSettings helper methods
func (g *GlobalSettings) GetTorrent() *TorrentSettings {
	if g == nil || g.Torrent == nil {
		return &TorrentSettings{}
	}
	return g.Torrent
}

func (g *GlobalSettings) GetLibrary() *LibrarySettings {
	if g == nil || g.Library == nil {
		return &LibrarySettings{}
	}
	return g.Library
}

func (g *GlobalSettings) GetAutoDownloader() *AutoDownloaderSettings {
	if g == nil || g.AutoDownloader == nil {
		return &AutoDownloaderSettings{}
	}
	return g.AutoDownloader
}

func (g *GlobalSettings) GetSensitiveValues() []string {
	if g == nil {
		return []string{}
	}
	return []string{
		g.GetTorrent().QBittorrentPassword,
		g.GetTorrent().TransmissionPassword,
	}
}

// Debrid feature removed: (*DebridSettings).GetSensitiveValues deleted
