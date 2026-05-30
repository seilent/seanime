package mediastream

import (
	"fmt"
	"seanime/internal/mediastream/videofile"
	"seanime/internal/util/result"

	"github.com/rs/zerolog"
	"github.com/samber/mo"
)

const (
	StreamTypeDirect StreamType = "direct"    // Direct streaming
)

type (
	StreamType string

	PlaybackManager struct {
		logger                *zerolog.Logger
		currentMediaContainers map[string]mo.Option[*MediaContainer] // Client ID -> current media being played.
		repository            *Repository
		mediaContainers       *result.Map[string, *MediaContainer] // Temporary cache for the media containers.
	}

	PlaybackState struct {
		MediaId int `json:"mediaId"` // The media ID
	}

	MediaContainer struct {
		Filepath   string               `json:"filePath"`
		Hash       string               `json:"hash"`
		StreamType StreamType           `json:"streamType"` // Tells the frontend how to play the media.
		StreamUrl  string               `json:"streamUrl"`  // The relative endpoint to stream the media.
		MediaInfo  *videofile.MediaInfo `json:"mediaInfo"`
		//Metadata  *Metadata       `json:"metadata"`
		// todo: add more fields (e.g. metadata)
	}
)

func NewPlaybackManager(repository *Repository) *PlaybackManager {
	return &PlaybackManager{
		logger:                  repository.logger,
		repository:              repository,
		mediaContainers:         result.NewResultMap[string, *MediaContainer](),
		currentMediaContainers:  make(map[string]mo.Option[*MediaContainer]),
	}
}

func (p *PlaybackManager) KillPlayback() {
	p.logger.Debug().Msg("mediastream: Killing playback for all clients")
	for clientId := range p.currentMediaContainers {
		p.currentMediaContainers[clientId] = mo.None[*MediaContainer]()
		p.logger.Trace().Str("clientId", clientId).Msg("mediastream: Removed current media container for client")
	}
}

func (p *PlaybackManager) KillPlaybackForClient(clientId string) {
	p.logger.Debug().Str("clientId", clientId).Msg("mediastream: Killing playback for client")
	if container, exists := p.currentMediaContainers[clientId]; exists && container.IsPresent() {
		p.currentMediaContainers[clientId] = mo.None[*MediaContainer]()
		p.logger.Trace().Str("clientId", clientId).Msg("mediastream: Removed current media container for client")
	}
}

// RequestPlayback is called by the frontend to stream a media file
func (p *PlaybackManager) RequestPlayback(filepath string, streamType StreamType, clientId string) (ret *MediaContainer, err error) {

	p.logger.Debug().Str("filepath", filepath).Any("type", streamType).Str("clientId", clientId).Msg("mediastream: Requesting playback")

	// Create a new media container
	ret, err = p.newMediaContainer(filepath, streamType)

	if err != nil {
		p.logger.Error().Err(err).Msg("mediastream: Failed to create media container")
		return nil, fmt.Errorf("failed to create media container: %v", err)
	}

	// Set the current media container for this client.
	p.currentMediaContainers[clientId] = mo.Some(ret)

	p.logger.Info().Str("filepath", filepath).Str("clientId", clientId).Msg("mediastream: Ready to play media")

	return
}

// PreloadPlayback is called by the frontend to preload a media container so that the data is stored in advanced
func (p *PlaybackManager) PreloadPlayback(filepath string, streamType StreamType) (ret *MediaContainer, err error) {

	p.logger.Debug().Str("filepath", filepath).Any("type", streamType).Msg("mediastream: Preloading playback")

	// Create a new media container
	ret, err = p.newMediaContainer(filepath, streamType)

	if err != nil {
		p.logger.Error().Err(err).Msg("mediastream: Failed to create media container")
		return nil, fmt.Errorf("failed to create media container: %v", err)
	}

	p.logger.Info().Str("filepath", filepath).Msg("mediastream: Ready to play media")

	return
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Optimize
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (p *PlaybackManager) newMediaContainer(filepath string, streamType StreamType) (ret *MediaContainer, err error) {
	p.logger.Debug().Str("filepath", filepath).Any("type", streamType).Msg("mediastream: New media container requested")
	// Get the hash of the file.
	hash, err := videofile.GetHashFromPath(filepath)
	if err != nil {
		return nil, err
	}

	p.logger.Trace().Str("hash", hash).Msg("mediastream: Checking cache")

	// Check the cache ONLY if the stream type is the same.
	if mc, ok := p.mediaContainers.Get(hash); ok && mc.StreamType == streamType {
		p.logger.Debug().Str("hash", hash).Msg("mediastream: Media container cache HIT")
		return mc, nil
	}

	p.logger.Trace().Str("hash", hash).Msg("mediastream: Creating media container")

	// Get the media information of the file.
	ret = &MediaContainer{
		Filepath:   filepath,
		Hash:       hash,
		StreamType: streamType,
	}

	p.logger.Debug().Msg("mediastream: Extracting media info")

	ret.MediaInfo, err = p.repository.mediaInfoExtractor.GetInfo("ffprobe", filepath)
	if err != nil {
		return nil, err
	}

	p.logger.Debug().Msg("mediastream: Extracted media info, extracting attachments")

	// Extract the attachments from the file.
	err = videofile.ExtractAttachment("ffmpeg", filepath, hash, ret.MediaInfo, p.repository.cacheDir, p.logger)
	if err != nil {
		p.logger.Error().Err(err).Msg("mediastream: Failed to extract attachments")
		return nil, err
	}

	p.logger.Debug().Msg("mediastream: Extracted attachments")

	// Directly serve the file.
	streamUrl := fmt.Sprintf("/api/v1/mediastream/direct?v=%s", hash)

	// Set the stream URL.
	ret.StreamUrl = streamUrl

	// Store the media container in the map.
	p.mediaContainers.Set(hash, ret)

	return
}
