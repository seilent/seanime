package anime

import (
	"context"
	"fmt"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/platforms/platform"
	"seanime/internal/util/result"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
)

var episodeCollectionCache = result.NewBoundedCache[int, *EpisodeCollection](10)
var EpisodeCollectionFromLocalFilesCache = result.NewBoundedCache[int, *EpisodeCollection](10)

type (
	// EpisodeCollection represents a collection of episodes.
	EpisodeCollection struct {
		HasMappingError bool                    `json:"hasMappingError"`
		Episodes        []*Episode              `json:"episodes"`
		Metadata        *metadata.AnimeMetadata `json:"metadata"`
	}
)


func ClearEpisodeCollectionCache() {
	episodeCollectionCache.Clear()
}

/////////

type NewEpisodeCollectionFromLocalFilesOptions struct {
	LocalFiles       []*LocalFile
	Media            *anilist.BaseAnime
	AnimeCollection  *anilist.AnimeCollection
	Platform         platform.Platform
	MetadataProvider metadata.Provider
	Logger           *zerolog.Logger
}

func NewEpisodeCollectionFromLocalFiles(ctx context.Context, opts NewEpisodeCollectionFromLocalFilesOptions) (*EpisodeCollection, error) {
	if opts.Logger == nil {
		opts.Logger = lo.ToPtr(zerolog.Nop())
	}

	if ec, ok := EpisodeCollectionFromLocalFilesCache.Get(opts.Media.GetID()); ok {
		return ec, nil
	}

	// Make sure to keep the local files from the media only
	opts.LocalFiles = lo.Filter(opts.LocalFiles, func(lf *LocalFile, i int) bool {
		return lf.MediaId == opts.Media.GetID()
	})

	// Create a new media entry
	entry, err := NewEntry(ctx, &NewEntryOptions{
		MediaId:          opts.Media.GetID(),
		LocalFiles:       opts.LocalFiles,
		AnimeCollection:  opts.AnimeCollection,
		Platform:         opts.Platform,
		MetadataProvider: opts.MetadataProvider,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot play local file, could not create entry: %w", err)
	}

	// Should be cached if it exists
	animeMetadata, err := opts.MetadataProvider.GetAnimeMetadata(metadata.AnilistPlatform, opts.Media.ID)
	if err != nil {
		animeMetadata = &metadata.AnimeMetadata{
			Titles:       make(map[string]string),
			Episodes:     make(map[string]*metadata.EpisodeMetadata),
			EpisodeCount: 0,
			SpecialCount: 0,
			Mappings: &metadata.AnimeMappings{
				AnilistId: opts.Media.GetID(),
			},
		}
		animeMetadata.Titles["en"] = opts.Media.GetTitleSafe()
		animeMetadata.Titles["x-jat"] = opts.Media.GetRomajiTitleSafe()
		err = nil
	}

	ec := &EpisodeCollection{
		HasMappingError: false,
		Episodes:        entry.Episodes,
		Metadata:        animeMetadata,
	}

	EpisodeCollectionFromLocalFilesCache.SetT(opts.Media.GetID(), ec, time.Hour*6)

	return ec, nil
}

/////////

func (ec *EpisodeCollection) FindEpisodeByNumber(episodeNumber int) (*Episode, bool) {
	for _, episode := range ec.Episodes {
		if episode.EpisodeNumber == episodeNumber {
			return episode, true
		}
	}
	return nil, false
}

func (ec *EpisodeCollection) FindEpisodeByAniDB(anidbEpisode string) (*Episode, bool) {
	for _, episode := range ec.Episodes {
		if episode.AniDBEpisode == anidbEpisode {
			return episode, true
		}
	}
	return nil, false
}

// GetMainLocalFiles returns the *main* local files.
func (ec *EpisodeCollection) GetMainLocalFiles() ([]*Episode, bool) {
	ret := make([]*Episode, 0)
	for _, episode := range ec.Episodes {
		if episode.LocalFile == nil || episode.LocalFile.IsMain() {
			ret = append(ret, episode)
		}
	}
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

// FindNextEpisode returns the *main* local file whose episode number is after the given local file.
func (ec *EpisodeCollection) FindNextEpisode(current *Episode) (*Episode, bool) {
	episodes, ok := ec.GetMainLocalFiles()
	if !ok {
		return nil, false
	}
	// Get the local file whose episode number is after the given local file
	var next *Episode
	for _, e := range episodes {
		if e.GetEpisodeNumber() == current.GetEpisodeNumber()+1 {
			next = e
			break
		}
	}
	if next == nil {
		return nil, false
	}
	return next, true
}
