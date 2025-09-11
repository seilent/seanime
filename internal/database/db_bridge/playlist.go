package db_bridge

import (
	"github.com/goccy/go-json"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/library/anime"
)

// GetPlaylists returns playlists for a specific user
func GetPlaylists(db *db.Database, userID uint) ([]*anime.Playlist, error) {
	var res []*models.PlaylistEntry
	err := db.Gorm().Where("user_id = ?", userID).Find(&res).Error
	if err != nil {
		return nil, err
	}

	playlists := make([]*anime.Playlist, 0)
	for _, p := range res {
		var localFiles []*anime.LocalFile
		if err := json.Unmarshal(p.Value, &localFiles); err == nil {
			playlist := anime.NewPlaylist(p.Name)
			playlist.SetLocalFiles(localFiles)
			playlist.DbId = p.ID
			playlists = append(playlists, playlist)
		}
	}
	return playlists, nil
}

func SavePlaylist(db *db.Database, userID uint, playlist *anime.Playlist) error {
	data, err := json.Marshal(playlist.LocalFiles)
	if err != nil {
		return err
	}
	playlistEntry := &models.PlaylistEntry{
		UserID: userID,
		Name:   playlist.Name,
		Value:  data,
	}

	return db.Gorm().Save(playlistEntry).Error
}

func DeletePlaylist(db *db.Database, userID uint, id uint) error {
	return db.Gorm().Where("id = ? AND user_id = ?", id, userID).Delete(&models.PlaylistEntry{}).Error
}

func UpdatePlaylist(db *db.Database, userID uint, playlist *anime.Playlist) error {
	data, err := json.Marshal(playlist.LocalFiles)
	if err != nil {
		return err
	}

	// Get the playlist entry (ensure it belongs to the user)
	playlistEntry := &models.PlaylistEntry{}
	if err := db.Gorm().Where("id = ? AND user_id = ?", playlist.DbId, userID).First(playlistEntry).Error; err != nil {
		return err
	}

	// Update the playlist entry
	playlistEntry.Name = playlist.Name
	playlistEntry.Value = data

	return db.Gorm().Save(playlistEntry).Error
}

func GetPlaylist(db *db.Database, userID uint, id uint) (*anime.Playlist, error) {
	playlistEntry := &models.PlaylistEntry{}
	if err := db.Gorm().Where("id = ? AND user_id = ?", id, userID).First(playlistEntry).Error; err != nil {
		return nil, err
	}

	var localFiles []*anime.LocalFile
	if err := json.Unmarshal(playlistEntry.Value, &localFiles); err != nil {
		return nil, err
	}

	playlist := anime.NewPlaylist(playlistEntry.Name)
	playlist.SetLocalFiles(localFiles)
	playlist.DbId = playlistEntry.ID

	return playlist, nil
}
