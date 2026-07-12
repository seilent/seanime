package subsplease

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	hibiketorrent "seanime/internal/extension/hibike/torrent"

	"github.com/rs/zerolog"
)

const (
	ProviderName = "subsplease"
	showsURL     = "https://subsplease.org/shows/"
	apiURL       = "https://subsplease.org/api/"
)

var sidRegex = regexp.MustCompile(`sid="(\d+)"`)
var infoHashRegex = regexp.MustCompile(`btih:([0-9a-zA-Z]+)`)
var showLinkRegex = regexp.MustCompile(`/shows/([^"]+)"\s+title="([^"]+)"`)

type Provider struct {
	logger *zerolog.Logger
	// slug -> sid cache
	mu       sync.RWMutex
	sidCache map[string]string
	// shows index: normalized title -> slug
	showsIndex map[string]string
}

func NewProvider(logger *zerolog.Logger) hibiketorrent.AnimeProvider {
	return &Provider{
		logger:   logger,
		sidCache: make(map[string]string),
	}
}

func (p *Provider) GetSettings() hibiketorrent.AnimeProviderSettings {
	return hibiketorrent.AnimeProviderSettings{
		Type:           hibiketorrent.AnimeProviderTypeMain,
		CanSmartSearch: true,
		SupportsAdult:  false,
		SmartSearchFilters: []hibiketorrent.AnimeProviderSmartSearchFilter{
			hibiketorrent.AnimeProviderSmartSearchFilterEpisodeNumber,
		},
	}
}

func (p *Provider) Search(opts hibiketorrent.AnimeSearchOptions) ([]*hibiketorrent.AnimeTorrent, error) {
	// Search by slug directly
	slug := TitleToSlug(opts.Query)
	return p.fetchShowEpisodes(slug, 0, false, 0)
}

func (p *Provider) SmartSearch(opts hibiketorrent.AnimeSmartSearchOptions) ([]*hibiketorrent.AnimeTorrent, error) {
	// Build candidate slugs from titles and synonyms
	slugs := []string{}

	// Detect season from titles
	season := 0
	romSlug := TitleToSlug(opts.Media.RomajiTitle)
	slugs = append(slugs, romSlug)
	if s := extractTrailingSeason(opts.Media.RomajiTitle); s > 0 {
		season = s
	}
	if opts.Media.EnglishTitle != nil && *opts.Media.EnglishTitle != "" {
		slugs = append(slugs, TitleToSlug(*opts.Media.EnglishTitle))
		if s := extractTrailingSeason(*opts.Media.EnglishTitle); s > 0 && season == 0 {
			season = s
		}
	}
	for _, syn := range opts.Media.Synonyms {
		if isLatin(syn) {
			slug := TitleToSlug(syn)
			slugs = append(slugs, slug)
			// Also try without trailing number + "-s{N}"
			if season > 0 {
				base := strings.TrimRight(slug, "0123456789")
				base = strings.TrimRight(base, "-")
				if base != "" {
					slugs = append(slugs, fmt.Sprintf("%s-s%d", base, season))
				}
			}
		}
	}

	// Deduplicate
	seen := map[string]bool{}
	unique := []string{}
	for _, s := range slugs {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}

	// Try each slug until one works
	for _, slug := range unique {
		results, err := p.fetchShowEpisodes(slug, opts.EpisodeNumber, opts.Batch, opts.Media.EpisodeCount)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	// Fallback: fuzzy-match against the SubsPlease shows index
	if slug := p.findSlugFromShowsIndex(opts.Media); slug != "" {
		results, err := p.fetchShowEpisodes(slug, opts.EpisodeNumber, opts.Batch, opts.Media.EpisodeCount)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	return nil, fmt.Errorf("show not found on SubsPlease")
}

func (p *Provider) GetLatest() ([]*hibiketorrent.AnimeTorrent, error) {
	return nil, nil
}

func (p *Provider) GetTorrentInfoHash(torrent *hibiketorrent.AnimeTorrent) (string, error) {
	if torrent.InfoHash != "" {
		return torrent.InfoHash, nil
	}
	matches := infoHashRegex.FindStringSubmatch(torrent.MagnetLink)
	if len(matches) > 1 {
		hash := matches[1]
		if len(hash) == 32 {
			decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(hash))
			if err == nil {
				return strings.ToLower(fmt.Sprintf("%x", decoded)), nil
			}
		}
		return strings.ToLower(hash), nil
	}
	return "", fmt.Errorf("no info hash found")
}

func (p *Provider) GetTorrentMagnetLink(torrent *hibiketorrent.AnimeTorrent) (string, error) {
	return torrent.MagnetLink, nil
}

// fetchShowEpisodes gets episodes from the SubsPlease API for a given slug
func (p *Provider) fetchShowEpisodes(slug string, episodeNumber int, batch bool, episodeCount int) ([]*hibiketorrent.AnimeTorrent, error) {
	sid, err := p.getSid(slug)
	if err != nil {
		return nil, err
	}

	// Call the show API
	u := fmt.Sprintf("%s?f=show&tz=UTC&sid=%s", apiURL, sid)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var showData struct {
		Episode map[string]struct {
			ReleaseDate string `json:"release_date"`
			Show        string `json:"show"`
			Episode     string `json:"episode"`
			Downloads   []struct {
				Res     string `json:"res"`
				Torrent string `json:"torrent"`
				Magnet  string `json:"magnet"`
			} `json:"downloads"`
		} `json:"episode"`
	}
	if err := json.Unmarshal(body, &showData); err != nil {
		return nil, err
	}

	var results []*hibiketorrent.AnimeTorrent
	for _, ep := range showData.Episode {
		// Filter by episode if specified (skip when batch)
		if !batch && episodeNumber > 0 {
			epNum, _ := strconv.Atoi(ep.Episode)
			if epNum != episodeNumber {
				continue
			}
		}

		// Find 1080p download
		for _, dl := range ep.Downloads {
			if dl.Res != "1080" {
				continue
			}

			infoHash := ""
			if matches := infoHashRegex.FindStringSubmatch(dl.Magnet); len(matches) > 1 {
				hash := matches[1]
				if len(hash) == 32 {
					// Base32 encoded — decode to hex
					decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(hash))
					if err == nil {
						infoHash = strings.ToLower(fmt.Sprintf("%x", decoded))
					}
				} else {
					infoHash = strings.ToLower(hash)
				}
			}

			date := ""
			if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", ep.ReleaseDate); err == nil {
				date = t.Format(time.RFC3339)
			}

			epNum, _ := strconv.Atoi(ep.Episode)
			name := fmt.Sprintf("[SubsPlease] %s - %s (1080p)", ep.Show, ep.Episode)

			results = append(results, &hibiketorrent.AnimeTorrent{
				Name:          name,
				Date:          date,
				Link:          dl.Torrent,
				DownloadUrl:   dl.Torrent,
				MagnetLink:    dl.Magnet,
				InfoHash:      infoHash,
				Resolution:    "1080",
				EpisodeNumber: epNum,
				Provider:      ProviderName,
			})
			break
		}
	}

	// If batch requested and all episodes are available, mark as batch
	if batch && episodeCount > 0 && len(results) >= episodeCount {
		showName := ""
		if len(results) > 0 {
			// Use show name from first result
			showName = results[0].Name
		}
		for _, r := range results {
			r.IsBatch = true
			_ = showName
		}
	}

	return results, nil
}

