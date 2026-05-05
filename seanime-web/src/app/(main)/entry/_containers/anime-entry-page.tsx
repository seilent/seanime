import { useGetAnilistAnimeDetails } from "@/api/hooks/anilist.hooks"
import { useGetAnimeEntry, useValidateAnimeEntryLocalFiles } from "@/api/hooks/anime_entries.hooks"
import { MediaEntryCharactersSection } from "@/app/(main)/_features/media/_components/media-entry-characters-section"
import { MediaEntryPageLoadingDisplay } from "@/app/(main)/_features/media/_components/media-entry-page-loading-display"
import { useSeaCommandInject } from "@/app/(main)/_features/sea-command/use-inject"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { MetaSection } from "@/app/(main)/entry/_components/meta-section"
import { RelationsRecommendationsSection } from "@/app/(main)/entry/_components/relations-recommendations-section"
import { EpisodeSection } from "@/app/(main)/entry/_containers/episode-list/episode-section"
import { __torrentSearch_selectionAtom, TorrentSearchDrawer } from "@/app/(main)/entry/_containers/torrent-search/torrent-search-drawer"
import { PageWrapper } from "@/components/shared/page-wrapper"
import { ThemeMediaPageInfoBoxSize, useThemeSettings } from "@/lib/theme/hooks"
import { atom } from "jotai"
import { useAtom, useSetAtom } from "jotai/react"
import { AnimatePresence } from "motion/react"
import { useRouter, useSearchParams } from "next/navigation"
import React from "react"
import { useUnmount } from "react-use"

export const __anime_entryPageViewAtom = atom<"library">("library")

export function useAnimeEntryPageView() {
    const [currentView, setView] = useAtom(__anime_entryPageViewAtom)

    const isLibraryView = currentView === "library"

    return {
        currentView,
        setView,
        isLibraryView,
    }
}

export function AnimeEntryPage() {

    const serverStatus = useServerStatus()
    const router = useRouter()
    const searchParams = useSearchParams()
    const mediaId = searchParams.get("id")
    const { data: animeEntry, isLoading: animeEntryLoading } = useGetAnimeEntry(mediaId)
    const { data: animeDetails, isLoading: animeDetailsLoading } = useGetAnilistAnimeDetails(mediaId)
    const ts = useThemeSettings()

    const { currentView, isLibraryView, setView } = useAnimeEntryPageView()

    // Validate local files when entry is loaded
    const validatedEntryRef = React.useRef<Set<string>>(new Set())
    const { mutate: validateLocalFiles } = useValidateAnimeEntryLocalFiles(mediaId ? parseInt(mediaId) : undefined)

    React.useEffect(() => {
        // Only validate once per entry per session
        if (animeEntry && mediaId && !validatedEntryRef.current.has(mediaId)) {
            validatedEntryRef.current.add(mediaId)
            validateLocalFiles({ mediaId: parseInt(mediaId) })
        }
    }, [animeEntry, mediaId, validateLocalFiles])

    React.useEffect(() => {
        try {
            if (animeEntry?.media?.title?.userPreferred) {
                document.title = `${animeEntry?.media?.title?.userPreferred} | Seanime`
            }
        }
        catch {
        }
    }, [animeEntry])

    // useWebsocketSendEffect({
    //     type: WebviewEvents.ANIME_ENTRY_PAGE_VIEWED,
    //     payload: {
    //         animeEntry,
    //     },
    // }, animeEntry)

    const switchedView = React.useRef(false)
    React.useLayoutEffect(() => {
        if (!animeEntryLoading &&
            animeEntry?.media?.status !== "NOT_YET_RELEASED" && // Anime is not yet released
            searchParams.get("tab") && searchParams.get("tab") !== "library" && // Tab is not library
            !switchedView.current // View has not been switched yet
        ) {
            switchedView.current = true
        }

    }, [animeEntryLoading, searchParams, currentView])

    React.useEffect(() => {
        if (!mediaId || (!animeEntryLoading && !animeEntry)) {
            router.push("/")
        }
    }, [animeEntry, animeEntryLoading])

    // Reset view when unmounting
    useUnmount(() => {
        setView("library")
    })

    const setTorrentSearchDrawer = useSetAtom(__torrentSearch_selectionAtom)

    const { inject, remove } = useSeaCommandInject()
    React.useEffect(() => {
        inject("anime-entry-navigation", {
            items: [
                ...[{
                    id: "library",
                    description: "Downloaded episodes",
                    show: currentView !== "library",
                },
                ].map(item => ({
                    id: item.id,
                    value: item.id,
                    heading: "Views",
                    data: item,
                    render: () => <div>{item.description}</div>,
                    onSelect: () => setView(item.id as any),
                    shouldShow: () => !!item.show,
                })),
                {
                    id: "download",
                    value: "download",
                    render: () => <div>Download torrents</div>,
                    heading: "Views",
                    data: "download torrents",
                    onSelect: () => setTorrentSearchDrawer("download"),
                    shouldShow: () => currentView === "library",
                },
            ],
            filter: ({ item, input }) => {
                if (!input) return true
                return item.data?.description?.toLowerCase().startsWith(input.toLowerCase())
            },
            priority: -1,
        })

        return () => remove("anime-entry-navigation")
    }, [currentView, serverStatus])

    if (animeEntryLoading || animeDetailsLoading) return <MediaEntryPageLoadingDisplay />
    if (!animeEntry) return null

    return (
        <div data-anime-entry-page data-media={JSON.stringify(animeEntry.media)} data-anime-entry-list-data={JSON.stringify(animeEntry.listData)}>
            <MetaSection entry={animeEntry} details={animeDetails} />

            <div className="px-4 md:px-8 relative z-[8]" data-anime-entry-page-content-container>
                <PageWrapper
                    data-anime-entry-page-content
                    className="relative 2xl:order-first pb-10 lg:min-h-[calc(100vh-10rem)]"
                    {...{
                        initial: { opacity: 0, y: 60 },
                        animate: { opacity: 1, y: 0 },
                        exit: { opacity: 0, y: 60 },
                        transition: {
                            type: "spring",
                            damping: 10,
                            stiffness: 80,
                            delay: 0.6,
                        },
                    }}
                >
                    {(ts.mediaPageBannerInfoBoxSize === ThemeMediaPageInfoBoxSize.Fluid) && (
                        <>
                            {/*{currentView !== "library" ? <div className="h-10 lg:h-0" /> : }*/}
                        </>
                    )}
                    <AnimatePresence mode="wait" initial={false}>

                        {(currentView === "library") && <PageWrapper
                            data-anime-entry-page-episode-list-view
                            key="episode-list"
                            className="relative 2xl:order-first pb-10"
                            {...{
                                initial: { opacity: 0, y: 60 },
                                animate: { opacity: 1, y: 0 },
                                exit: { opacity: 0, scale: 0.99 },
                                transition: {
                                    duration: 0.35,
                                },
                            }}
                        >
                            <div className="h-10" />
                            <EpisodeSection
                                entry={animeEntry}
                                details={animeDetails}
                                bottomSection={<>
                                    <MediaEntryCharactersSection details={animeDetails} />
                                    <RelationsRecommendationsSection entry={animeEntry} details={animeDetails} />
                                </>}
                            />
                        </PageWrapper>}


                    </AnimatePresence>
                </PageWrapper>
            </div>

            <TorrentSearchDrawer entry={animeEntry} />
        </div>
    )
}
