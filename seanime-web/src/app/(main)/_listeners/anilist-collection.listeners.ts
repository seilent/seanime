import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"

/**
 * @description
 * - Listens to REFRESHED_ANILIST_COLLECTION events and re-fetches queries associated with AniList collection.
 */
export function useAnimeCollectionListener() {

    const qc = useQueryClient()

    useSSEEvents({
        enabled: true,
        onEvent: async (evt: any) => {
            if (evt.type === WSEvents.REFRESHED_ANILIST_ANIME_COLLECTION) {
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANILIST.GetAnimeCollection.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANILIST.GetRawAnimeCollection.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetMissingEpisodes.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetAnimeCollectionSchedule.key] })
            }
            if (evt.type === WSEvents.REFRESHED_ANILIST_MANGA_COLLECTION) {
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.MANGA.GetAnilistMangaCollection.key] })
                await qc.invalidateQueries({ queryKey: [API_ENDPOINTS.MANGA.GetMangaEntry.key] })
            }
        },
    })

}