// normalizeTitle strips non-alphanumeric chars and lowercases for fuzzy matching.
func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// loadShowsIndex fetches and caches the SubsPlease shows index (normalized title -> slug).
func (p *Provider) loadShowsIndex() map[string]string {
	p.mu.RLock()
	if p.showsIndex != nil {
		idx := p.showsIndex
		p.mu.RUnlock()
		return idx
	}
	p.mu.RUnlock()

	resp, err := http.Get(showsURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	idx := make(map[string]string)
	for _, m := range showLinkRegex.FindAllStringSubmatch(string(body), -1) {
		slug := strings.TrimSuffix(m[1], "/")
		title := m[2]
		idx[normalizeTitle(title)] = slug
	}

	p.mu.Lock()
	p.showsIndex = idx
	p.mu.Unlock()
	return idx
}

// findSlugFromShowsIndex fuzzy-matches the media's titles against the shows index.
func (p *Provider) findSlugFromShowsIndex(media hibiketorrent.Media) string {
	idx := p.loadShowsIndex()
	if idx == nil {
		return ""
	}

	candidates := []string{media.RomajiTitle}
	if media.EnglishTitle != nil {
		candidates = append(candidates, *media.EnglishTitle)
	}
	candidates = append(candidates, media.Synonyms...)

	for _, c := range candidates {
		if !isLatin(c) {
			continue
		}
		norm := normalizeTitle(c)
		if norm == "" {
			continue
		}
		if slug, ok := idx[norm]; ok {
			return slug
		}
	}
	return ""
}

// getSid resolves a slug to a numeric sid, using cache
func (p *Provider) getSid(slug string) (string, error) {
	p.mu.RLock()
	if sid, ok := p.sidCache[slug]; ok {
		p.mu.RUnlock()
		return sid, nil
	}
	p.mu.RUnlock()

	// Fetch the show page to extract sid
	resp, err := http.Get(showsURL + slug + "/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("show page not found: %s", slug)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	matches := sidRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("sid not found on page: %s", slug)
	}

	sid := string(matches[1])
	p.mu.Lock()
	p.sidCache[slug] = sid
	p.mu.Unlock()

	return sid, nil
}

// titleToSlug converts a title to a SubsPlease URL slug
func TitleToSlug(title string) string {
	s := strings.ToLower(title)
	// Remove common punctuation
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == ' ', r == '-':
			return '-'
		default:
			return -1
		}
	}, s)
	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return url.PathEscape(s)
}

// isLatin checks if a string contains only latin/ASCII characters
func isLatin(s string) bool {
	for _, r := range s {
		if r > 0x024F {
			return false
		}
	}
	return true
}

// extractTrailingSeason extracts a season number from "Season N" or trailing " N" in a title
func extractTrailingSeason(title string) int {
	lower := strings.ToLower(title)
	// Match "season N"
	re := regexp.MustCompile(`season\s*(\d+)`)
	if m := re.FindStringSubmatch(lower); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	// Match trailing number like "LasTame 2"
	re2 := regexp.MustCompile(`\s(\d+)$`)
	if m := re2.FindStringSubmatch(strings.TrimSpace(title)); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}
