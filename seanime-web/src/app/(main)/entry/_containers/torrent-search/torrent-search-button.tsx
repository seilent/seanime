import { Anime_Entry, HibikeTorrent_AnimeTorrent } from "@/api/generated/types"
import { useTorrentClientDownload } from "@/api/hooks/torrent_client.hooks"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { AnimeMetaActionButton } from "@/app/(main)/entry/_components/meta-section"
import { __torrentSearch_selectionAtom } from "@/app/(main)/entry/_containers/torrent-search/torrent-search-drawer"
import { useServerMutation } from "@/api/client/requests"
import { useSetAtom } from "jotai/react"
import React, { useEffect, useMemo, useState } from "react"
import { BiDownload } from "react-icons/bi"
import { FiSearch } from "react-icons/fi"

export function TorrentSearchButton({ entry }: { entry: Anime_Entry }) {

    const setter = useSetAtom(__torrentSearch_selectionAtom)
    const serverStatus = useServerStatus()
    const libraryPath = serverStatus?.settings?.library?.libraryPath || ""
    const count = entry.downloadInfo?.episodesToDownload?.length
    const isMovie = useMemo(() => entry.media?.format === "MOVIE", [entry.media?.format])

    // SubsPlease direct download state
    const [spEpisodes, setSpEpisodes] = useState<HibikeTorrent_AnimeTorrent[]>([])

    // Check SubsPlease for available episodes
    const { mutate: fetchSpEpisodes } = useServerMutation<HibikeTorrent_AnimeTorrent[], { media: any }>({
        endpoint: "/api/v1/subsplease/episodes",
        method: "POST",
        mutationKey: ["subsplease-episodes", String(entry.mediaId)],
        onSuccess: (data) => {
            setSpEpisodes(data || [])
        },
    })

    useEffect(() => {
        if (entry.media) {
            fetchSpEpisodes({ media: entry.media })
        }
    }, [entry.mediaId])

    // Download mutation
    const { mutate: download, isPending } = useTorrentClientDownload(() => {
        setSpEpisodes([])
    })

    const handleDirectDownload = () => {
        if (!spEpisodes.length || !libraryPath) return
        download({
            torrents: spEpisodes,
            destination: libraryPath,
            smartSelect: { enabled: false, missingEpisodeNumbers: [] },
            media: entry.media,
            deleteExistingFiles: true,
        })
    }

    // If SubsPlease has missing episodes, show direct download button
    if (spEpisodes.length > 0) {
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
                    {isPending ? "Syncing..." : `Sync ${spEpisodes.length} episode${spEpisodes.length > 1 ? "s" : ""}`}
                </AnimeMetaActionButton>
            </div>
        )
    }

    // Fallback to normal search button
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
