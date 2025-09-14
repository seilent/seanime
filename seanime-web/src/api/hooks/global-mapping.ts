import { useServerMutation, useServerQuery } from "@/api/client/requests"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { Models_GlobalAnimeFileMapping, Models_UnmappedFile } from "@/api/generated/types"
import { toast } from "sonner"

// Query hooks for fetching data

export function useGetUnmappedFiles() {
    return useServerQuery<Array<Models_UnmappedFile>>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.methods[0],
        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key],
        enabled: true,
    })
}

export function useGetIgnoredFiles() {
    return useServerQuery<Array<Models_UnmappedFile>>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.methods[0],
        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.key],
        enabled: true,
    })
}

export function useGetGlobalMappings() {
    return useServerQuery<Array<Models_GlobalAnimeFileMapping>>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.methods[0],
        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key],
        enabled: true,
    })
}

export function useGetProgressSyncStats() {
    return useServerQuery<Record<string, number>>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.methods[0],
        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.key],
        enabled: true,
        refetchInterval: 30000, // Refetch every 30 seconds
    })
}

export function useGetUserSubscriptions() {
    return useServerQuery<{ userId: number; subscribedIds: number[] }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.methods[0],
        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.key],
        enabled: true,
    })
}

export function useGetFilesForAnime(anilistId: number, enabled = true) {
    return useServerQuery<Array<string>>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetFilesForAnime.endpoint.replace(":anilistId", String(anilistId)),
        method: API_ENDPOINTS.GLOBAL_MAPPING.GetFilesForAnime.methods[0],
        queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetFilesForAnime.key, anilistId],
        enabled: enabled && anilistId > 0,
    })
}

// Mutation hooks for modifying data

export function useMapFileToAniList() {
    return useServerMutation<boolean, {
        filePath: string
        anilistId: number
        title: string
        year: number
        episodeNumber: number
    }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.MapFileToAniList.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.MapFileToAniList.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.MapFileToAniList.key],
        onSuccess: async () => {
            toast.success("File mapped successfully")
            // Invalidate relevant queries
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key,
                API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to map file: " + error?.message)
        },
    })
}

export function useIgnoreFile() {
    return useServerMutation<boolean, { filePath: string }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.IgnoreFile.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.IgnoreFile.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.IgnoreFile.key],
        onSuccess: async () => {
            toast.success("File ignored successfully")
            // Invalidate relevant queries
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key,
                API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to ignore file: " + error?.message)
        },
    })
}

export function useUnignoreFile() {
    return useServerMutation<boolean, { filePath: string }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.UnignoreFile.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.UnignoreFile.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.UnignoreFile.key],
        onSuccess: async () => {
            toast.success("File unignored successfully")
            // Invalidate relevant queries
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.key,
                API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to unignore file: " + error?.message)
        },
    })
}

export function useRemoveMapping() {
    return useServerMutation<boolean, { filePath: string }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.RemoveMapping.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.RemoveMapping.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.RemoveMapping.key],
        onSuccess: async () => {
            toast.success("Mapping removed successfully")
            // Invalidate relevant queries
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key,
                API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to remove mapping: " + error?.message)
        },
    })
}

export function useRetryFailedSync() {
    return useServerMutation<boolean, void>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.RetryFailedSyncItems.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.RetryFailedSyncItems.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.RetryFailedSyncItems.key],
        onSuccess: async () => {
            toast.success("Failed sync items queued for retry")
            // Invalidate progress stats
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to retry sync items: " + error?.message)
        },
    })
}

export function useSubscribeToAnime() {
    return useServerMutation<boolean, { anilistId: number }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.SubscribeToAnime.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.SubscribeToAnime.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.SubscribeToAnime.key],
        onSuccess: async () => {
            toast.success("Subscribed to anime")
            // Invalidate user subscriptions
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to subscribe: " + error?.message)
        },
    })
}

export function useUnsubscribeFromAnime() {
    return useServerMutation<boolean, { anilistId: number }>({
        endpoint: API_ENDPOINTS.GLOBAL_MAPPING.UnsubscribeFromAnime.endpoint,
        method: API_ENDPOINTS.GLOBAL_MAPPING.UnsubscribeFromAnime.methods[0],
        mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.UnsubscribeFromAnime.key],
        onSuccess: async () => {
            toast.success("Unsubscribed from anime")
            // Invalidate user subscriptions
            await Promise.all([
                API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.key,
            ])
        },
        onError: (error) => {
            toast.error("Failed to unsubscribe: " + error?.message)
        },
    })
}