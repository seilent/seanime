import { useServerMutation, useServerQuery } from "@/api/client/requests"
import { HibikeTorrent_AnimeTorrent } from "@/api/generated/types"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

export type SubspleaseStatus = {
    available: boolean
    episodeCount: number
    localCount: number
    toSync: HibikeTorrent_AnimeTorrent[] | null
    slug?: string
}

export type SubspleaseShow = {
    title: string
    slug: string
}

const SUBSPLEASE_STATUS_KEY = "subsplease-status"
const SUBSPLEASE_SHOWS_KEY = "subsplease-shows"

export function useGetSubspleaseStatus(media: any, enabled: boolean) {
    return useServerQuery<SubspleaseStatus, { media: any }>({
        endpoint: "/api/v1/subsplease/episodes",
        method: "POST",
        data: { media },
        queryKey: [SUBSPLEASE_STATUS_KEY, String(media?.id)],
        enabled: enabled,
        gcTime: 0,
    })
}

export function useGetSubspleaseShows(enabled: boolean) {
    return useServerQuery<SubspleaseShow[]>({
        endpoint: "/api/v1/subsplease/shows",
        method: "GET",
        queryKey: [SUBSPLEASE_SHOWS_KEY],
        enabled: enabled,
        staleTime: 1000 * 60 * 60,
    })
}

export function useLinkSubsplease(mediaId: number, onSuccess?: () => void) {
    const queryClient = useQueryClient()

    return useServerMutation<SubspleaseStatus, { mediaId: number, url: string }>({
        endpoint: "/api/v1/subsplease/link",
        method: "POST",
        mutationKey: ["subsplease-link", String(mediaId)],
        onSuccess: async () => {
            toast.success("Linked to SubsPlease")
            await queryClient.invalidateQueries({ queryKey: [SUBSPLEASE_STATUS_KEY, String(mediaId)] })
            onSuccess?.()
        },
    })
}
