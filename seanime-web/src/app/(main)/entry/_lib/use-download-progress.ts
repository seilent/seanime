import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { useCallback, useState } from "react"

export type DownloadProgressItem = { mediaId: number; episode: number; progress: number }

export function useDownloadProgress(): DownloadProgressItem[] {
    const [items, setItems] = useState<DownloadProgressItem[]>([])
    useSSEEvents({
        onEvent: useCallback((event: { type: string; payload: any }) => {
            if (event.type === WSEvents.DOWNLOAD_PROGRESS) {
                setItems(event.payload as DownloadProgressItem[])
            }
        }, []),
    })
    return items
}
