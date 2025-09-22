import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"

/**
 * @description
 * - Listens to DOWNLOADED_CHAPTER events and re-fetches queries associated with media ID
 */
export function useMangaListener() {

    const qc = useQueryClient()

    useSSEEvents({
        enabled: true,
        onEvent: async (evt: any) => {
            if (evt.type === WSEvents.REFRESHED_MANGA_DOWNLOAD_DATA) {
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadsList.key] })
            }
            if (evt.type === WSEvents.CHAPTER_DOWNLOAD_QUEUE_UPDATED) {
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadData.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadQueue.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadsList.key] })
            }
        },
    })

}
