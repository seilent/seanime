package mediastream

import (
	"errors"
	"github.com/rs/zerolog"
	"github.com/samber/mo"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/global_mapping"
	"seanime/internal/mediastream/optimizer"
	"seanime/internal/mediastream/videofile"
	"seanime/internal/util/filecache"
	"sync"

	"golang.org/x/sync/singleflight"
)

type (
	Repository struct {
		optimizer            *optimizer.Optimizer
		settings             mo.Option[*models.GlobalSettings]
		playbackManager      *PlaybackManager
		mediaInfoExtractor   *videofile.MediaInfoExtractor
		logger               *zerolog.Logger
		wsEventManager       events.WSEventManagerInterface
		fileCacher           *filecache.Cacher
		reqMu                sync.Mutex
		remuxGroup           singleflight.Group
		cacheDir             string
		globalMappingService *global_mapping.GlobalMappingService
		inflightRemuxHashes  sync.Map
	}

	NewRepositoryOptions struct {
		Logger               *zerolog.Logger
		WSEventManager       events.WSEventManagerInterface
		FileCacher           *filecache.Cacher
		GlobalMappingService *global_mapping.GlobalMappingService
	}
)

func NewRepository(opts *NewRepositoryOptions) *Repository {
	ret := &Repository{
		logger: opts.Logger,
		optimizer: optimizer.NewOptimizer(&optimizer.NewOptimizerOptions{
			Logger:         opts.Logger,
			WSEventManager: opts.WSEventManager,
		}),
		settings:             mo.None[*models.GlobalSettings](),
		wsEventManager:       opts.WSEventManager,
		fileCacher:           opts.FileCacher,
		mediaInfoExtractor:   videofile.NewMediaInfoExtractor(opts.FileCacher, opts.Logger),
		globalMappingService: opts.GlobalMappingService,
	}
	ret.playbackManager = NewPlaybackManager(ret)

	return ret
}

func (r *Repository) IsInitialized() bool {
	return r.settings.IsPresent()
}

func (r *Repository) OnCleanup() {

}

func (r *Repository) InitializeModules(settings *models.GlobalSettings, cacheDir string) {
	if settings == nil {
		r.logger.Error().Msg("mediastream: Settings not present")
		return
	}

	r.settings = mo.Some[*models.GlobalSettings](settings)

	r.cacheDir = cacheDir

	r.optimizer.SetLibraryDir(settings.Library.LibraryPath)

	r.logger.Info().Msg("mediastream: Module initialized")
}

func (r *Repository) CacheWasCleared() {
	r.playbackManager.mediaContainers.Clear()
}

func (r *Repository) ActiveVideoFileHashes() map[string]struct{} {
	var hashes map[string]struct{}
	if r.playbackManager != nil {
		hashes = r.playbackManager.ActiveVideoFileHashes()
	}
	if hashes == nil {
		hashes = make(map[string]struct{})
	}
	r.inflightRemuxHashes.Range(func(key, _ interface{}) bool {
		hashes[key.(string)] = struct{}{}
		return true
	})
	return hashes
}

func (r *Repository) RequestDirectPlay(filepath string, clientId string) (ret *MediaContainer, err error) {
	r.reqMu.Lock()
	defer r.reqMu.Unlock()

	r.logger.Debug().Str("filepath", filepath).Msg("mediastream: Direct play requested")

	if !r.IsInitialized() {
		return nil, errors.New("module not initialized")
	}

	ret, err = r.playbackManager.RequestPlayback(filepath, StreamTypeDirect, clientId)

	return
}

func (r *Repository) RequestPreloadDirectPlay(filepath string) (err error) {
	r.logger.Debug().Str("filepath", filepath).Msg("mediastream: Direct stream preloading requested")

	if !r.IsInitialized() {
		return errors.New("module not initialized")
	}

	_, err = r.playbackManager.PreloadPlayback(filepath, StreamTypeDirect)

	return
}
