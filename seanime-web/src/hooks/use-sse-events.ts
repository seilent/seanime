import { useEffect, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { API_ENDPOINTS } from "@/api/generated/endpoints"

interface SSEEvent {
    type: string
    payload: any
}

interface UseSSEEventsOptions {
    enabled?: boolean
    onEvent?: (event: SSEEvent) => void
}

export function useSSEEvents(options: UseSSEEventsOptions = {}) {
    const { enabled = true, onEvent } = options
    const queryClient = useQueryClient()

    const handleEvent = useCallback((event: SSEEvent) => {
        // Call custom event handler if provided
        if (onEvent) {
            onEvent(event)
        }

        // Handle global mapping events for auto-refresh
        switch (event.type) {
            case "file_mapped":
            case "file_unmapped":
            case "file_ignored":
            case "file_unignored":
                // Invalidate relevant queries
                queryClient.invalidateQueries({
                    queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key]
                })
                queryClient.invalidateQueries({
                    queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.key]
                })
                queryClient.invalidateQueries({
                    queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key]
                })
                break

            case "progress_sync_updated":
                queryClient.invalidateQueries({
                    queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.key]
                })
                break

            case "system-scan-completed":
                // Extract event payload
                const scanPayload = event.payload
                const currentTimestamp = scanPayload?.timestamp
                const mappedAnime = scanPayload?.mappedAnime || []

                // Check if this is a new scan result by comparing timestamps
                const lastScanTimestamp = localStorage.getItem('last-scan-timestamp')
                const isNewScan = !lastScanTimestamp || parseInt(lastScanTimestamp) !== currentTimestamp

                if (isNewScan && currentTimestamp) {
                    // Store new timestamp
                    localStorage.setItem('last-scan-timestamp', currentTimestamp.toString())

                    // Console logging for new mappings
                    const scanDate = new Date(currentTimestamp * 1000).toLocaleString()
                    console.log(`[SSE] New mappings received at ${scanDate}`)

                    if (mappedAnime.length > 0) {
                        console.log(`[SSE] ${scanPayload?.newMappings} new files mapped for ${mappedAnime.length} anime:`)
                        mappedAnime.forEach((anime: any) => {
                            const episodeList = anime.episodes.sort((a: number, b: number) => a - b).join(', ')
                            console.log(`  - ${anime.title}: ${anime.episodeCount} episode(s) (${episodeList})`)
                        })
                    }

                    // Invalidate specific anime entry queries for mapped anime
                    mappedAnime.forEach((anime: any) => {
                        queryClient.invalidateQueries({
                            queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key, String(anime.anilistId)]
                        })
                    })

                    // Invalidate general queries that show lists of anime/files
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key]
                    })
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.LOCALFILES.GetLocalFiles.key]
                    })
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key]
                    })
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key]
                    })

                    console.log(`[SSE] Cache invalidated for ${mappedAnime.length} specific anime entries and general queries`)
                } else {
                    console.log("[SSE] Duplicate scan event ignored (same timestamp)")
                }
                break

            case "sync-check":
                // Handle database-cache synchronization check
                const payload = event.payload
                if (payload?.cache_stale === true) {
                    console.log("SSE: Cache is stale, refreshing queries")
                    // Invalidate all mapping-related queries to ensure fresh data
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key]
                    })
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.LOCALFILES.GetLocalFiles.key]
                    })
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key]
                    })
                    queryClient.invalidateQueries({
                        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key]
                    })
                } else {
                    console.log("SSE: Cache is up to date")
                }
                break

            case "connected":
                console.log("SSE: Connection established", event.payload)
                break

            default:
                // Handle other events as needed
                break
        }
    }, [onEvent, queryClient])

    useEffect(() => {
        if (!enabled) return

        // Create SSE connection
        const eventSource = new EventSource("/api/v1/sse/events")

        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data) as SSEEvent
                handleEvent(data)
            } catch (error) {
                console.error("Failed to parse SSE event:", error)
            }
        }

        eventSource.onerror = (error) => {
            console.error("SSE connection error:", error)
        }

        // Cleanup on unmount
        return () => {
            eventSource.close()
        }
    }, [enabled, handleEvent])
}

export default useSSEEvents