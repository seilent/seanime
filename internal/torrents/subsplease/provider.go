package subsplease

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	hibiketorrent "seanime/internal/extension/hibike/torrent"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog"
)

const (
	ProviderName = "subsplease"
	rssURL       = "https://subsplease.org/rss/?t&r=1080"
)

// episodeRegex extracts episode number from SubsPlease title format:
// [SubsPlease] Show Name - 08 (1080p) [CRC32]
var episodeRegex = regexp.MustCompile(`- (\d+) \(`)

// infoHashRegex extracts btih from magnet link
var infoHashRegex = regexp.MustCompile(`btih:([0-9a-fA-F]+)`)

type Provider struct {
	logger *zerolog.Logger
}

func NewProvider(logger *zerolog.Logger) hibiketorrent.AnimeProvider {
	return &Provider{logger: logger}
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
	return p.fetchAndFilter(opts.Query, 0, false)
}

func (p *Provider) SmartSearch(opts hibiketorrent.AnimeSmartSearchOptions) ([]*hibiketorrent.AnimeTorrent, error) {
	// Build search terms from media titles
	titles := []string{strings.ToLower(opts.Media.RomajiTitle)}
	if opts.Media.EnglishTitle != nil && *opts.Media.EnglishTitle != "" {
		titles = append(titles, strings.ToLower(*opts.Media.EnglishTitle))
	}
	for _, syn := range opts.Media.Synonyms {
		// Only use romaji/latin synonyms
		if isLatin(syn) {
			titles = append(titles, strings.ToLower(syn))
		}
	}

	var results []*hibiketorrent.AnimeTorrent
	seen := make(map[string]struct{})

	feed, err := p.parseFeed()
	if err != nil {
		return nil, err
	}

	for _, item := range feed.Items {
		lowerTitle := strings.ToLower(item.Title)
		matched := false
		for _, t := range titles {
			if strings.Contains(lowerTitle, t) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// Filter by episode number if specified
		if opts.EpisodeNumber > 0 {
			ep := extractEpisode(item.Title)
			if ep != opts.EpisodeNumber {
				continue
			}
		}

		torrent := p.itemToTorrent(item)
		if _, exists := seen[torrent.Link]; exists {
			continue
		}
		seen[torrent.Link] = struct{}{}
		results = append(results, torrent)
	}

	return results, nil
}

func (p *Provider) GetLatest() ([]*hibiketorrent.AnimeTorrent, error) {
	feed, err := p.parseFeed()
	if err != nil {
		return nil, err
	}
	var results []*hibiketorrent.AnimeTorrent
	for _, item := range feed.Items {
		results = append(results, p.itemToTorrent(item))
	}
	return results, nil
}

func (p *Provider) GetTorrentInfoHash(torrent *hibiketorrent.AnimeTorrent) (string, error) {
	if torrent.InfoHash != "" {
		return torrent.InfoHash, nil
	}
	matches := infoHashRegex.FindStringSubmatch(torrent.MagnetLink)
	if len(matches) > 1 {
		return strings.ToLower(matches[1]), nil
	}
	return "", fmt.Errorf("no info hash found")
}

func (p *Provider) GetTorrentMagnetLink(torrent *hibiketorrent.AnimeTorrent) (string, error) {
	return torrent.MagnetLink, nil
}

func (p *Provider) parseFeed() (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	return fp.ParseURL(rssURL)
}

func (p *Provider) fetchAndFilter(query string, episodeNumber int, batch bool) ([]*hibiketorrent.AnimeTorrent, error) {
	feed, err := p.parseFeed()
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	var results []*hibiketorrent.AnimeTorrent
	for _, item := range feed.Items {
		if query != "" && !strings.Contains(strings.ToLower(item.Title), lowerQuery) {
			continue
		}
		results = append(results, p.itemToTorrent(item))
	}
	return results, nil
}

func (p *Provider) itemToTorrent(item *gofeed.Item) *hibiketorrent.AnimeTorrent {
	date := ""
	if item.PublishedParsed != nil {
		date = item.PublishedParsed.Format(time.RFC3339)
	}

	infoHash := ""
	magnetLink := ""
	if strings.HasPrefix(item.Link, "magnet:") {
		magnetLink = item.Link
		matches := infoHashRegex.FindStringSubmatch(item.Link)
		if len(matches) > 1 {
			infoHash = strings.ToLower(matches[1])
		}
	}

	return &hibiketorrent.AnimeTorrent{
		Name:          item.Title,
		Date:          date,
		Link:          item.Link,
		MagnetLink:    magnetLink,
		InfoHash:      infoHash,
		Resolution:    "1080",
		EpisodeNumber: extractEpisode(item.Title),
		Provider:      ProviderName,
	}
}

func extractEpisode(title string) int {
	matches := episodeRegex.FindStringSubmatch(title)
	if len(matches) > 1 {
		ep := 0
		fmt.Sscanf(matches[1], "%d", &ep)
		return ep
	}
	return 0
}

// isLatin checks if a string contains only latin/ASCII characters (romaji synonyms)
func isLatin(s string) bool {
	for _, r := range s {
		if r > 0x024F { // Beyond Latin Extended-B
			return false
		}
	}
	return true
}
