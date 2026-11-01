package videofile

import (
	"os"
	"os/exec"
	"path/filepath"
	"seanime/internal/util"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestServeSubripAsASS(t *testing.T) {
	mi := &MediaInfo{
		Subtitles: []Subtitle{
			{Index: 0, Codec: "ass", Extension: lo.ToPtr("ass"), Link: lo.ToPtr("/0.ass")},
			{Index: 1, Codec: "subrip", Extension: lo.ToPtr("srt"), Link: lo.ToPtr("/1.srt")},
		},
		Fonts: []string{"Roboto-Medium.ttf", "arial.ttf"},
	}
	serveSubripAsASS(mi)
	serveSubripAsASS(mi)

	require.Equal(t, "ass", *mi.Subtitles[0].Extension)
	require.Equal(t, "/0.ass", *mi.Subtitles[0].Link)
	require.Equal(t, "ass", *mi.Subtitles[1].Extension)
	require.Equal(t, "/1.ass", *mi.Subtitles[1].Link)
	require.Equal(t, []string{"Roboto-Medium.ttf", "arial.ttf", "Roboto-MediumItalic.ttf"}, mi.Fonts)
	require.Equal(t, "copy", subtitleCodecArg(mi.Subtitles[0]))
	require.Equal(t, "ass", subtitleCodecArg(mi.Subtitles[1]))

	noSrt := &MediaInfo{Subtitles: []Subtitle{{Index: 0, Codec: "ass", Extension: lo.ToPtr("ass"), Link: lo.ToPtr("/0.ass")}}}
	serveSubripAsASS(noSrt)
	require.Empty(t, noSrt.Fonts)
}

func TestExtractAttachmentConvertsSubrip(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found")
	}

	dir := t.TempDir()
	srt := filepath.Join(dir, "in.srt")
	require.NoError(t, os.WriteFile(srt, []byte("1\n00:00:00,100 --> 00:00:01,000\n<i>Line one</i>\n\n2\n00:00:01,100 --> 00:00:01,900\n{\\an8}EPISODE 1\n\n"), 0644))
	src := filepath.Join(dir, "src.mkv")
	out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=24",
		"-i", srt, "-map", "0", "-map", "1", "-c:v", "mpeg4", "-c:s", "srt", src).CombinedOutput()
	require.NoError(t, err, string(out))

	mi, err := FfprobeGetInfo("", src, "hash")
	require.NoError(t, err)
	require.Equal(t, "srt", *mi.Subtitles[0].Extension)
	serveSubripAsASS(mi)

	cacheDir := filepath.Join(dir, "cache")
	require.NoError(t, ExtractAttachment("ffmpeg", src, "hash", mi, cacheDir, util.NewLogger()))

	ass, err := os.ReadFile(filepath.Join(GetFileSubsCacheDir(cacheDir, "hash"), "0.ass"))
	require.NoError(t, err)
	text := string(ass)
	require.Contains(t, text, "PlayResX: 640")
	require.Contains(t, text, "PlayResY: 360")
	require.Contains(t, text, subripStyle)
	require.NotContains(t, text, "Style: Default,Arial")
	require.Contains(t, text, `{\an8}EPISODE 1`)
	require.Contains(t, text, `{\i1}Line one{\i0}`)

	for _, font := range subripFonts {
		data, err := os.ReadFile(filepath.Join(GetFileAttCacheDir(cacheDir, "hash"), font.name))
		require.NoError(t, err)
		require.Equal(t, font.data, data)
	}

	require.True(t, subtitlesExtracted(GetFileSubsCacheDir(cacheDir, "hash"), mi))

	assPath := filepath.Join(GetFileSubsCacheDir(cacheDir, "hash"), "0.ass")
	require.NoError(t, os.WriteFile(assPath, []byte("[Script Info]\nPlayResX: 384\nPlayResY: 288\n\n[V4+ Styles]\nStyle: Default,Arial,16,&Hffffff,&Hffffff,&H0,&H0,0,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1\n"), 0644))
	require.NoError(t, os.Remove(filepath.Join(GetFileAttCacheDir(cacheDir, "hash"), "Roboto-Medium.ttf")))
	require.NoError(t, ExtractAttachment("ffmpeg", src, "hash", mi, cacheDir, util.NewLogger()))
	again, err := os.ReadFile(assPath)
	require.NoError(t, err)
	require.Contains(t, string(again), subripStyle)
	require.FileExists(t, filepath.Join(GetFileAttCacheDir(cacheDir, "hash"), "Roboto-Medium.ttf"))
}

func TestExtractAttachmentMigratesSrtCache(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found")
	}

	dir := t.TempDir()
	srt := filepath.Join(dir, "in.srt")
	require.NoError(t, os.WriteFile(srt, []byte("1\n00:00:00,100 --> 00:00:01,000\nHello\n\n"), 0644))
	src := filepath.Join(dir, "src.mkv")
	out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=24",
		"-i", srt, "-map", "0", "-map", "1", "-c:v", "mpeg4", "-c:s", "srt", src).CombinedOutput()
	require.NoError(t, err, string(out))

	cacheDir := filepath.Join(dir, "cache")
	subsDir := GetFileSubsCacheDir(cacheDir, "hash")
	require.NoError(t, os.MkdirAll(subsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subsDir, "0.srt"), []byte("old"), 0644))

	mi, err := FfprobeGetInfo("", src, "hash")
	require.NoError(t, err)
	serveSubripAsASS(mi)
	require.False(t, subtitlesExtracted(subsDir, mi))
	require.NoError(t, ExtractAttachment("ffmpeg", src, "hash", mi, cacheDir, util.NewLogger()))
	ass, err := os.ReadFile(filepath.Join(subsDir, "0.ass"))
	require.NoError(t, err)
	require.Contains(t, string(ass), subripStyle)
	require.Contains(t, string(ass), "Hello")
}

func TestRestyleSubripASSKeepsCRLF(t *testing.T) {
	p := filepath.Join(t.TempDir(), "0.ass")
	require.NoError(t, os.WriteFile(p, []byte("[Script Info]\r\nPlayResX: 384\r\nPlayResY: 288\r\n\r\n[V4+ Styles]\r\nStyle: Default,Arial,16,&Hffffff,&Hffffff,&H0,&H0,0,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1\r\n"), 0644))
	require.NoError(t, restyleSubripASS(p))
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, "[Script Info]\r\nPlayResX: 640\r\nPlayResY: 360\r\n\r\n[V4+ Styles]\r\n"+subripStyle+"\r\n", string(data))
}
