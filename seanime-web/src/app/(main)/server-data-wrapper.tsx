import { useGetStatus } from "@/api/hooks/status.hooks"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { GettingStartedPage } from "@/app/(main)/_features/getting-started/getting-started-page"
import { useServerStatus, useSetServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { LoadingOverlayWithLogo } from "@/components/shared/loading-overlay-with-logo"
import { LuffyError } from "@/components/shared/luffy-error"
import { logger } from "@/lib/helpers/debug"
import { WSEvents } from "@/lib/server/ws-events"
import { usePathname } from "next/navigation"
import React from "react"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { useQueryClient } from "@tanstack/react-query"

type ServerDataWrapperProps = {
    host: string
    children?: React.ReactNode
}

export function ServerDataWrapper(props: ServerDataWrapperProps) {

    const {
        host,
        children,
        ...rest
    } = props

    const pathname = usePathname()
    const serverStatus = useServerStatus()
    const setServerStatus = useSetServerStatus()
    const { data: _serverStatus, isLoading, refetch } = useGetStatus()
    const queryClient = useQueryClient()

    React.useEffect(() => {
        if (_serverStatus) {
            // logger("SERVER").info("Server status", _serverStatus)
            setServerStatus(_serverStatus)
        }
    }, [_serverStatus])

    // Refetch status on server-ready

    // Enable global SSE for real-time updates across all pages
    useSSEEvents({
        enabled: true,
        onEvent: (event: any) => {
            try {
                const type = event?.type
                logger("SSE Global").info("Received SSE event:", type)

                switch (type) {
                    case WSEvents.ANILIST_DATA_LOADED: {
                        logger("Data Wrapper").info("Anilist data loaded (SSE), refetching server status")
                        refetch()
                        break
                    }
                    case "file_mapped":
                    case "file_unmapped":
                    case "file_ignored":
                    case "file_unignored": {
                        queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key] })
                        queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.key] })
                        queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key] })
                        break
                    }
                    case "progress_sync_updated": {
                        queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.key] })
                        break
                    }
                    case "system-scan-completed": {
                        const scanPayload = event.payload
                        const currentTimestamp = scanPayload?.timestamp
                        const mappedAnime = scanPayload?.mappedAnime || []
                        const lastScanTimestamp = localStorage.getItem('last-scan-timestamp')
                        const isNewScan = !lastScanTimestamp || parseInt(lastScanTimestamp as string) !== currentTimestamp
                        if (isNewScan && currentTimestamp) {
                            localStorage.setItem('last-scan-timestamp', String(currentTimestamp))
                            const scanDate = new Date(currentTimestamp * 1000).toLocaleString()
                            console.log(`[SSE] New mappings received at ${scanDate}`)
                            if (mappedAnime.length > 0) {
                                console.log(`[SSE] ${scanPayload?.newMappings} new files mapped for ${mappedAnime.length} anime:`)
                                mappedAnime.forEach((anime: any) => {
                                    const episodeList = anime.episodes.sort((a: number, b: number) => a - b).join(', ')
                                    console.log(`  - ${anime.title}: ${anime.episodeCount} episode(s) (${episodeList})`)
                                })
                            }
                            mappedAnime.forEach((anime: any) => {
                                queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key, String(anime.anilistId)] })
                            })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key] })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.LOCALFILES.GetLocalFiles.key] })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key] })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key] })
                            console.log(`[SSE] Cache invalidated for ${mappedAnime.length} specific anime entries and general queries`)
                        } else {
                            console.log("[SSE] Duplicate scan event ignored (same timestamp)")
                        }
                        break
                    }
                    case "sync-check": {
                        const payload = event.payload
                        if (payload?.cache_stale === true) {
                            console.log("SSE: Cache is stale, refreshing queries")
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key] })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.LOCALFILES.GetLocalFiles.key] })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key] })
                            queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key] })
                        } else {
                            console.log("SSE: Cache is up to date")
                        }
                        break
                    }
                    case "connected": {
                        console.log("SSE: Connection established", event.payload)
                        break
                    }
                }
            } catch (e) {
                // swallow
            }
        }
    })


    // Refetch the server status every 2 seconds if serverReady is false
    // This is a fallback to the websocket
    const intervalId = React.useRef<NodeJS.Timeout | null>(null)
    React.useEffect(() => {
        if (!serverStatus?.serverReady) {
            intervalId.current = setInterval(() => {
                logger("Data Wrapper").info("Refetching server status")
                refetch()
            }, 2000)
        }
        return () => {
            logger("Data Wrapper").info("Clearing interval")
            if (intervalId.current) {
                clearInterval(intervalId.current)
                intervalId.current = null
            }
        }
    }, [serverStatus?.serverReady])

    /**
     * If the server status is loading or doesn't exist, show the loading overlay
     */
    if (isLoading || !serverStatus) return <LoadingOverlayWithLogo />
    if (!serverStatus?.serverReady) return <LoadingOverlayWithLogo title="L o a d i n g" />

    /**
     * If the pathname is /auth/callback, show the callback page
     */
    if (pathname.startsWith("/auth/callback")) return children

    /**
     * Show getting started page if setup is not completed
     */
    if (!serverStatus?.setupCompleted) {
        return <GettingStartedPage status={serverStatus} />
    }

    /**
     * If the app is updating, show a different screen
     */
    if (serverStatus?.updating) {
        return <div className="container max-w-3xl py-10">
            <div className="mb-4 flex justify-center w-full">
                <img src="/logo_2.png" alt="logo" className="w-36 h-auto" />
            </div>
            <p className="text-center text-lg">
                Seanime is currently updating. Refresh the page once the update is complete and the connection has been reestablished.
            </p>
        </div>
    }

    

    return children
}
