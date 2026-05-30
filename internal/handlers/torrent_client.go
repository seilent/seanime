package handlers

import (
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	hibiketorrent "seanime/internal/extension/hibike/torrent"
	"seanime/internal/torrent_clients/torrent_client"
	"strconv"
	"strings"

	"github.com/5rahim/habari"
	"github.com/labstack/echo/v4"
)

// HandleGetActiveTorrentList
//
//	@summary returns all active torrents.
//	@desc This handler is used by the client to display the active torrents.
//
//	@route /api/v1/torrent-client/list [GET]
//	@returns []torrent_client.Torrent
func (h *Handler) HandleGetActiveTorrentList(c echo.Context) error {

	// Get torrent list
	res, err := h.App.TorrentClientRepository.GetActiveTorrents()
	// If an error occurred, try to start the torrent client and get the list again
	// DEVNOTE: We try to get the list first because this route is called repeatedly by the client.
	if err != nil {
		ok := h.App.TorrentClientRepository.Start()
		if !ok {
			return h.RespondWithError(c, errors.New("could not start torrent client, verify your settings"))
		}
		res, err = h.App.TorrentClientRepository.GetActiveTorrents()
	}

	// Filter to only show seanime-tracked, non-seeding torrents
	tracked, _ := h.App.Database.GetIncompleteIntentHashes()
	if tracked != nil {
		filtered := make([]*torrent_client.Torrent, 0)
		for _, t := range res {
			if _, ok := tracked[strings.ToLower(t.Hash)]; ok && t.Status != torrent_client.TorrentStatusSeeding {
				filtered = append(filtered, t)
			}
		}
		res = filtered
	}

	return h.RespondWithData(c, res)

}

