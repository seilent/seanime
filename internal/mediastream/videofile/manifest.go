package videofile

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type SubsManifest struct {
	Subtitles []Subtitle `json:"subtitles"`
	Fonts     []string   `json:"fonts"`
}

func SubsManifestPath(cacheDir, hash string) string {
	return filepath.Join(cacheDir, "videofiles", hash, "subs.json")
}

func WriteSubsManifest(cacheDir, hash string, subs []Subtitle, fonts []string) error {
	dir := filepath.Join(cacheDir, "videofiles", hash)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	m := SubsManifest{Subtitles: subs, Fonts: fonts}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(SubsManifestPath(cacheDir, hash), data, 0644)
}

func ReadSubsManifest(cacheDir, hash string) (*SubsManifest, error) {
	data, err := os.ReadFile(SubsManifestPath(cacheDir, hash))
	if err != nil {
		return nil, err
	}
	var m SubsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
