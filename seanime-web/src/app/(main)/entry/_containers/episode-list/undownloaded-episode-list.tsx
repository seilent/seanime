import { useDownloadProgress } from "@/app/(main)/entry/_lib/use-download-progress"
import { AL_BaseAnime, Anime_EntryDownloadInfo } from "@/api/generated/types"
import { EpisodeGridItem } from "@/app/(main)/_features/anime/_components/episode-grid-item"
import { PluginEpisodeGridItemMenuItems } from "@/app/(main)/_features/plugin/actions/plugin-actions"
import { useHasTorrentProvider } from "@/app/(main)/_hooks/use-server-status"
import { EpisodeListGrid } from "@/app/(main)/entry/_components/episode-list-grid"
import {
    __torrentSearch_selectionAtom,
    __torrentSearch_selectionEpisodeAtom,
} from "@/app/(main)/entry/_containers/torrent-search/torrent-search-drawer"
import { useSetAtom } from "jotai"
import React, { startTransition } from "react"
import { BiCalendarAlt, BiDownload } from "react-icons/bi"
import { EpisodeItemInfoModalButton } from "./episode-item"

export function UndownloadedEpisodeList({ downloadInfo, media }: {
    downloadInfo: Anime_EntryDownloadInfo | undefined,
    media: AL_BaseAnime
}) {

    const episodes = downloadInfo?.episodesToDownload

    const activeDownloads = useDownloadProgress()
    const myDownloads = activeDownloads.filter(d => d.mediaId === media.id)
    const dlEpisodes = new Set(myDownloads.filter(d => d.episode > 0).map(d => d.episode))
    const batchDownloading = myDownloads.some(d => d.episode === 0)
    const progressMap = new Map(myDownloads.filter(d => d.episode > 0).map(d => [d.episode, d.progress]))
    const batchProgress = myDownloads.find(d => d.episode === 0)?.progress ?? 0

    const setTorrentSearchIsOpen = useSetAtom(__torrentSearch_selectionAtom)
    const setTorrentSearchEpisode = useSetAtom(__torrentSearch_selectionEpisodeAtom)

    const { hasTorrentProvider } = useHasTorrentProvider()

    const text = hasTorrentProvider ? (downloadInfo?.rewatch
            ? "You have not downloaded the following:"
            : "You have not watched nor downloaded the following:") :
        "The following episodes are not in your library:"

    if (!episodes?.length) return null

    return (
        <div className="space-y-4" data-undownloaded-episode-list>
            <p className={""}>
                {text}
            </p>
            <EpisodeListGrid>
                {episodes?.sort((a, b) => a.episodeNumber - b.episodeNumber).slice(0, 28).map((ep, idx) => {
                    if (!ep.episode) return null
                    const episode = ep.episode
                    return (
                        <EpisodeGridItem
                            key={ep.episode.localFile?.path || idx}
                            media={media}
                            image={episode.episodeMetadata?.image}
                            isInvalid={episode.isInvalid}
                            title={episode.displayTitle}
                            episodeTitle={episode.episodeTitle}
                            episodeNumber={episode.episodeNumber}
                            progressNumber={episode.progressNumber}
                            description={episode.episodeMetadata?.summary || episode.episodeMetadata?.overview}
                            action={<>
                                {hasTorrentProvider && <div
                                    data-undownloaded-episode-list-action-download-button
                                    onClick={() => {
                                        setTorrentSearchEpisode(episode.episodeNumber)
                                        startTransition(() => {
                                            setTorrentSearchIsOpen("download")
                                        })
                                    }}
                                    className="inline-block text-orange-200 text-2xl animate-pulse cursor-pointer py-2"
                                >
                                    {(dlEpisodes.has(episode.episodeNumber) || batchDownloading) ? <CircularProgress progress={progressMap.get(episode.episodeNumber) ?? batchProgress} /> : <BiDownload />}
                                </div>}

                                <EpisodeItemInfoModalButton episode={episode} />

                                <PluginEpisodeGridItemMenuItems isDropdownMenu={true} type="undownloaded" episode={episode} />
                            </>}
                        >
                            <div data-undownloaded-episode-list-episode-metadata-container className="mt-1">
                                <p data-undownloaded-episode-list-episode-metadata-text className="flex gap-1 items-center text-sm text-[--muted]">
                                    <BiCalendarAlt /> {episode.episodeMetadata?.airDate
                                    ? `Aired on ${new Date(episode.episodeMetadata?.airDate).toLocaleDateString()}`
                                    : "Aired"}
                                </p>
                            </div>
                        </EpisodeGridItem>
                    )
                })}
            </EpisodeListGrid>
            {episodes.length > 28 && <h3>And more...</h3>}
        </div>
    )

}

function CircularProgress({ progress }: { progress: number }) {
    const p = Math.min(1, Math.max(0, progress || 0))
    const pct = Math.round(p * 100)
    const r = 10, c = 2 * Math.PI * r, offset = c * (1 - p)
    return (
        <svg className="size-6" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r={r} fill="none" stroke="currentColor" opacity={0.25} strokeWidth="3" />
            <circle cx="12" cy="12" r={r} fill="none" stroke="currentColor" strokeWidth="3"
                strokeDasharray={c} strokeDashoffset={offset} strokeLinecap="round"
                transform="rotate(-90 12 12)" />
            <text x="12" y="12" textAnchor="middle" dominantBaseline="central" fill="currentColor" fontSize="7">{pct}</text>
        </svg>
    )
}
