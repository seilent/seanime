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

type Remuxer interface {
	PrewarmDirectPlay(sourcePath string) error
	RemuxToFile(sourcePath, destPath string) error
	ExtractToCache(sourcePath string, targetHash string) (*videofile.MediaInfo, error)
	CacheDir() string
}

type Monitor struct {
	db                 *db.Database
	repo               *torrent_client.Repository
	logger             *zerolog.Logger
	wsEventManager     events.WSEventManagerInterface
	mediaInfoExtractor *videofile.MediaInfoExtractor
	remuxer            Remuxer
	cacheDir           string
	inFlight           sync.Map
	sem                chan struct{}
}

func New(db *db.Database, repo *torrent_client.Repository, logger *zerolog.Logger, wsEventManager events.WSEventManagerInterface, fileCacher *filecache.Cacher, remuxer Remuxer, cacheDir string) *Monitor {
	mie := videofile.NewMediaInfoExtractor(fileCacher, logger)
	mie.SetCacheDir(cacheDir)
	return &Monitor{
		db:                 db,
		repo:               repo,
		logger:             logger,
		wsEventManager:     wsEventManager,
		mediaInfoExtractor: mie,
		remuxer:            remuxer,
		cacheDir:           cacheDir,
		sem:                make(chan struct{}, 2),
	}
}

var (
	monitorMu  sync.Mutex
	activeStop chan struct{}
)

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
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error().Msgf("downloadmonitor: panic in emitProgress: %v", r)
		}
	}()

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

		if intent.FlattenState != "linked" {
			if !found || t.Progress < 1.0 {
				continue
			}

			contentPath := t.ContentPath
			_, statErr := os.Stat(contentPath)
			videos := collectVideos(contentPath)
			if len(videos) == 0 {
				if statErr != nil {
					m.logger.Warn().Str("hash", intent.Hash).Str("contentPath", contentPath).Msg("downloadmonitor: content path missing, abandoning intent")
					m.db.MarkPendingDownloadIntentCompleted(intent.Hash)
					continue
				}
				m.logger.Warn().Str("hash", intent.Hash).Str("contentPath", contentPath).Msg("downloadmonitor: torrent reports complete but no video files found, leaving intent pending for retry")
				continue
			}

			if _, loaded := m.inFlight.LoadOrStore(intent.Hash, struct{}{}); loaded {
				continue
			}

			intentCopy := intent
			tCopy := t
			go func() {
				m.sem <- struct{}{}
				defer func() {
					<-m.sem
					m.inFlight.Delete(intentCopy.Hash)
					if r := recover(); r != nil {
						m.logger.Error().Msgf("downloadmonitor: panic in processCompletedIntent: %v", r)
					}
				}()
				m.processCompletedIntent(intentCopy, tCopy)
			}()
			continue
		}

		if found && t.Status != torrent_client.TorrentStatusStopped {
			continue
		}

		if intent.ContentPath != "" && intent.Destination != "" {
			cp := filepath.Clean(intent.ContentPath)
			dest := filepath.Clean(intent.Destination)
			if cp != dest && filepath.Dir(cp) == dest {
				libraryPaths, _ := m.db.GetAllLibraryPathsFromSettings()
				inside := false
				for _, lp := range libraryPaths {
					if strings.HasPrefix(cp, lp+string(filepath.Separator)) || cp == lp {
						inside = true
						break
					}
				}
				if inside {
					os.RemoveAll(cp)
				}
			}
		}

		m.db.MarkPendingDownloadIntentCompleted(intent.Hash)
		m.logger.Info().Str("hash", intent.Hash).Msg("downloadmonitor: Completed intent")
	}
}

