package mediastream

import (
	"os"
	"os/exec"
	"path/filepath"
	"seanime/internal/mediastream/videofile"
	"seanime/internal/util"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	chromeUA  = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	firefoxUA = "Mozilla/5.0 (X11; Linux x86_64; rv:159.0) Gecko/20100101 Firefox/159.0"
)

func audios(codecs ...string) *videofile.MediaInfo {
	info := &videofile.MediaInfo{}
	for i, codec := range codecs {
		info.Audios = append(info.Audios, videofile.Audio{Index: uint32(i), Codec: codec})
	}
	return info
}

func TestAudioNeedsTranscode(t *testing.T) {
	require.False(t, audioNeedsTranscode(nil))
	require.False(t, audioNeedsTranscode(&videofile.MediaInfo{}))
	require.False(t, audioNeedsTranscode(audios("aac", "eac3")))
	for _, codec := range []string{"ac3", "eac3", "dts", "truehd", "mlp"} {
		require.True(t, audioNeedsTranscode(audios(codec, "aac")), codec)
	}

	info := audios("aac", "eac3")
	info.Audios[1].IsDefault = true
	require.True(t, audioNeedsTranscode(info))
}

func TestDirectPlayNeedsMP4(t *testing.T) {
	require.False(t, directPlayNeedsMP4(chromeUA, audios("aac")))
	require.True(t, directPlayNeedsMP4(chromeUA, audios("eac3")))
	require.True(t, directPlayNeedsMP4(firefoxUA, audios("aac")))
	require.True(t, directPlayNeedsMP4(firefoxUA, audios("eac3")))
}

func TestRemuxArgs(t *testing.T) {
	require.Equal(t, []string{"-i", "in.mkv", "-c:v", "copy", "-c:a", "copy", "-sn", "-f", "mp4", "out.mp4"},
		remuxArgs("in.mkv", "out.mp4", nil))
	require.Equal(t, []string{"-i", "in.mkv", "-map", "0:v:0?", "-map", "0:a:0", "-c:v", "copy", "-c:a", "copy", "-sn", "-f", "mp4", "out.mp4"},
		remuxArgs("in.mkv", "out.mp4", audios("aac", "eac3")))

	info := audios("aac", "eac3")
	info.Audios[1].IsDefault = true
	require.Equal(t, []string{"-i", "in.mkv", "-map", "0:v:0?", "-map", "0:a:1", "-c:v", "copy", "-c:a", "aac", "-ac", "2", "-b:a", "192k", "-sn", "-f", "mp4", "out.mp4"},
		remuxArgs("in.mkv", "out.mp4", info))
}

func TestCreateRemuxedFileTranscodesEAC3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src.mkv")
	out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=24",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2:sample_rate=48000",
		"-f", "lavfi", "-i", "sine=frequency=880:duration=2:sample_rate=48000",
		"-map", "0", "-map", "1", "-map", "2", "-ac", "6",
		"-metadata:s:a:0", "language=jpn", "-metadata:s:a:1", "language=eng",
		"-c:v", "mpeg4", "-c:a", "eac3", src).CombinedOutput()
	require.NoError(t, err, string(out))

	r := &Repository{logger: util.NewLogger()}
	dst := filepath.Join(dir, "hash", "direct.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0755))
	info := audios("eac3", "eac3")
	info.Audios[1].IsDefault = true
	require.NoError(t, r.createRemuxedFile(src, dst, info))

	probe, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "a", "-show_entries", "stream=codec_name,channels:stream_tags=language",
		"-of", "csv=p=0", dst).Output()
	require.NoError(t, err)
	require.Equal(t, "aac,2,eng", strings.TrimSpace(string(probe)))
}
