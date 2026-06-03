package downloadmonitor

import (
	"os"
	"path/filepath"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/mediastream/videofile"
	"seanime/internal/torrent_clients/torrent_client"
	"seanime/internal/util"
	"seanime/internal/util/filecache"
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

type downloadProgressItem struct {
	MediaID  int     `json:"mediaId"`
	Episode  int     `json:"episode"`
	Progress float64 `json:"progress"`
}

type Monitor struct {
	db                 *db.Database
	repo               *torrent_client.Repository
	logger             *zerolog.Logger
	wsEventManager     events.WSEventManagerInterface
	mediaInfoExtractor *videofile.MediaInfoExtractor
}

func New(db *db.Database, repo *torrent_client.Repository, logger *zerolog.Logger, wsEventManager events.WSEventManagerInterface, fileCacher *filecache.Cacher) *Monitor {
	return &Monitor{
		db:                 db,
		repo:               repo,
		logger:             logger,
		wsEventManager:     wsEventManager,
		mediaInfoExtractor: videofile.NewMediaInfoExtractor(fileCacher, logger),
	}
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
	progressTicker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	defer progressTicker.Stop()
	go func() {
		for {
			select {
			case <-progressTicker.C:
				m.emitProgress()
			case <-stop:
				return
			}
		}
	}()
	for {
		select {
		case <-ticker.C:
			m.tick()
		case <-stop:
			return
		}
	}
}

func (m *Monitor) emitProgress() {
	intents, _ := m.db.GetIncompletePendingDownloadIntents()
	if len(intents) == 0 {
		return
	}
	torrents, _ := m.repo.GetList()
	byHash := make(map[string]*torrent_client.Torrent, len(torrents))
	for _, t := range torrents {
		byHash[strings.ToLower(t.Hash)] = t
	}
	items := make([]downloadProgressItem, 0)
	for _, intent := range intents {
		if intent.FlattenState == "linked" {
			continue
		}
		t, found := byHash[intent.Hash]
		if !found {
			continue
		}
		episode := 0
		if meta := habari.Parse(t.Name); meta != nil && len(meta.EpisodeNumber) == 1 {
			if n, e := strconv.Atoi(meta.EpisodeNumber[0]); e == nil {
				episode = n
			}
		}
		items = append(items, downloadProgressItem{MediaID: intent.MediaID, Episode: episode, Progress: t.Progress})
	}
	if m.wsEventManager != nil {
		m.wsEventManager.SendEvent(events.DownloadProgress, items)
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
		t, found := byHash[intent.Hash]

		// PHASE 1 — flatten + map (when files fully downloaded, once)
		if intent.FlattenState != "linked" {
			if !found || t.Progress < 1.0 {
				continue // wait for full download
			}

			contentPath := t.ContentPath
			info, statErr := os.Stat(contentPath)
			isFolder := statErr == nil && info.IsDir()
			videos := collectVideos(contentPath)
			destDir := intent.Destination // known anime/<Title>/ dir from download time
			if destDir == "" {
				destDir = filepath.Dir(contentPath)
			}
			storedContentPath := ""

			// Build per-episode index of existing mappings for per-episode replace
			existingMappings, _ := m.db.GetGlobalMappingsByAniListID(intent.MediaID)
			byEp := make(map[int][]*models.GlobalAnimeFileMapping)
			for _, em := range existingMappings {
				byEp[em.EpisodeNumber] = append(byEp[em.EpisodeNumber], em)
			}

			for _, v := range videos {
				target := v
				if isFolder {
					target = filepath.Join(destDir, filepath.Base(v))
					if e := util.HardlinkOrCopy(v, target); e != nil {
						m.logger.Warn().Err(e).Str("src", v).Str("dst", target).Msg("downloadmonitor: hardlink failed, mapping original")
						target = v // fall back to mapping original path
					}
				}
				ep := parseEpisode(filepath.Base(target), len(videos))

				// Per-episode replace: remove old file(s) for this (media, episode) if different path
				cleanTarget := filepath.Clean(target)
				for _, old := range byEp[ep] {
					if filepath.Clean(old.LocalFilePath) != cleanTarget {
						if err := os.Remove(old.LocalFilePath); err != nil {
							m.logger.Debug().Err(err).Str("path", old.LocalFilePath).Msg("downloadmonitor: could not remove old episode file")
						}
						_ = m.db.DeleteGlobalMapping(old.LocalFilePath)
					}
				}

				m.db.UpsertGlobalMapping(&models.GlobalAnimeFileMapping{
					AniListID:     intent.MediaID,
					LocalFilePath: target,
					EpisodeNumber: ep,
					FileType:      "main",
					LastScanned:   time.Now(),
				})

				// Pre-extract media info so it's cached before first playback
				if _, err := m.mediaInfoExtractor.GetInfo("ffprobe", target); err != nil {
					m.logger.Warn().Err(err).Str("path", target).Msg("downloadmonitor: failed to extract media info")
				}
			}

			if isFolder {
				storedContentPath = contentPath
			}
			m.db.SetPendingDownloadIntentLinked(intent.Hash, storedContentPath)
			if m.wsEventManager != nil {
				m.wsEventManager.SendEvent(events.InvalidateQueries, []string{events.GetAnimeEntryEndpoint})
			}
			m.logger.Info().Str("hash", intent.Hash).Int("files", len(videos)).Msg("downloadmonitor: Linked intent")
			continue
		}

		// PHASE 2 — cleanup torrent subfolder after seeding stops (or torrent gone)
		if found && t.Status != torrent_client.TorrentStatusStopped {
			continue // still seeding
		}

		if intent.ContentPath != "" && intent.Destination != "" {
			cp := filepath.Clean(intent.ContentPath)
			dest := filepath.Clean(intent.Destination)
			if cp != dest && filepath.Dir(cp) == dest {
				os.RemoveAll(cp)
			}
		}

		m.db.MarkPendingDownloadIntentCompleted(intent.Hash)
		m.logger.Info().Str("hash", intent.Hash).Msg("downloadmonitor: Completed intent")
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
