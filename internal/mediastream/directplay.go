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

	mediaContainer, found := r.playbackManager.GetCurrentMediaContainer(clientId)
	if !found {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "no file has been loaded for client")
		return errors.New("no file has been loaded for client")
	}

	filePath := mediaContainer.Filepath

	userAgent := c.Request().UserAgent()
	needsMP4 := !browserSupportsMKV(userAgent)
	isMKV := filepath.Ext(filePath) == ".mkv"

	if needsMP4 && isMKV {
		mp4Path := r.directPlayMP4Path(mediaContainer.Hash)

		if _, err := os.Stat(mp4Path); os.IsNotExist(err) {
			_, err, _ := r.remuxGroup.Do(mp4Path, func() (interface{}, error) {
				if _, e := os.Stat(mp4Path); e == nil {
					return nil, nil
				}
				if e := os.MkdirAll(filepath.Dir(mp4Path), 0755); e != nil {
					return nil, e
				}
				r.logger.Info().Str("source", filePath).Str("target", mp4Path).Msg("mediastream: Remuxing MKV to MP4 (cache)")
				return nil, r.createRemuxedFile(filePath, mp4Path)
			})

			if err != nil {
				r.logger.Error().Err(err).Msg("mediastream: Failed to remux, serving original")
			} else {
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
	hash := filepath.Base(filepath.Dir(outputPath))
	r.inflightRemuxHashes.Store(hash, struct{}{})
	defer r.inflightRemuxHashes.Delete(hash)

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

func (r *Repository) directPlayMP4Path(hash string) string {
	return filepath.Join(r.cacheDir, "videofiles", hash, "direct.mp4")
}

func (r *Repository) PrewarmDirectPlay(sourcePath string) error {
	if !r.IsInitialized() {
		return nil
	}

	hash, err := videofile.GetHashFromPath(sourcePath)
	if err != nil {
		return err
	}

	info, err := r.mediaInfoExtractor.GetInfo("ffprobe", sourcePath)
	if err != nil {
		return err
	}

	if err := videofile.ExtractAttachment("ffmpeg", sourcePath, hash, info, r.cacheDir, r.logger); err != nil {
		r.logger.Warn().Err(err).Str("path", sourcePath).Msg("mediastream: Failed to pre-extract subtitles")
	}

	if strings.ToLower(filepath.Ext(sourcePath)) != ".mkv" {
		return nil
	}

	mp4Path := r.directPlayMP4Path(hash)
	if _, e := os.Stat(mp4Path); e == nil {
		return nil
	}

	_, err, _ = r.remuxGroup.Do(mp4Path, func() (interface{}, error) {
		if _, e := os.Stat(mp4Path); e == nil {
			return nil, nil
		}
		if e := os.MkdirAll(filepath.Dir(mp4Path), 0755); e != nil {
			return nil, e
		}
		r.logger.Info().Str("source", sourcePath).Str("target", mp4Path).Msg("mediastream: Remuxing MKV to MP4 (prewarm)")
		return nil, r.createRemuxedFile(sourcePath, mp4Path)
	})
	return err
}

func (r *Repository) RemuxToFile(sourcePath, destPath string) error {
	tempPath := destPath + ".tmp"
	os.Remove(tempPath)

	cmd := exec.Command("ffmpeg", "-i", sourcePath, "-map", "0:v", "-map", "0:a", "-c:v", "copy", "-c:a", "copy", "-sn", "-dn", "-f", "mp4", tempPath)
	var stderr, stdout strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	r.logger.Info().Str("source", sourcePath).Str("dest", destPath).Msg("mediastream: Remuxing MKV to MP4 (library)")

	if err := cmd.Run(); err != nil {
		r.logger.Error().Err(err).Str("stderr", stderr.String()).Msg("mediastream: ffmpeg library remux failed")
		os.Remove(tempPath)
		return fmt.Errorf("ffmpeg remux failed: %w", err)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("rename failed: %w", err)
	}

	return nil
}

func (r *Repository) ExtractToCache(sourcePath string, targetHash string) (*videofile.MediaInfo, error) {
	info, err := r.mediaInfoExtractor.GetInfo("ffprobe", sourcePath)
	if err != nil {
		return nil, err
	}

	if err := videofile.ExtractAttachment("ffmpeg", sourcePath, targetHash, info, r.cacheDir, r.logger); err != nil {
		return nil, err
	}

	return info, nil
}

func (r *Repository) CacheDir() string {
	return r.cacheDir
}
