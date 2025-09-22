import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { ExtensionRepo_UpdateData } from "@/api/generated/types"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"

/**
 * @description
 * - Re-fetches queries associated with extension data
 */
export function useExtensionListener() {

    const qc = useQueryClient()

    useSSEEvents({
        enabled: true,
        onEvent: async (evt: any) => {
            switch (evt.type) {
                case WSEvents.EXTENSIONS_RELOADED: {
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.ListAnimeTorrentProviderExtensions.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.ListMangaProviderExtensions.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.ListOnlinestreamProviderExtensions.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.ListExtensionData.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.GetAllExtensions.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.GetExtensionUserConfig.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.GetExtensionUpdateData.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.ListDevelopmentModeExtensions.key] })
                    break
                }
                case WSEvents.PLUGIN_UNLOADED: {
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.ListDevelopmentModeExtensions.key] })
                    break
                }
                case WSEvents.EXTENSION_UPDATES_FOUND: {
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.GetExtensionUpdateData.key] })
                    await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.EXTENSIONS.GetAllExtensions.key] })
                    break
                }
            }
        },
    })

}
