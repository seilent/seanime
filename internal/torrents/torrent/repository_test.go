package torrent

import (
	"seanime/internal/api/metadata"
	"seanime/internal/extension"
	"seanime/internal/torrents/nyaa"
	"seanime/internal/torrents/subsplease"
	"seanime/internal/util"
	"testing"
)

func getTestRepo(t *testing.T) *Repository {
	logger := util.NewLogger()
	metadataProvider := metadata.GetMockProvider(t)

	extensionBank := extension.NewUnifiedBank()

	extensionBank.Set("nyaa", extension.NewAnimeTorrentProviderExtension(&extension.Extension{
		ID:       "nyaa",
		Name:     "Nyaa",
		Version:  "1.0.0",
		Language: extension.LanguageGo,
		Type:     extension.TypeAnimeTorrentProvider,
		Author:   "Seanime",
	}, nyaa.NewProvider(logger, nyaa.CategoryAnimeEng)))

	extensionBank.Set("nyaa-sukebei", extension.NewAnimeTorrentProviderExtension(&extension.Extension{
		ID:       "nyaa-sukebei",
		Name:     "Nyaa Sukebei",
		Version:  "1.0.0",
		Language: extension.LanguageGo,
		Type:     extension.TypeAnimeTorrentProvider,
		Author:   "Seanime",
	}, nyaa.NewSukebeiProvider(logger)))

	extensionBank.Set("subsplease", extension.NewAnimeTorrentProviderExtension(&extension.Extension{
		ID:       "subsplease",
		Name:     "SubsPlease",
		Version:  "1.0.0",
		Language: extension.LanguageGo,
		Type:     extension.TypeAnimeTorrentProvider,
		Author:   "Seanime",
	}, subsplease.NewProvider(logger)))

	repo := NewRepository(&NewRepositoryOptions{
		Logger:           logger,
		MetadataProvider: metadataProvider,
	})

	repo.InitExtensionBank(extensionBank)

	repo.SetSettings(&RepositorySettings{
		DefaultAnimeProvider: ProviderNyaa,
	})

	return repo
}
