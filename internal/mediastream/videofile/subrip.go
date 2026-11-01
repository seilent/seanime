package videofile

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
)

//go:embed fonts/Roboto-Medium.ttf
var robotoMedium []byte

//go:embed fonts/Roboto-MediumItalic.ttf
var robotoMediumItalic []byte

var subripFonts = []struct {
	name string
	data []byte
}{
	{"Roboto-Medium.ttf", robotoMedium},
	{"Roboto-MediumItalic.ttf", robotoMediumItalic},
}

const subripStyle = "Style: Default,Roboto Medium,26,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,1.3,0,2,20,20,23,1"

var (
	assPlayResX     = regexp.MustCompile(`(?m)^PlayResX:[^\r\n]*`)
	assPlayResY     = regexp.MustCompile(`(?m)^PlayResY:[^\r\n]*`)
	assDefaultStyle = regexp.MustCompile(`(?m)^Style: Default,[^\r\n]*`)
)

func hasSubrip(mi *MediaInfo) bool {
	return mi != nil && slices.ContainsFunc(mi.Subtitles, func(sub Subtitle) bool { return sub.Codec == "subrip" })
}

func serveSubripAsASS(mi *MediaInfo) {
	if !hasSubrip(mi) {
		return
	}
	for i := range mi.Subtitles {
		sub := &mi.Subtitles[i]
		if sub.Codec != "subrip" {
			continue
		}
		sub.Extension = lo.ToPtr("ass")
		sub.Link = lo.ToPtr(fmt.Sprintf("/%d.ass", sub.Index))
	}
	for _, font := range subripFonts {
		if !slices.Contains(mi.Fonts, font.name) {
			mi.Fonts = append(mi.Fonts, font.name)
		}
	}
}

func subtitleCodecArg(sub Subtitle) string {
	if sub.Codec == "subrip" && sub.Extension != nil && *sub.Extension == "ass" {
		return "ass"
	}
	return "copy"
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func restyleSubripASS(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	styled := assPlayResX.ReplaceAll(data, []byte("PlayResX: 640"))
	styled = assPlayResY.ReplaceAll(styled, []byte("PlayResY: 360"))
	styled = assDefaultStyle.ReplaceAll(styled, []byte(subripStyle))
	if bytes.Equal(styled, data) {
		return nil
	}
	return writeFileAtomic(path, styled)
}

func writeSubripFonts(attachmentPath string) error {
	for _, font := range subripFonts {
		p := filepath.Join(attachmentPath, font.name)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := writeFileAtomic(p, font.data); err != nil {
			return err
		}
	}
	return nil
}

func finishSubrip(subsPath, attachmentPath string, mi *MediaInfo, logger *zerolog.Logger) {
	if !hasSubrip(mi) {
		return
	}
	for _, sub := range mi.Subtitles {
		if subtitleCodecArg(sub) != "ass" {
			continue
		}
		if err := restyleSubripASS(filepath.Join(subsPath, fmt.Sprintf("%d.ass", sub.Index))); err != nil {
			logger.Warn().Err(err).Uint32("index", sub.Index).Msg("videofile: Failed to restyle converted subtitle")
		}
	}
	if err := writeSubripFonts(attachmentPath); err != nil {
		logger.Warn().Err(err).Msg("videofile: Failed to write subtitle fonts")
	}
}
