package mediastream

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"seanime/internal/events"
	"seanime/internal/mediastream/videofile"
	"seanime/internal/util"
	"strings"

	"github.com/labstack/echo/v4"
)

func (r *Repository) ServeEchoFile(c echo.Context, rawFilePath string, clientId string, libraryPaths []string) error {
	filePath, _ := url.PathUnescape(rawFilePath)

	if util.IsBase64(rawFilePath) {
		var err error
		filePath, err = util.Base64DecodeStr(rawFilePath)
		if err != nil {
			filePath, _ = url.PathUnescape(rawFilePath)
		}
	}

	inLibrary := false
	for _, libraryPath := range libraryPaths {
		if util.IsFileUnderDir(filePath, libraryPath) {
			inLibrary = true
			break
		}
	}

	if !inLibrary {
		return c.NoContent(http.StatusNotFound)
	}

	r.logger.Trace().Str("filepath", filePath).Str("payload", rawFilePath).Msg("mediastream: Served file")
	filename := filepath.Base(filePath)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	return c.File(filePath)
}

func (r *Repository) ServeEchoDirectPlay(c echo.Context, clientId string) error {

	if !r.IsInitialized() {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "Module not initialized")
		return errors.New("module not initialized")
	}

	mediaContainer, found := r.playbackManager.currentMediaContainers[clientId].Get()
	if !found {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "no file has been loaded for client")
		return errors.New("no file has been loaded for client")
	}

	filePath := mediaContainer.Filepath

	userAgent := c.Request().UserAgent()
	needsMP4 := !browserSupportsMKV(userAgent)
	isMKV := filepath.Ext(filePath) == ".mkv"

	if needsMP4 && isMKV {
		mp4Path := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".mp4"

		if _, err := os.Stat(mp4Path); os.IsNotExist(err) {
			r.logger.Info().Str("source", filePath).Str("target", mp4Path).Msg("mediastream: Remuxing MKV to MP4 in-place")

			if err := r.createRemuxedFile(filePath, mp4Path); err != nil {
				r.logger.Error().Err(err).Msg("mediastream: Failed to remux, serving original")
			} else {
				r.preserveSubtitlesForRemux(filePath, mp4Path)
				os.Remove(filePath)

				if r.globalMappingService != nil {
					fileInfo, _ := os.Stat(mp4Path)
					var size int64
					if fileInfo != nil {
						size = fileInfo.Size()
					}
					r.globalMappingService.RenameFileMapping(filePath, mp4Path, size)
				}

				filePath = mp4Path
			}
		} else {
			filePath = mp4Path
		}
	}

	if c.Request().Method == http.MethodHead {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}

		c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		c.Response().Header().Set("Content-Type", "video/mp4")
		c.Response().Header().Set("Accept-Ranges", "bytes")
		filename := filepath.Base(filePath)
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
		return c.NoContent(http.StatusOK)
	}

	return c.File(filePath)
}

func browserSupportsMKV(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	ua := strings.ToLower(userAgent)

	if strings.Contains(ua, "firefox") {
		return false
	}

	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		return false
	}

	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") {
		return false
	}

	return true
}

func (r *Repository) createRemuxedFile(inputPath, outputPath string) error {
	tempPath := outputPath + ".tmp"

	os.Remove(tempPath)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:v", "copy", "-c:a", "copy", "-sn", "-f", "mp4", tempPath)

	var stderr, stdout strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	r.logger.Debug().Str("input", inputPath).Str("output", tempPath).Msg("mediastream: Starting ffmpeg remux")

	if err := cmd.Run(); err != nil {
		r.logger.Error().Err(err).Str("stderr", stderr.String()).Str("stdout", stdout.String()).Msg("mediastream: ffmpeg remux failed")
		os.Remove(tempPath)
		return fmt.Errorf("ffmpeg remux failed: %w", err)
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		r.logger.Error().Err(err).Msg("mediastream: Failed to rename temporary file")
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	r.logger.Debug().Str("input", inputPath).Str("output", outputPath).Msg("mediastream: Successfully created remuxed file")
	return nil
}

func (r *Repository) preserveSubtitlesForRemux(sourcePath, mp4Path string) {
	srcInfo, err := r.mediaInfoExtractor.GetInfo("ffprobe", sourcePath)
	if err != nil || len(srcInfo.Subtitles) == 0 {
		return
	}

	mp4Hash, err := videofile.GetHashFromPath(mp4Path)
	if err != nil {
		r.logger.Error().Err(err).Msg("mediastream: Failed to hash remuxed file for subtitle preservation")
		return
	}

	if err := videofile.ExtractAttachment("ffmpeg", sourcePath, mp4Hash, srcInfo, r.cacheDir, r.logger); err != nil {
		r.logger.Error().Err(err).Msg("mediastream: Failed to preserve subtitles during remux")
		return
	}

	mp4Info, err := videofile.FfprobeGetInfo("ffprobe", mp4Path, mp4Hash)
	if err != nil {
		r.logger.Error().Err(err).Msg("mediastream: Failed to probe remuxed file for subtitle preservation")
		return
	}
	mp4Info.Subtitles = srcInfo.Subtitles
	mp4Info.Fonts = srcInfo.Fonts

	if err := r.mediaInfoExtractor.SetInfo(mp4Path, mp4Info); err != nil {
		r.logger.Error().Err(err).Msg("mediastream: Failed to cache remuxed media info")
	}
}