func (m *Monitor) processCompletedIntent(intent *models.PendingDownloadIntent, t *torrent_client.Torrent) {
	contentPath := t.ContentPath
	videos := collectVideos(contentPath)

	destDir := intent.Destination
	if destDir == "" {
		destDir = filepath.Dir(contentPath)
	}

	_ = m.repo.PauseTorrents([]string{intent.Hash})

	existingMappings, _ := m.db.GetGlobalMappingsByAniListID(intent.MediaID)
	byEp := make(map[int][]*models.GlobalAnimeFileMapping)
	for _, em := range existingMappings {
		byEp[em.EpisodeNumber] = append(byEp[em.EpisodeNumber], em)
	}

	allMaterialized := true
	for _, v := range videos {
		ext := strings.ToLower(filepath.Ext(v))
		base := strings.TrimSuffix(filepath.Base(v), filepath.Ext(v))
		var target string
		var usedFallback bool

		if ext == ".mkv" && m.remuxer != nil {
			mp4Name := base + ".mp4"
			mp4Path := filepath.Join(destDir, mp4Name)
			if err := os.MkdirAll(destDir, 0755); err == nil {
				if err := m.remuxer.RemuxToFile(v, mp4Path); err == nil {
					mp4Hash, hashErr := videofile.GetHashFromPath(mp4Path)
					if hashErr == nil {
						cacheDir := m.remuxer.CacheDir()
						if cacheDir == "" {
							cacheDir = m.cacheDir
						}
						mkvInfo, extractErr := m.remuxer.ExtractToCache(v, mp4Hash)
						if extractErr == nil && mkvInfo != nil {
							_ = videofile.WriteSubsManifest(cacheDir, mp4Hash, mkvInfo.Subtitles, mkvInfo.Fonts)
						} else {
							m.logger.Warn().Err(extractErr).Str("path", v).Msg("downloadmonitor: sub extraction failed, playback may lack subs")
						}
					}
					target = mp4Path
				} else {
					m.logger.Warn().Err(err).Str("path", v).Msg("downloadmonitor: remux failed, falling back to copy")
					usedFallback = true
				}
			} else {
				m.logger.Warn().Err(err).Str("dir", destDir).Msg("downloadmonitor: mkdir failed, falling back to copy")
				usedFallback = true
			}
		} else {
			usedFallback = true
		}

		if usedFallback {
			target = filepath.Join(destDir, filepath.Base(v))
			if err := os.MkdirAll(destDir, 0755); err != nil {
				m.logger.Warn().Err(err).Str("dir", destDir).Msg("downloadmonitor: mkdir failed for fallback")
				target = v
			} else if target != v {
				if e := util.HardlinkOrCopy(v, target); e != nil {
					m.logger.Warn().Err(e).Str("src", v).Str("dst", target).Msg("downloadmonitor: hardlink failed, mapping original")
					target = v
				}
			}

			if m.remuxer != nil {
				go func(p string) {
					if m.cacheDir != "" {
						if free, err := util.GetFreeSpace(m.cacheDir); err == nil && free < util.DiskFloorBytes {
							return
						}
					}
					if err := m.remuxer.PrewarmDirectPlay(p); err != nil {
						m.logger.Warn().Err(err).Str("path", p).Msg("downloadmonitor: direct play prewarm failed")
					}
				}(target)
			}
		}

		if filepath.Clean(target) == filepath.Clean(v) {
			allMaterialized = false
		}

		ep := parseEpisode(filepath.Base(target), len(videos))

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

		if _, err := m.mediaInfoExtractor.GetInfo("ffprobe", target); err != nil {
			m.logger.Warn().Err(err).Str("path", target).Msg("downloadmonitor: failed to extract media info")
		}
	}

	if allMaterialized {
		_ = m.repo.RemoveTorrents([]string{intent.Hash})
		m.db.MarkPendingDownloadIntentCompleted(intent.Hash)
	} else {
		m.db.SetPendingDownloadIntentLinked(intent.Hash, "")
		_ = m.repo.ResumeTorrents([]string{intent.Hash})
	}

	if m.wsEventManager != nil {
		m.wsEventManager.SendEvent(events.InvalidateQueries, []string{events.GetAnimeEntryEndpoint})
	}
	m.logger.Info().Str("hash", intent.Hash).Int("files", len(videos)).Msg("downloadmonitor: Linked intent")
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
