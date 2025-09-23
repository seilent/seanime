import { useEffect, useCallback } from "react"
import { playbackProgressCacheService } from "./playback-progress-cache.service"
import { logger } from "@/lib/helpers/debug"

export function useCacheMaintenance() {
    const performCleanup = useCallback(() => {
        try {
            const beforeCount = Object.keys(playbackProgressCacheService.getCache().entries).length

            playbackProgressCacheService.clearExpiredEntries()

            const afterCount = Object.keys(playbackProgressCacheService.getCache().entries).length
            const removedCount = beforeCount - afterCount

            if (removedCount > 0) {
                logger("CACHE").info(`Cache cleanup completed`, {
                    removedEntries: removedCount,
                    remainingEntries: afterCount,
                })
            }
        } catch (error) {
            logger("CACHE").error("Error during cache cleanup:", error)
        }
    }, [])

    const clearAllCache = useCallback(() => {
        try {
            playbackProgressCacheService.clearCache()
            logger("CACHE").info("All cache cleared")
        } catch (error) {
            logger("CACHE").error("Error clearing cache:", error)
        }
    }, [])

    const getCacheStats = useCallback(() => {
        try {
            const cache = playbackProgressCacheService.getCache()
            const entryCount = Object.keys(cache.entries).length
            const pendingSyncCount = cache.pendingSync.length
            const totalSize = new Blob([JSON.stringify(cache)]).size

            // Calculate oldest and newest entries
            const timestamps = Object.values(cache.entries).map(entry => entry.lastUpdated)
            const oldestEntry = timestamps.length > 0 ? Math.min(...timestamps) : null
            const newestEntry = timestamps.length > 0 ? Math.max(...timestamps) : null

            return {
                entryCount,
                pendingSyncCount,
                totalSizeBytes: totalSize,
                oldestEntryTimestamp: oldestEntry,
                newestEntryTimestamp: newestEntry,
                lastSync: cache.lastSync,
                cacheVersion: cache.version,
                userId: cache.userId,
            }
        } catch (error) {
            logger("CACHE").error("Error getting cache stats:", error)
            return null
        }
    }, [])

    // Perform cleanup on mount and periodically
    useEffect(() => {
        // Initial cleanup
        performCleanup()

        // Set up periodic cleanup (every hour)
        const cleanupInterval = setInterval(performCleanup, 60 * 60 * 1000)

        return () => {
            clearInterval(cleanupInterval)
        }
    }, [performCleanup])

    // Perform cleanup when page becomes visible again
    useEffect(() => {
        const handleVisibilityChange = () => {
            if (document.visibilityState === "visible") {
                performCleanup()
            }
        }

        document.addEventListener("visibilitychange", handleVisibilityChange)

        return () => {
            document.removeEventListener("visibilitychange", handleVisibilityChange)
        }
    }, [performCleanup])

    return {
        performCleanup,
        clearAllCache,
        getCacheStats,
    }
}