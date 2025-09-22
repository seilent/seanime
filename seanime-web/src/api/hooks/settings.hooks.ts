import { useServerMutation, useServerQuery } from "@/api/client/requests"
import { GettingStarted_Variables, SaveAutoDownloaderSettings_Variables, SaveSettings_Variables } from "@/api/generated/endpoint.types"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { Models_Settings, Status } from "@/api/generated/types"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

export function useGetSettings() {
    return useServerQuery<Models_Settings>({
        endpoint: API_ENDPOINTS.SETTINGS.GetSettings.endpoint,
        method: API_ENDPOINTS.SETTINGS.GetSettings.methods[0],
        queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key],
        enabled: true,
    })
}

export function useGettingStarted() {
    const queryClient = useQueryClient()

    return useServerMutation<Status, GettingStarted_Variables>({
        endpoint: API_ENDPOINTS.SETTINGS.GettingStarted.endpoint,
        method: API_ENDPOINTS.SETTINGS.GettingStarted.methods[0],
        mutationKey: [API_ENDPOINTS.SETTINGS.GettingStarted.key],
        onSuccess: async () => {
            console.log("Getting Started - API call succeeded")
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.STATUS.GetStatus.key] })
        },
        onError: (error) => {
            console.error("Getting Started - API call failed:", error)
            toast.error(`Setup failed: ${error.message || 'Unknown error'}`)
        },
    })
}

export function useSaveSettings() {
    const queryClient = useQueryClient()

    return useServerMutation<Status, SaveSettings_Variables>({
        endpoint: API_ENDPOINTS.SETTINGS.SaveSettings.endpoint,
        method: API_ENDPOINTS.SETTINGS.SaveSettings.methods[0],
        mutationKey: [API_ENDPOINTS.SETTINGS.SaveSettings.key],
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.STATUS.GetStatus.key] })
            toast.success("Settings saved")
        },
    })
}

export function useSaveAutoDownloaderSettings() {
    const queryClient = useQueryClient()

    return useServerMutation<boolean, SaveAutoDownloaderSettings_Variables>({
        endpoint: API_ENDPOINTS.SETTINGS.SaveAutoDownloaderSettings.endpoint,
        method: API_ENDPOINTS.SETTINGS.SaveAutoDownloaderSettings.methods[0],
        mutationKey: [API_ENDPOINTS.SETTINGS.SaveAutoDownloaderSettings.key],
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.STATUS.GetStatus.key] })
            toast.success("Settings saved")
        },
    })
}

// Multi-user settings hooks
export function useGetGlobalSettings() {
    return useServerQuery<any>({
        endpoint: "/api/v1/settings/global",
        method: "GET",
        queryKey: ["global-settings"],
        enabled: false, // Only enable for admin users
    })
}

export function useUpdateGlobalSettings() {
    const queryClient = useQueryClient()

    return useServerMutation<any, any>({
        endpoint: "/api/v1/settings/global",
        method: "PUT",
        mutationKey: ["update-global-settings"],
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: ["global-settings"] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.STATUS.GetStatus.key] })
            toast.success("Global settings saved")
        },
    })
}

export function useGetUserSettings() {
    return useServerQuery<Models_Settings>({
        endpoint: "/api/v1/settings/user",
        method: "GET",
        queryKey: ["user-settings"],
        enabled: true,
    })
}

export function useUpdateUserSettings() {
    const queryClient = useQueryClient()

    return useServerMutation<Models_Settings, Models_Settings>({
        endpoint: "/api/v1/settings/user",
        method: "PUT",
        mutationKey: ["update-user-settings"],
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: ["user-settings"] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.STATUS.GetStatus.key] })
            toast.success("Personal settings saved")
        },
    })
}
