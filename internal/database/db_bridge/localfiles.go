package db_bridge

import (
	"fmt"

	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/library/anime"
)

// GetLocalFilesForUser returns local files from global mappings. userID is unused (kept for caller compatibility).
func GetLocalFilesForUser(database *db.Database, userID uint) ([]*anime.LocalFile, uint, error) {
	return GetLocalFilesFromGlobalMappings(database)
}

// GetLocalFilesFromGlobalMappings retrieves local files from the global anime file mappings table.
func GetLocalFilesFromGlobalMappings(database *db.Database) ([]*anime.LocalFile, uint, error) {
	var mappings []models.GlobalAnimeFileMapping
	err := database.Gorm().Find(&mappings).Error
	if err != nil {
		return nil, 0, err
	}

	localFiles := make([]*anime.LocalFile, 0, len(mappings))

	for _, mapping := range mappings {
		localFile := anime.NewLocalFile(mapping.LocalFilePath, "/")
		localFile.MediaId = mapping.AniListID

		// Determine file type from row, default to Main
		fileType := anime.LocalFileTypeMain
		if mapping.FileType != "" {
			fileType = anime.LocalFileType(mapping.FileType)
		}

		localFile.Metadata = &anime.LocalFileMetadata{
			Episode:      mapping.EpisodeNumber,
			AniDBEpisode: fmt.Sprintf("%d", mapping.EpisodeNumber),
			Type:         fileType,
		}

		localFile.Locked = false
		localFile.Ignored = mapping.Ignored

		localFiles = append(localFiles, localFile)
	}

	database.Logger.Debug().
		Int("count", len(localFiles)).
		Msg("db: Local files retrieved from global mappings table")

	return localFiles, 1, nil
}

// GetLocalFilesByMediaId retrieves all global file mappings for a specific media ID
func GetLocalFilesByMediaId(database *db.Database, mediaId int) ([]*models.GlobalAnimeFileMapping, error) {
	var mappings []*models.GlobalAnimeFileMapping
	err := database.Gorm().Where("anilist_id = ?", mediaId).Find(&mappings).Error
	return mappings, err
}