// HandleTorrentClientAction
//
//	@summary performs an action on a torrent.
//	@desc This handler is used to pause, resume or remove a torrent.
//	@route /api/v1/torrent-client/action [POST]
//	@returns bool
func (h *Handler) HandleTorrentClientAction(c echo.Context) error {

	type body struct {
		Hash   string `json:"hash"`
		Action string `json:"action"`
		Dir    string `json:"dir"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	if b.Hash == "" || b.Action == "" {
		return h.RespondWithError(c, errors.New("missing arguments"))
	}

	switch b.Action {
	case "pause":
		err := h.App.TorrentClientRepository.PauseTorrents([]string{b.Hash})
		if err != nil {
			return h.RespondWithError(c, err)
		}
	case "resume":
		err := h.App.TorrentClientRepository.ResumeTorrents([]string{b.Hash})
		if err != nil {
			return h.RespondWithError(c, err)
		}
	case "remove":
		err := h.App.TorrentClientRepository.RemoveTorrents([]string{b.Hash})
		if err != nil {
			return h.RespondWithError(c, err)
		}
	}

	return h.RespondWithData(c, true)

}

// HandleTorrentClientDownload
//
//	@summary adds torrents to the torrent client.
//	@desc It fetches the magnets from the provided URLs and adds them to the torrent client.
//	@desc If smart select is enabled, it will try to select the best torrent based on the missing episodes.
//	@route /api/v1/torrent-client/download [POST]
//	@returns bool
func (h *Handler) HandleTorrentClientDownload(c echo.Context) error {

	type body struct {
		Torrents    []hibiketorrent.AnimeTorrent `json:"torrents"`
		Destination string                       `json:"destination"`
		SmartSelect struct {
			Enabled               bool  `json:"enabled"`
			MissingEpisodeNumbers []int `json:"missingEpisodeNumbers"`
		} `json:"smartSelect"`
		Media               *anilist.BaseAnime `json:"media"`
		DeleteExistingFiles bool               `json:"deleteExistingFiles"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	if b.Destination == "" {
		return h.RespondWithError(c, errors.New("destination not found"))
	}

	if !filepath.IsAbs(b.Destination) {
		return h.RespondWithError(c, errors.New("destination path must be absolute"))
	}

	// Check that the destination path is a library path
	//libraryPaths, err := h.App.Database.GetAllLibraryPathsFromSettings()
	//if err != nil {
	//	return h.RespondWithError(c, err)
	//}
	//isInLibrary := util.IsSubdirectoryOfAny(libraryPaths, b.Destination)
	//if !isInLibrary {
	//	return h.RespondWithError(c, errors.New("destination path is not a library path"))
	//}

	// try to start torrent client if it's not running
	ok := h.App.TorrentClientRepository.Start()
	if !ok {
		return h.RespondWithError(c, errors.New("could not contact torrent client, verify your settings or make sure it's running"))
	}

	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	completeAnime, err := userPlatform.GetAnimeWithRelations(c.Request().Context(), b.Media.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Check if downloading a batch torrent
	isBatchDownload := len(b.Torrents) == 1 && b.Torrents[0].IsBatch

	// For batch downloads or explicit request, delete existing files
	// Smart-select is additive (per-episode replace handled by downloadmonitor), so skip blanket wipe
	if (isBatchDownload || b.DeleteExistingFiles) && !b.SmartSelect.Enabled {
		if b.Media != nil {
			// Get all local files for this media
			localFiles, err := db_bridge.GetLocalFilesByMediaId(h.App.Database, b.Media.ID)
			if err != nil {
				// Log but don't block the download
				h.App.Logger.Warn().Err(err).Int("mediaId", b.Media.ID).
					Msg("Failed to get local files for cleanup")
			} else if len(localFiles) > 0 {
				// Attempt to delete files
				err = h.App.FileCleanupManager.CleanupFiles(localFiles)
				if err != nil {
					h.App.Logger.Warn().Err(err).
						Msg("Some files could not be deleted immediately, added to cleanup queue")
				}
			}
			// Purge all existing mappings for this media so new download starts fresh
			_ = h.App.Database.DeleteGlobalMappingsByAniListID(b.Media.ID)
		}
	} else if b.Media != nil {
		// Per-episode SYNC (SubsPlease only): delete existing local files for the specific
		// episodes being downloaded, replacing non-SubsPlease releases with SubsPlease.
		isSubsPlease := len(b.Torrents) > 0 && b.Torrents[0].Provider == "subsplease"
		if isSubsPlease {
			localFiles, _ := db_bridge.GetLocalFilesByMediaId(h.App.Database, b.Media.ID)
			downloadEps := make(map[int]bool)
			for _, t := range b.Torrents {
				if t.EpisodeNumber > 0 {
					downloadEps[t.EpisodeNumber] = true
				}
			}
			for _, lf := range localFiles {
				if downloadEps[lf.EpisodeNumber] && !strings.Contains(lf.LocalFilePath, "[SubsPlease]") {
					_ = os.Remove(lf.LocalFilePath)
					_ = h.App.Database.DeleteGlobalMapping(lf.LocalFilePath)
				}
			}
		}
	}

	if b.SmartSelect.Enabled {
		if len(b.Torrents) > 1 {
			return h.RespondWithError(c, errors.New("smart select is not supported for multiple torrents"))
		}

		// smart select
		err = h.App.TorrentClientRepository.SmartSelect(&torrent_client.SmartSelectParams{
			Torrent:          &b.Torrents[0],
			EpisodeNumbers:   b.SmartSelect.MissingEpisodeNumbers,
			Media:            completeAnime,
			Destination:      b.Destination,
			Platform:         h.App.AnilistPlatform,
			ShouldAddTorrent: true,
		})
		if err != nil {
			return h.RespondWithError(c, err)
		}

		// Record download intent
		hash := b.Torrents[0].InfoHash
		if hash == "" {
			hash = parseHashFromMagnet(b.Torrents[0].MagnetLink)
		}
		if err2 := h.App.Database.UpsertPendingDownloadIntent(hash, b.Media.ID, b.Destination); err2 != nil {
			h.App.Logger.Warn().Err(err2).Msg("torrent-client: Failed to record download intent")
		}
	} else {

		// Get magnets
		magnets := make([]string, 0)
		for _, t := range b.Torrents {
			// Get the torrent's provider extension
			providerExtension, ok := h.App.TorrentRepository.GetAnimeProviderExtension(t.Provider)
			if !ok {
				return h.RespondWithError(c, errors.New("provider extension not found for torrent"))
			}
			// Get the torrent magnet link
			magnet, err := providerExtension.GetProvider().GetTorrentMagnetLink(&t)
			if err != nil {
				return h.RespondWithError(c, err)
			}

			magnets = append(magnets, magnet)
		}

		// try to add torrents to client, on error return error
		err = h.App.TorrentClientRepository.AddMagnets(magnets, b.Destination)
		if err != nil {
			return h.RespondWithError(c, err)
		}

		// Record download intents
		for i, t := range b.Torrents {
			hash := t.InfoHash
			if hash == "" && i < len(magnets) {
				hash = parseHashFromMagnet(magnets[i])
			}
			if err2 := h.App.Database.UpsertPendingDownloadIntent(hash, b.Media.ID, b.Destination); err2 != nil {
				h.App.Logger.Warn().Err(err2).Msg("torrent-client: Failed to record download intent")
			}
		}
	}

	// Note: Files will be downloaded to library folder and picked up by scanner
	// Users can manually refresh their AniList collections if needed

	return h.RespondWithData(c, true)

}

// HandleTorrentClientAddMagnetFromRule
//
//	@summary adds magnets to the torrent client based on the AutoDownloader item.
//	@desc This is used to download torrents that were queued by the AutoDownloader.
//	@desc The item will be removed from the queue if the magnet was added successfully.
//	@desc The AutoDownloader items should be re-fetched after this.
//	@route /api/v1/torrent-client/rule-magnet [POST]
//	@returns bool
func (h *Handler) HandleTorrentClientAddMagnetFromRule(c echo.Context) error {

	type body struct {
		MagnetUrl    string `json:"magnetUrl"`
		RuleId       uint   `json:"ruleId"`
		QueuedItemId uint   `json:"queuedItemId"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	if b.MagnetUrl == "" || b.RuleId == 0 {
		return h.RespondWithError(c, errors.New("missing parameters"))
	}

	// Get rule from database
	rule, err := db_bridge.GetAutoDownloaderRule(h.App.Database, b.RuleId)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// try to start torrent client if it's not running
	ok := h.App.TorrentClientRepository.Start()
	if !ok {
		return h.RespondWithError(c, errors.New("could not start torrent client, verify your settings"))
	}

	// try to add torrents to client, on error return error
	err = h.App.TorrentClientRepository.AddMagnets([]string{b.MagnetUrl}, rule.Destination)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if b.QueuedItemId > 0 {
		// the magnet was added successfully, remove the item from the queue
		err = h.App.Database.DeleteAutoDownloaderItem(b.QueuedItemId)
	}

	return h.RespondWithData(c, true)

}

// parseHashFromMagnet extracts the info hash from a magnet URI (xt=urn:btih:<hash>).
// Handles both hex (40 chars) and base32 (32 chars) encoded hashes, always returns lowercase hex.
func parseHashFromMagnet(magnet string) string {
	u, err := url.Parse(magnet)
	if err != nil {
		return ""
	}
	xt := u.Query().Get("xt")
	if !strings.HasPrefix(xt, "urn:btih:") {
		return ""
	}
	hash := strings.TrimPrefix(xt, "urn:btih:")
	// If 40 chars, it's already hex
	if len(hash) == 40 {
		return strings.ToLower(hash)
	}
	// If 32 chars, it's base32 — decode to hex
	if len(hash) == 32 {
		decoded, err := base32.StdEncoding.DecodeString(strings.ToUpper(hash))
		if err == nil {
			return strings.ToLower(fmt.Sprintf("%x", decoded))
		}
	}
	return strings.ToLower(hash)
}

// ActiveDownloadItem represents an in-progress download mapped to an episode.
type ActiveDownloadItem struct {
	MediaId  int     `json:"mediaId"`
	Episode  int     `json:"episode"`  // 0 = whole batch / unknown
	Progress float64 `json:"progress"`
}

// HandleGetActiveDownloads
//
//	@summary returns currently downloading episodes with progress.
//	@desc This handler is used by the client to show download spinners on episodes.
//
//	@route /api/v1/torrent-client/active-downloads [GET]
//	@returns []handlers.ActiveDownloadItem
func (h *Handler) HandleGetActiveDownloads(c echo.Context) error {
	items := make([]ActiveDownloadItem, 0)

	intents, err := h.App.Database.GetIncompletePendingDownloadIntents()
	if err != nil || len(intents) == 0 {
		return h.RespondWithData(c, items)
	}

	torrents, err := h.App.TorrentClientRepository.GetList()
	if err != nil {
		return h.RespondWithData(c, items)
	}

	byHash := make(map[string]*torrent_client.Torrent, len(torrents))
	for _, t := range torrents {
		byHash[strings.ToLower(t.Hash)] = t
	}

	for _, intent := range intents {
		t, ok := byHash[strings.ToLower(intent.Hash)]
		if !ok || t.Status == torrent_client.TorrentStatusSeeding || t.Status == torrent_client.TorrentStatusStopped {
			continue
		}
		ep := 0
		if meta := habari.Parse(t.Name); meta != nil && len(meta.EpisodeNumber) == 1 {
			if n, e := strconv.Atoi(meta.EpisodeNumber[0]); e == nil {
				ep = n
			}
		}
		items = append(items, ActiveDownloadItem{MediaId: intent.MediaID, Episode: ep, Progress: t.Progress})
	}

	return h.RespondWithData(c, items)
}
