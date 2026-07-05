import { Anime_Entry } from "@/api/generated/types"
import { useTorrentClientDownload } from "@/api/hooks/torrent_client.hooks"
import { useGetSubspleaseStatus } from "@/api/hooks/subsplease.hooks"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { AnimeMetaActionButton } from "@/app/(main)/entry/_components/meta-section"
import { __torrentSearch_selectionAtom } from "@/app/(main)/entry/_containers/torrent-search/torrent-search-drawer"
import { useSetAtom } from "jotai/react"
import React, { useMemo, useRef } from "react"
import { BiCheck, BiDownload } from "react-icons/bi"
import { FiSearch } from "react-icons/fi"

export function TorrentSearchButton({ entry }: { entry: Anime_Entry }) {

    const setter = useSetAtom(__torrentSearch_selectionAtom)
    const serverStatus = useServerStatus()
    const libraryPath = serverStatus?.settings?.library?.libraryPath || ""
    const count = entry.downloadInfo?.episodesToDownload?.length
    const isMovie = useMemo(() => entry.media?.format === "MOVIE", [entry.media?.format])

    const syncStarted = useRef(false)
    const syncCount = useRef(0)

    const enabled = !!(entry.media?.id && entry.media?.status && entry.media?.format) && !syncStarted.current
    const { data: spStatus } = useGetSubspleaseStatus(entry.media, enabled)

    const { mutate: download, isPending } = useTorrentClientDownload()

    const handleDirectDownload = () => {
        if (!spStatus?.toSync?.length || !libraryPath) return
        syncStarted.current = true
        syncCount.current = spStatus.toSync.length
        download({
            torrents: spStatus.toSync,
            destination: libraryPath,
            smartSelect: { enabled: false, missingEpisodeNumbers: [] },
            media: entry.media,
            deleteExistingFiles: false,
        })
    }

    // Downloading state
    if (syncStarted.current) {
        return (
            <div className="contents" data-torrent-search-button-container>
                <AnimeMetaActionButton
                    intent="gray-subtle"
                    size="md"
                    leftIcon={<BiCheck />}
                    iconClass="text-2xl"
                    disabled
                    data-torrent-search-button
                >
                    Downloading {syncCount.current} episode{syncCount.current > 1 ? "s" : ""}
                </AnimeMetaActionButton>
            </div>
        )
    }

    // Fully synced — hide button
    if (spStatus?.available && (!spStatus.toSync || spStatus.toSync.length === 0)) {
        return null
    }

    // Episodes to sync — show sync button
    if (spStatus?.toSync && spStatus.toSync.length > 0) {
        return (
            <div className="contents" data-torrent-search-button-container>
                <AnimeMetaActionButton
                    intent="white"
                    size="md"
                    leftIcon={<BiDownload />}
                    iconClass="text-2xl"
                    onClick={handleDirectDownload}
                    disabled={isPending}
                    data-torrent-search-button
                >
                    {isPending ? "Syncing..." : `Sync ${spStatus.toSync.length} episode${spStatus.toSync.length > 1 ? "s" : ""}`}
                </AnimeMetaActionButton>
            </div>
        )
    }

    // Fallback — not on SubsPlease or still loading
    return (
        <div className="contents" data-torrent-search-button-container>
            <AnimeMetaActionButton
                intent={!entry.downloadInfo?.hasInaccurateSchedule ? (!!count ? "white" : "gray-subtle") : "white-subtle"}
                size="md"
                leftIcon={(!!count) ? <BiDownload /> : <FiSearch />}
                iconClass="text-2xl"
                onClick={() => setter("download")}
                data-torrent-search-button
            >
                {(!entry.downloadInfo?.hasInaccurateSchedule && !!count) ? <>
                    {(!isMovie) && `Download ${entry.downloadInfo?.batchAll ? "batch /" : "next"} ${count > 1 ? `${count} episodes` : "episode"}`}
                    {(isMovie) && `Download movie`}
                </> : <>
                    Search torrents
                </>}
            </AnimeMetaActionButton>
        </div>
    )
}
