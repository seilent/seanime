package videofile

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"seanime/internal/util"
	"seanime/internal/util/crashlog"
)

func GetFileSubsCacheDir(outDir string, hash string) string {
	return filepath.Join(outDir, "videofiles", hash, "/subs")
}

func GetFileAttCacheDir(outDir string, hash string) string {
	return filepath.Join(outDir, "videofiles", hash, "/att")
}

func ExtractAttachment(ffmpegPath string, path string, hash string, mediaInfo *MediaInfo, cacheDir string, logger *zerolog.Logger) (err error) {
	logger.Debug().Str("hash", hash).Msgf("videofile: Starting media attachment extraction")

	attachmentPath := GetFileAttCacheDir(cacheDir, hash)
	subsPath := GetFileSubsCacheDir(cacheDir, hash)
	_ = os.MkdirAll(attachmentPath, 0755)
	_ = os.MkdirAll(subsPath, 0755)

	if subtitlesExtracted(subsPath, mediaInfo) {
		logger.Debug().Str("hash", hash).Msgf("videofile: Attachments already extracted")
		finishSubrip(subsPath, attachmentPath, mediaInfo, logger)
		return nil
	}

	for _, sub := range mediaInfo.Subtitles {
		if sub.Extension == nil || *sub.Extension == "" {
			logger.Error().Msgf("videofile: Subtitle format is not supported")
			return fmt.Errorf("videofile: Unsupported subtitle format")
		}
	}

	// Instantiate a new crash logger
	crashLogger := crashlog.GlobalCrashLogger.InitArea("ffmpeg")
	defer crashLogger.Close()

	crashLogger.LogInfof("Extracting attachments from %s", path)

	// DEVNOTE: All paths fed into this command should be absolute
	cmd := util.NewCmdCtx(
		context.Background(),
		ffmpegPath,
		"-dump_attachment:t", "",
		// override old attachments
		"-y",
		"-i", path,
	)
	// The working directory for the command is the attachment directory
	cmd.Dir = attachmentPath

	for _, sub := range mediaInfo.Subtitles {
		if ext := sub.Extension; ext != nil {
			cmd.Args = append(
				cmd.Args,
				"-map", fmt.Sprintf("0:s:%d", sub.Index),
				"-c:s", subtitleCodecArg(sub),
				fmt.Sprintf("%s/%d.%s", subsPath, sub.Index, *ext),
			)
		}
	}

	cmd.Stdout = crashLogger.Stdout()
	cmd.Stderr = crashLogger.Stdout()
	err = cmd.Run()
	if err != nil {
		logger.Error().Err(err).Msgf("videofile: Error starting FFmpeg")
		crashlog.GlobalCrashLogger.WriteAreaLogToFile(crashLogger)
		return err
	}

	finishSubrip(subsPath, attachmentPath, mediaInfo, logger)
	return nil
}

func subtitlesExtracted(subsPath string, mediaInfo *MediaInfo) bool {
	for _, sub := range mediaInfo.Subtitles {
		if sub.Extension == nil {
			return false
		}
		if _, err := os.Stat(fmt.Sprintf("%s/%d.%s", subsPath, sub.Index, *sub.Extension)); err != nil {
			return false
		}
	}
	return true
}
