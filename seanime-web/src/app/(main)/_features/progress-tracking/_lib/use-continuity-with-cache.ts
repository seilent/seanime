import { useAtom } from "jotai"
import { MediaPlayerInstance } from "@vidstack/react"
import { useEffect, useRef, useCallback } from "react"
import { useHandleContinuityWithMediaPlayer } from "@/api/hooks/continuity.hooks"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { useAuth } from "@/contexts/auth-context"
import { logger } from "@/lib/helpers/debug"
import { playbackProgressCacheService } from "./playback-progress-cache.service"
import { useUpdateContinuityWatchHistoryItem } from "@/api/hooks/continuity.hooks"
import { Nullish } from "@/api/generated/types"

interface UseContinuityWithCacheProps {
    playerRef: React.RefObject<MediaPlayerInstance | HTMLVideoElement>
    episodeNumber: Nullish<number>
    mediaId: Nullish<number | string>
}

export function useContinuityWithCache({ playerRef, episodeNumber, mediaId }: UseContinuityWithCacheProps) {
    const serverStatus = useServerStatus()
    const { user } = useAuth()
    const { handleUpdateWatchHistory } = useHandleContinuityWithMediaPlayer(playerRef, episodeNumber, mediaId)
    const { mutate: updateWatchHistory } = useUpdateContinuityWatchHistoryItem()

    const isCacheEnabled = serverStatus?.settings?.library?.enableWatchContinuity
    const syncIntervalRef = useRef<NodeJS.Timeout | null>(null)
    const cacheUpdateIntervalRef = useRef<NodeJS.Timeout | null>(null)
    const lastUpdateTimeRef = useRef<number>(0)

    // Clean up intervals on unmount
    useEffect(() => {
        return () => {
            if (syncIntervalRef.current) {
                clearInterval(syncIntervalRef.current)
            }
            if (cacheUpdateIntervalRef.current) {
                clearInterval(cacheUpdateIntervalRef.current)
            }
            // Perform final sync before unmount
            if (isCacheEnabled && mediaId && episodeNumber) {
                performSync()
            }
        }
    }, [isCacheEnabled, mediaId, episodeNumber])

    // Start cache update interval (1 second)
    useEffect(() => {
        if (!isCacheEnabled || !playerRef.current || !mediaId || !episodeNumber) {
            if (cacheUpdateIntervalRef.current) {
                clearInterval(cacheUpdateIntervalRef.current)
                cacheUpdateIntervalRef.current = null
            }
            return
        }

        cacheUpdateIntervalRef.current = setInterval(() => {
            updateLocalCache()
        }, 1000)

        return () => {
            if (cacheUpdateIntervalRef.current) {
                clearInterval(cacheUpdateIntervalRef.current)
                cacheUpdateIntervalRef.current = null
            }
        }
    }, [isCacheEnabled, playerRef.current, mediaId, episodeNumber])

    // Start sync interval (30 seconds)
    useEffect(() => {
        if (!isCacheEnabled || !mediaId || !episodeNumber) {
            if (syncIntervalRef.current) {
                clearInterval(syncIntervalRef.current)
                syncIntervalRef.current = null
            }
            return
        }

        syncIntervalRef.current = setInterval(() => {
            performSync()
        }, 30000)

        return () => {
            if (syncIntervalRef.current) {
                clearInterval(syncIntervalRef.current)
                syncIntervalRef.current = null
            }
        }
    }, [isCacheEnabled, mediaId, episodeNumber])

    const updateLocalCache = useCallback(() => {
        if (!playerRef.current || !mediaId || !episodeNumber) return

        const currentTime = playerRef.current.currentTime
        const duration = playerRef.current.duration

        if (!duration || currentTime < 0) return

        // Only update if there's a meaningful change (more than 0.5 seconds)
        const now = Date.now()
        if (now - lastUpdateTimeRef.current < 500) return
        lastUpdateTimeRef.current = now

        try {
            playbackProgressCacheService.saveProgress(
                Number(mediaId),
                episodeNumber,
                currentTime,
                duration,
                user?.id || null
            )
        } catch (error) {
            logger("CACHE").error("Error updating local cache:", error)
        }
    }, [playerRef.current, mediaId, episodeNumber, user?.id])

    const performSync = useCallback(() => {
        if (!mediaId || !episodeNumber) return

        const pendingItems = playbackProgressCacheService.getPendingSyncItems()
        if (pendingItems.length === 0) return

        // Get the most recent progress for the current episode
        const currentItem = pendingItems
            .filter(item => item.mediaId === Number(mediaId) && item.episodeNumber === episodeNumber)
            .sort((a, b) => b.timestamp - a.timestamp)[0]

        if (!currentItem) return

        try {
            // Sync the most recent cached progress to server
            updateWatchHistory({
                options: {
                    currentTime: currentItem.currentTime,
                    duration: currentItem.duration,
                    mediaId: Number(mediaId),
                    episodeNumber: episodeNumber,
                    kind: "onlinestream",
                },
            })

            // Clear the sync queue for this item
            playbackProgressCacheService.clearSyncQueue()

            logger("CACHE").info("Synced cached progress to server", {
                mediaId,
                episodeNumber,
                currentTime: currentItem.currentTime,
                duration: currentItem.duration,
            })
        } catch (error) {
            logger("CACHE").error("Error syncing cached progress to server:", error)
        }
    }, [mediaId, episodeNumber, updateWatchHistory])

    const manualSync = useCallback(() => {
        if (!isCacheEnabled) return
        performSync()
    }, [isCacheEnabled, performSync])

    // Enhanced watch history update that also updates cache
    const handleUpdateWatchHistoryWithCache = useCallback(() => {
        // Update local cache immediately
        updateLocalCache()

        // Call the original update function for immediate server sync
        return handleUpdateWatchHistory()
    }, [updateLocalCache, handleUpdateWatchHistory])

    return {
        handleUpdateWatchHistory: handleUpdateWatchHistoryWithCache,
        manualSync,
        isCacheEnabled,
    }
}