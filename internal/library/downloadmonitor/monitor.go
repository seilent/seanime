package downloadmonitor

import (
	"os"
	"path/filepath"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/torrent_clients/torrent_client"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/5rahim/habari"
	"github.com/rs/zerolog"
)

var videoExts = map[string]struct{}{
	".mkv": {}, ".mp4": {}, ".avi": {}, ".webm": {}, ".m4v": {}, ".ts": {},
}

type Monitor struct {
	db     *db.Database
	repo   *torrent_client.Repository
	logger *zerolog.Logger
}

func New(db *db.Database, repo *torrent_client.Repository, logger *zerolog.Logger) *Monitor {
	return &Monitor{db: db, repo: repo, logger: logger}
}

var (
	monitorMu  sync.Mutex
	activeStop chan struct{}
)

// Start launches the monitor loop, stopping any previously started monitor first
// so that module refreshes (which reconstruct the monitor) don't leak goroutines.
func (m *Monitor) Start() {
	monitorMu.Lock()
	if activeStop != nil {
		close(activeStop)
	}
	stop := make(chan struct{})
	activeStop = stop
	monitorMu.Unlock()
	go m.run(stop)
}

func (m *Monitor) run(stop chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.tick()
		case <-stop:
			return
		}
	}
}

func (m *Monitor) tick() {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error().Msgf("downloadmonitor: panic: %v", r)
		}
	}()

	intents, err := m.db.GetIncompletePendingDownloadIntents()
	if err != nil || len(intents) == 0 {
		return
	}

	torrents, err := m.repo.GetList()
	if err != nil {
		return
	}
	byHash := make(map[string]*torrent_client.Torrent, len(torrents))
	for _, t := range torrents {
		byHash[strings.ToLower(t.Hash)] = t
	}

	for _, intent := range intents {
		t, ok := byHash[intent.Hash]
		if !ok {
			continue
		}
		if t.Progress < 1.0 && t.Status != torrent_client.TorrentStatusSeeding && t.Status != torrent_client.TorrentStatusStopped {
			continue
		}

		// Torrent is complete — walk for video files
		videos := collectVideos(t.ContentPath)
		for _, path := range videos {
			ep := parseEpisode(filepath.Base(path), len(videos))
			m.db.UpsertGlobalMapping(&models.GlobalAnimeFileMapping{
				AniListID:     intent.MediaID,
				LocalFilePath: path,
				EpisodeNumber: ep,
				FileType:      "main",
				LastScanned:   time.Now(),
			})
		}

		m.db.MarkPendingDownloadIntentCompleted(intent.Hash)
		m.logger.Info().Str("hash", intent.Hash).Int("files", len(videos)).Msg("downloadmonitor: Completed intent")
	}
}

func collectVideos(contentPath string) []string {
	if contentPath == "" {
		return nil
	}
	info, err := os.Stat(contentPath)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		if isVideo(contentPath) {
			return []string{contentPath}
		}
		return nil
	}
	var videos []string
	filepath.WalkDir(contentPath, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && isVideo(path) {
			videos = append(videos, path)
		}
		return nil
	})
	return videos
}

func isVideo(path string) bool {
	_, ok := videoExts[strings.ToLower(filepath.Ext(path))]
	return ok
}

func parseEpisode(filename string, totalFiles int) int {
	meta := habari.Parse(filename)
	if meta != nil && len(meta.EpisodeNumber) > 0 {
		if n, err := strconv.Atoi(meta.EpisodeNumber[0]); err == nil {
			return n
		}
	}
	if totalFiles == 1 {
		return 1
	}
	return 0
}
