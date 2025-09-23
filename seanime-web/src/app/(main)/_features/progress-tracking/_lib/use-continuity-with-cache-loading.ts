import { useHandleCurrentMediaContinuity } from "@/api/hooks/continuity.hooks"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { useAuth } from "@/contexts/auth-context"
import { useCallback, useEffect } from "react"
import { logger } from "@/lib/helpers/debug"
import { playbackProgressCacheService } from "./playback-progress-cache.service"
import { Nullish } from "@/api/generated/types"

interface UseContinuityWithCacheLoadingProps {
    mediaId: Nullish<number | string>
    episodeNumber: Nullish<number>
}

export function useContinuityWithCacheLoading({ mediaId, episodeNumber }: UseContinuityWithCacheLoadingProps) {
    const serverStatus = useServerStatus()
    const { user } = useAuth()
    const { watchHistory, waitForWatchHistory, getEpisodeContinuitySeekTo } = useHandleCurrentMediaContinuity(mediaId)

    const isCacheEnabled = serverStatus?.settings?.library?.enableWatchContinuity

    const getCachedSeekTime = useCallback(() => {
        if (!isCacheEnabled || !mediaId || !episodeNumber) return 0

        try {
            const cachedProgress = playbackProgressCacheService.getProgress(
                Number(mediaId),
                episodeNumber
            )

            if (cachedProgress) {
                logger("CACHE").info("Using cached seek time", {
                    mediaId,
                    episodeNumber,
                    currentTime: cachedProgress.currentTime,
                    duration: cachedProgress.duration,
                })
                return cachedProgress.currentTime
            }
        } catch (error) {
            logger("CACHE").error("Error getting cached seek time:", error)
        }

        return 0
    }, [isCacheEnabled, mediaId, episodeNumber])

    const getEpisodeContinuitySeekToWithCache = useCallback((
        episodeNumber: Nullish<number>,
        playerCurrentTime: Nullish<number>,
        playerDuration: Nullish<number>
    ) => {
        if (!isCacheEnabled || !mediaId || !episodeNumber || !playerDuration) return 0

        // First try cache
        const cachedSeekTime = getCachedSeekTime()
        if (cachedSeekTime > 0) {
            return cachedSeekTime
        }

        // Fallback to server data
        return getEpisodeContinuitySeekTo(episodeNumber, playerCurrentTime, playerDuration)
    }, [isCacheEnabled, mediaId, getCachedSeekTime, getEpisodeContinuitySeekTo])

    const syncCacheWithServer = useCallback(() => {
        if (!isCacheEnabled || !mediaId || !episodeNumber || !watchHistory?.item) return

        try {
            const serverData = watchHistory.item
            const cachedProgress = playbackProgressCacheService.getProgress(
                Number(mediaId),
                episodeNumber
            )

            // Use timeUpdated field for conflict resolution, fallback to current time
            const serverTimestamp = serverData.timeUpdated ? new Date(serverData.timeUpdated).getTime() : Date.now()
            const cacheTimestamp = cachedProgress?.lastUpdated || 0

            // Update cache if server data is newer or no cache exists
            if (!cachedProgress || serverTimestamp > cacheTimestamp) {
                playbackProgressCacheService.saveProgress(
                    Number(mediaId),
                    episodeNumber,
                    serverData.currentTime,
                    serverData.duration,
                    user?.id || null
                )

                logger("CACHE").info("Synced server data to cache", {
                    mediaId,
                    episodeNumber,
                    currentTime: serverData.currentTime,
                    duration: serverData.duration,
                    serverTimestamp,
                    cacheTimestamp,
                })
            }
        } catch (error) {
            logger("CACHE").error("Error syncing server data to cache:", error)
        }
    }, [isCacheEnabled, mediaId, episodeNumber, watchHistory, user?.id])

    // Sync server data to cache when it becomes available
    useEffect(() => {
        if (watchHistory?.item && !waitForWatchHistory) {
            syncCacheWithServer()
        }
    }, [watchHistory, waitForWatchHistory, syncCacheWithServer])

    return {
        watchHistory,
        waitForWatchHistory,
        getEpisodeContinuitySeekTo: getEpisodeContinuitySeekToWithCache,
        getCachedSeekTime,
        isCacheEnabled,
    }
}