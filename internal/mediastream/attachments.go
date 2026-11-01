package mediastream

import (
	"errors"
	"net/url"
	"path/filepath"
	"seanime/internal/events"
	"seanime/internal/mediastream/videofile"
	"strings"

	"github.com/labstack/echo/v4"
)

func (r *Repository) ServeEchoExtractedSubtitles(c echo.Context, clientId string) error {

	if !r.IsInitialized() {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "Module not initialized")
		return errors.New("module not initialized")
	}

	
	// Get the parameter group
	subFilePath := c.Param("*")

	// Get current media for this client
	mediaContainer, found := r.playbackManager.GetCurrentMediaContainer(clientId)
	if !found {
		return errors.New("no file has been loaded for client")
	}

	retPath := videofile.GetFileSubsCacheDir(r.cacheDir, mediaContainer.Hash)

	if retPath == "" {
		return errors.New("could not find subtitles")
	}

	r.logger.Trace().Msgf("mediastream: Serving subtitles from %s", retPath)

	return serveFileInDir(c, retPath, subFilePath)
}

func serveFileInDir(c echo.Context, dir string, name string) error {
	p := filepath.Join(dir, name)
	rel, err := filepath.Rel(dir, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return echo.ErrNotFound
	}
	return c.File(p)
}

func (r *Repository) ServeEchoExtractedAttachments(c echo.Context, clientId string) error {
	if !r.IsInitialized() {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "Module not initialized")
		return errors.New("module not initialized")
	}

	
	// Get the parameter group
	subFilePath := c.Param("*")

	// Get current media for this client
	mediaContainer, found := r.playbackManager.GetCurrentMediaContainer(clientId)
	if !found {
		return errors.New("no file has been loaded for client")
	}

	retPath := videofile.GetFileAttCacheDir(r.cacheDir, mediaContainer.Hash)

	if retPath == "" {
		return errors.New("could not find subtitles")
	}

	subFilePath, _ = url.PathUnescape(subFilePath)

	return serveFileInDir(c, retPath, subFilePath)
}
