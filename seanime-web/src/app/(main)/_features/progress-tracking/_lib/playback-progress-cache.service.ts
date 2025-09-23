import { atomWithStorage } from "jotai/utils"
import { atom } from "jotai"
import { Nullish } from "@/api/generated/types"

export interface PlaybackProgressCacheEntry {
    mediaId: number
    episodeNumber: number
    currentTime: number
    duration: number
    lastUpdated: number
    expiresAt: number
}

export interface PlaybackProgressCache {
    version: string
    lastSync: number
    userId: number | null
    entries: Record<string, PlaybackProgressCacheEntry>
    pendingSync: Array<{
        mediaId: number
        episodeNumber: number
        currentTime: number
        duration: number
        timestamp: number
    }>
}

const CACHE_VERSION = "1.0"
const CACHE_TTL_MS = 24 * 60 * 60 * 1000 // 24 hours
const SYNC_INTERVAL_MS = 30 * 1000 // 30 seconds

export const initialCacheState: PlaybackProgressCache = {
    version: CACHE_VERSION,
    lastSync: 0,
    userId: null,
    entries: {},
    pendingSync: [],
}

export const playbackProgressCacheAtom = atomWithStorage(
    "seanime-playback-progress-cache",
    initialCacheState,
    undefined,
    {
        getOnInit: true,
    }
)

export const playbackProgressSyncStateAtom = atom({
    isSyncing: false,
    lastSyncError: null as string | null,
})

export class PlaybackProgressCacheService {
    private cacheAtom = playbackProgressCacheAtom
    private syncInterval: NodeJS.Timeout | null = null

    generateCacheKey(mediaId: number, episodeNumber: number): string {
        return `${mediaId}_${episodeNumber}`
    }

    saveProgress(
        mediaId: number,
        episodeNumber: number,
        currentTime: number,
        duration: number,
        userId: number | null = null
    ): void {
        const now = Date.now()
        const key = this.generateCacheKey(mediaId, episodeNumber)

        // This would be called from a component that has access to the atom
        // For now, we'll update the cache directly through localStorage
        const cache = this.getCache()

        cache.entries[key] = {
            mediaId,
            episodeNumber,
            currentTime,
            duration,
            lastUpdated: now,
            expiresAt: now + CACHE_TTL_MS,
        }

        if (userId) {
            cache.userId = userId
        }

        // Add to sync queue
        cache.pendingSync.push({
            mediaId,
            episodeNumber,
            currentTime,
            duration,
            timestamp: now,
        })

        this.saveCache(cache)
    }

    getProgress(mediaId: number, episodeNumber: number): PlaybackProgressCacheEntry | null {
        const cache = this.getCache()
        const key = this.generateCacheKey(mediaId, episodeNumber)
        const entry = cache.entries[key]

        if (!entry) return null

        // Check if expired
        if (Date.now() > entry.expiresAt) {
            this.removeExpiredEntry(key)
            return null
        }

        return entry
    }

    getCache(): PlaybackProgressCache {
        if (typeof window === "undefined") return initialCacheState

        try {
            const cached = localStorage.getItem("seanime-playback-progress-cache")
            if (!cached) return initialCacheState

            const parsed = JSON.parse(cached) as PlaybackProgressCache

            // Validate cache structure
            if (!parsed.version || parsed.version !== CACHE_VERSION) {
                console.warn("Cache version mismatch, resetting cache")
                this.clearCache()
                return initialCacheState
            }

            return parsed
        } catch (error) {
            console.error("Error reading cache:", error)
            return initialCacheState
        }
    }

    saveCache(cache: PlaybackProgressCache): void {
        if (typeof window === "undefined") return

        try {
            localStorage.setItem("seanime-playback-progress-cache", JSON.stringify(cache))
        } catch (error) {
            console.error("Error saving cache:", error)
        }
    }

    removeExpiredEntry(key: string): void {
        const cache = this.getCache()
        delete cache.entries[key]
        this.saveCache(cache)
    }

    clearExpiredEntries(): void {
        const cache = this.getCache()
        const now = Date.now()
        let hasChanges = false

        Object.keys(cache.entries).forEach(key => {
            if (now > cache.entries[key].expiresAt) {
                delete cache.entries[key]
                hasChanges = true
            }
        })

        if (hasChanges) {
            this.saveCache(cache)
        }
    }

    clearCache(): void {
        if (typeof window === "undefined") return

        localStorage.removeItem("seanime-playback-progress-cache")
    }

    getPendingSyncItems(): Array<{
        mediaId: number
        episodeNumber: number
        currentTime: number
        duration: number
        timestamp: number
    }> {
        const cache = this.getCache()
        return cache.pendingSync
    }

    clearSyncQueue(): void {
        const cache = this.getCache()
        cache.pendingSync = []
        cache.lastSync = Date.now()
        this.saveCache(cache)
    }

    startSyncInterval(callback: () => void): void {
        if (this.syncInterval) {
            clearInterval(this.syncInterval)
        }

        this.syncInterval = setInterval(callback, SYNC_INTERVAL_MS)
    }

    stopSyncInterval(): void {
        if (this.syncInterval) {
            clearInterval(this.syncInterval)
            this.syncInterval = null
        }
    }
}

export const playbackProgressCacheService = new PlaybackProgressCacheService()