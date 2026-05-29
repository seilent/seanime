package db

import (
	"seanime/internal/database/models"
	"strings"
)

func (db *Database) UpsertPendingDownloadIntent(hash string, mediaId int) error {
	hash = strings.ToLower(strings.TrimSpace(hash))
	if hash == "" {
		return nil
	}
	intent := &models.PendingDownloadIntent{Hash: hash, MediaID: mediaId}
	return db.gormdb.Where("hash = ?", hash).FirstOrCreate(intent).Error
}

func (db *Database) GetIncompletePendingDownloadIntents() ([]*models.PendingDownloadIntent, error) {
	var res []*models.PendingDownloadIntent
	err := db.gormdb.Where("completed = ?", false).Find(&res).Error
	return res, err
}

func (db *Database) MarkPendingDownloadIntentCompleted(hash string) error {
	return db.gormdb.Model(&models.PendingDownloadIntent{}).Where("hash = ?", hash).Update("completed", true).Error
}

func (db *Database) GetIncompleteIntentHashes() (map[string]struct{}, error) {
	var hashes []string
	err := db.gormdb.Model(&models.PendingDownloadIntent{}).Where("completed = ?", false).Pluck("hash", &hashes).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]struct{}, len(hashes))
	for _, h := range hashes {
		m[h] = struct{}{}
	}
	return m, nil
}
