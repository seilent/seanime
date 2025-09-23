import { useServerMutation, useServerQuery } from "@/api/client/requests"
import { PreloadMediastreamMediaContainer_Variables, RequestMediastreamMediaContainer_Variables } from "@/api/generated/endpoint.types"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { Mediastream_MediaContainer } from "@/api/generated/types"
import { logger } from "@/lib/helpers/debug"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

export type ClientMediaSettings = {
    directPlayOnly: boolean
}

// Client media (per-user)
export function useGetClientMediaSettings(enabled?: boolean) {
    return useServerQuery<ClientMediaSettings>({
        endpoint: "/api/v1/client-media/settings",
        method: "GET",
        queryKey: ["CLIENT-MEDIA-get-settings"],
        enabled,
    })
}

export function useSaveClientMediaSettings() {
    const qc = useQueryClient()
    return useServerMutation<ClientMediaSettings, { settings: ClientMediaSettings }>({
        endpoint: "/api/v1/client-media/settings",
        method: "PATCH",
        mutationKey: ["CLIENT-MEDIA-save-settings"],
        onSuccess: async () => {
            await qc.invalidateQueries({ queryKey: ["CLIENT-MEDIA-get-settings"] })
            await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.STATUS.GetStatus.key] })
            toast.success("Playback settings saved")
        },
    })
}

export function useRequestMediastreamMediaContainer(variables: Partial<RequestMediastreamMediaContainer_Variables>, enabled: boolean) {
    return useServerQuery<Mediastream_MediaContainer, RequestMediastreamMediaContainer_Variables>({
        endpoint: API_ENDPOINTS.MEDIASTREAM.RequestMediastreamMediaContainer.endpoint,
        method: API_ENDPOINTS.MEDIASTREAM.RequestMediastreamMediaContainer.methods[0],
        queryKey: [API_ENDPOINTS.MEDIASTREAM.RequestMediastreamMediaContainer.key, variables?.path, variables?.streamType],
        data: variables as RequestMediastreamMediaContainer_Variables,
        enabled: !!variables.path && !!variables.streamType && enabled,
    })
}

export function usePreloadMediastreamMediaContainer() {
    return useServerMutation<boolean, PreloadMediastreamMediaContainer_Variables>({
        endpoint: API_ENDPOINTS.MEDIASTREAM.PreloadMediastreamMediaContainer.endpoint,
        method: API_ENDPOINTS.MEDIASTREAM.PreloadMediastreamMediaContainer.methods[0],
        mutationKey: [API_ENDPOINTS.MEDIASTREAM.PreloadMediastreamMediaContainer.key],
        onSuccess: async () => {
            logger("MEDIASTREAM").success("Preloaded mediastream media container")
        },
    })
}

