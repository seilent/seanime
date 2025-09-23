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
	"seanime/internal/util"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Direct
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (r *Repository) ServeEchoFile(c echo.Context, rawFilePath string, clientId string, libraryPaths []string) error {
	// Unescape the file path, ignore errors
	filePath, _ := url.PathUnescape(rawFilePath)

	// If the file path is base64 encoded, decode it
	if util.IsBase64(rawFilePath) {
		var err error
		filePath, err = util.Base64DecodeStr(rawFilePath)
		if err != nil {
			// this shouldn't happen, but just in case IsBase64 is wrong
			filePath, _ = url.PathUnescape(rawFilePath)
		}
	}

	// Make sure the file is in the library directories
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
	// Content disposition
	filename := filepath.Base(filePath)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	return c.File(filePath)
}

func (r *Repository) ServeEchoDirectPlay(c echo.Context, clientId string) error {

	if !r.IsInitialized() {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "Module not initialized")
		return errors.New("module not initialized")
	}

	// Get current media for this client
	mediaContainer, found := r.playbackManager.currentMediaContainers[clientId].Get()
	if !found {
		r.wsEventManager.SendEvent(events.MediastreamShutdownStream, "no file has been loaded for client")
		return errors.New("no file has been loaded for client")
	}

	filePath := mediaContainer.Filepath

	// Check if browser doesn't support MKV and file is MKV, then remux to MP4
	userAgent := c.Request().UserAgent()
	needsMP4 := !browserSupportsMKV(userAgent)
	isMKV := filepath.Ext(filePath) == ".mkv"

	if needsMP4 && isMKV {
		r.logger.Debug().Str("userAgent", userAgent).Str("filepath", filePath).Msg("mediastream: Browser + MKV detected, remuxing to MP4")

		// Create remuxed file path
		remuxedPath := filepath.Join(r.cacheDir, mediaContainer.Hash+".mp4")

		// Check if remuxed file exists, if not create it
		if _, err := os.Stat(remuxedPath); os.IsNotExist(err) {
			r.logger.Info().Str("source", filePath).Str("target", remuxedPath).Msg("mediastream: Creating remuxed MP4 file")

			// Create MP4 file using ffmpeg remuxing
			if err := r.createRemuxedFile(filePath, remuxedPath); err != nil {
				r.logger.Error().Err(err).Msg("mediastream: Failed to create remuxed file, serving original")
				// Fallback to original file if remuxing fails
			} else {
				r.logger.Debug().Str("remuxedPath", remuxedPath).Msg("mediastream: Successfully created remuxed file")
				filePath = remuxedPath
			}
		} else {
			// Check if cached file is still valid (less than 1 week old)
			if fileInfo, err := os.Stat(remuxedPath); err == nil {
				if time.Since(fileInfo.ModTime()) > 7*24*time.Hour {
					r.logger.Debug().Str("remuxedPath", remuxedPath).Msg("mediastream: Cached file expired, recreating")
					if err := r.createRemuxedFile(filePath, remuxedPath); err != nil {
						r.logger.Error().Err(err).Msg("mediastream: Failed to recreate remuxed file, using cached version")
					}
				} else {
					r.logger.Debug().Str("remuxedPath", remuxedPath).Msg("mediastream: Using cached remuxed file")
				}
				filePath = remuxedPath
			} else {
				r.logger.Error().Err(err).Msg("mediastream: Failed to stat cached file, serving original")
			}
		}
	}

	if c.Request().Method == http.MethodHead {
		r.logger.Trace().Msg("mediastream: Received HEAD request for direct play")

		// Get the file size
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			r.logger.Error().Msg("mediastream: Failed to get file info")
			return c.NoContent(http.StatusInternalServerError)
		}

		// Set the content length
		c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		c.Response().Header().Set("Content-Type", "video/mp4")
		c.Response().Header().Set("Accept-Ranges", "bytes")
		filename := filepath.Base(filePath)
		if needsMP4 && isMKV {
			filename = filename[:len(filename)-4] + ".mp4" // Change extension to .mp4
		}
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
		return c.NoContent(http.StatusOK)
	}

	return c.File(filePath)
}

// browserSupportsMKV checks if the browser supports MKV playback natively
func browserSupportsMKV(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	ua := strings.ToLower(userAgent)

	// Firefox doesn't support MKV natively
	if strings.Contains(ua, "firefox") {
		return false
	}

	// Safari on iOS < 14 and desktop Safari < 14 have limited MKV support
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		return false
	}

	// Mobile browsers often have limited MKV support
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") {
		return false
	}

	// Most modern Chrome-based browsers and Edge support MKV
	return true
}

// createRemuxedFile creates an MP4 version of the MKV file using ffmpeg
func (r *Repository) createRemuxedFile(inputPath, outputPath string) error {
	// Create temporary file first to avoid serving incomplete files
	tempPath := outputPath + ".tmp"

	// Clean up any existing temporary file
	os.Remove(tempPath)

	// Use ffmpeg to remux MKV to MP4 without re-encoding
	// Note: MP4 doesn't support SRT subtitles directly, so we'll skip them for browser compatibility
	// The browser will handle subtitles through the existing subtitle extraction system
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:v", "copy", "-c:a", "copy", "-sn", "-f", "mp4", tempPath)

	// Capture output for debugging
	var stderr, stdout strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	r.logger.Debug().Str("input", inputPath).Str("output", tempPath).Msg("mediastream: Starting ffmpeg remux")

	if err := cmd.Run(); err != nil {
		r.logger.Error().Err(err).Str("stderr", stderr.String()).Str("stdout", stdout.String()).Msg("mediastream: ffmpeg remux failed")
		// Clean up temporary file
		os.Remove(tempPath)
		return fmt.Errorf("ffmpeg remux failed: %w", err)
	}

	// Rename temporary file to final location
	if err := os.Rename(tempPath, outputPath); err != nil {
		r.logger.Error().Err(err).Msg("mediastream: Failed to rename temporary file")
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	r.logger.Debug().Str("input", inputPath).Str("output", outputPath).Msg("mediastream: Successfully created remuxed file")
	return nil
}
