import { Anime_Entry, Anime_Episode } from "@/api/generated/types"
import { __torrentSearch_selectedTorrentsAtom } from "@/app/(main)/entry/_containers/torrent-search/torrent-search-container"
import { __torrentSearch_selectionAtom, TorrentSelectionType } from "@/app/(main)/entry/_containers/torrent-search/torrent-search-drawer"
import { atom, useSetAtom } from "jotai/index"
import { useAtom } from "jotai/react"
import React from "react"

const __torrentSearch_streamingSelectedEpisodeAtom = atom<Anime_Episode | null>(null)

export function useTorrentSearchSelectedStreamEpisode() {
    const [value, setter] = useAtom(__torrentSearch_streamingSelectedEpisodeAtom)

    return {
        torrentStreamingSelectedEpisode: value,
        setTorrentStreamingSelectedEpisode: setter,
    }
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

export function useTorrentSearchSelection({ type = "download", entry }: { type: TorrentSelectionType | undefined, entry: Anime_Entry }) {

    const [selectedTorrents, setSelectedTorrents] = useAtom(__torrentSearch_selectedTorrentsAtom)
    const { torrentStreamingSelectedEpisode } = useTorrentSearchSelectedStreamEpisode()
    const [, setDrawerOpen] = useAtom(__torrentSearch_selectionAtom)

    const onTorrentValidated = () => {
        // Torrent search validation logic would go here
        console.log("onTorrentValidated", torrentStreamingSelectedEpisode)
    }

    return {
        onTorrentValidated,
    }
}
