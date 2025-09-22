import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { useGetAutoDownloaderItems } from "@/api/hooks/auto_downloader.hooks"
import { autoDownloaderItemsAtom } from "@/app/(main)/_atoms/autodownloader.atoms"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"
import { useSetAtom } from "jotai/react"
import { usePathname } from "next/navigation"
import { useEffect } from "react"

/**
 * @description
 * - When the user is not on the main page, send a request to get auto downloader queue items
 */
export function useAutoDownloaderItemListener() {
    const pathname = usePathname()
    const setter = useSetAtom(autoDownloaderItemsAtom)
    const qc = useQueryClient()

    const { data } = useGetAutoDownloaderItems(pathname !== "/auto-downloader")

    useSSEEvents({
        enabled: true,
        onEvent: (evt: any) => {
            if (evt.type === WSEvents.AUTO_DOWNLOADER_ITEM_ADDED) {
                qc.invalidateQueries({ queryKey: [API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderItems.key] })
            }
        },
    })

    useEffect(() => {
        setter(data ?? [])
    }, [data])

    return null
}
