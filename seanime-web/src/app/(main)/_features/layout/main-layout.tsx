"use client"
import { PlaylistsModal } from "@/app/(main)/(library)/_containers/playlists/playlists-modal"
import { ScanProgressBar } from "@/app/(main)/(library)/_containers/scan-progress-bar"
import { ErrorExplainer } from "@/app/(main)/_features/error-explainer/error-explainer"
import { GlobalSearch } from "@/app/(main)/_features/global-search/global-search"
import { IssueReport } from "@/app/(main)/_features/issue-report/issue-report"
import { LibraryWatcher } from "@/app/(main)/_features/library-watcher/library-watcher"
import { MediaPreviewModal } from "@/app/(main)/_features/media/_containers/media-preview-modal"
import { MainSidebar } from "@/app/(main)/_features/navigation/main-sidebar"
import { PluginManager } from "@/app/(main)/_features/plugin/plugin-manager"
import { ManualProgressTracking } from "@/app/(main)/_features/progress-tracking/manual-progress-tracking"
import { PlaybackManagerProgressTracking } from "@/app/(main)/_features/progress-tracking/playback-manager-progress-tracking"
import { SeaCommand } from "@/app/(main)/_features/sea-command/sea-command"
import { VideoCoreProvider } from "@/app/(main)/_features/video-core/video-core"
import { theaterModeAtom } from "@/app/(main)/_features/sea-media-player/sea-media-player-layout"
import { useAnimeCollectionLoader } from "@/app/(main)/_hooks/anilist-collection-loader"
import { useAnimeLibraryCollectionLoader } from "@/app/(main)/_hooks/anime-library-collection-loader"
import { useMissingEpisodesLoader } from "@/app/(main)/_hooks/missing-episodes-loader"
import { useAnimeCollectionListener } from "@/app/(main)/_listeners/anilist-collection.listeners"
import { useAutoDownloaderItemListener } from "@/app/(main)/_listeners/autodownloader.listeners"
import { useExtensionListener } from "@/app/(main)/_listeners/extensions.listeners"
import { useExternalPlayerLinkListener } from "@/app/(main)/_listeners/external-player-link.listeners"
import { useMangaListener } from "@/app/(main)/_listeners/manga.listeners"
import { useMiscEventListeners } from "@/app/(main)/_listeners/misc-events.listeners"
import { useSyncListener } from "@/app/(main)/_listeners/sync.listeners"
import { ChapterDownloadsDrawer } from "@/app/(main)/manga/_containers/chapter-downloads/chapter-downloads-drawer"
import { LoadingOverlayWithLogo } from "@/components/shared/loading-overlay-with-logo"
import { AppLayout, AppLayoutContent, AppLayoutSidebar, AppSidebarProvider } from "@/components/ui/app-layout"
import { __isElectronDesktop__ } from "@/types/constants"
import { useAtom } from "jotai/react"
import { usePathname, useRouter } from "next/navigation"
import React from "react"
import { useServerStatus } from "../../_hooks/use-server-status"
import { useInvalidateQueriesListener } from "../../_listeners/invalidate-queries.listeners"
import { Announcements } from "../announcements"
import { NativePlayer } from "../native-player/native-player"
import { TopIndefiniteLoader } from "../top-indefinite-loader"

export const MainLayout = ({ children }: { children: React.ReactNode }) => {

    /**
     * Data loaders
     */
    useAnimeLibraryCollectionLoader()
    useAnimeCollectionLoader()
    useMissingEpisodesLoader()

    /**
     * Websocket listeners
     */
    useAutoDownloaderItemListener()
    useAnimeCollectionListener()
    useMiscEventListeners()
    useExtensionListener()
    useMangaListener()
    useExternalPlayerLinkListener()
    useSyncListener()
    useInvalidateQueriesListener()

    const serverStatus = useServerStatus()
    const router = useRouter()
    const pathname = usePathname()
    const [theaterMode, setTheaterMode] = useAtom(theaterModeAtom)
    const isMediaStreamPage = pathname.includes("/mediastream")


    return (
        <>
            <GlobalSearch />
            <ScanProgressBar />
            <LibraryWatcher />
            <PlaylistsModal />
            <ChapterDownloadsDrawer />
            <MediaPreviewModal />
            <PlaybackManagerProgressTracking />
            <ManualProgressTracking />
            <IssueReport />
            <ErrorExplainer />
            <SeaCommand />
            <PluginManager />
            {__isElectronDesktop__ && <VideoCoreProvider>
                <NativePlayer />
            </VideoCoreProvider>}
            <TopIndefiniteLoader />
            <Announcements />

            {isMediaStreamPage && theaterMode && (
                <div
                    className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[998] cursor-pointer"
                    style={{ display: theaterMode ? 'block' : 'none' }}
                    onClick={() => setTheaterMode(false)}
                />
            )}

            <AppSidebarProvider>
                <AppLayout withSidebar sidebarSize="slim">
                    <AppLayoutSidebar>
                        <MainSidebar />
                    </AppLayoutSidebar>
                    <AppLayout>
                        <AppLayoutContent>
                            {children}
                        </AppLayoutContent>
                    </AppLayout>
                </AppLayout>
            </AppSidebarProvider>
        </>
    )
}
