import { useSyncIsActive } from "@/app/(main)/_atoms/sync.atoms"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import React from "react"

export function useSyncListener() {
    useSSEEvents({
        enabled: true,
        onEvent: (evt: any) => {
            // Sync events are no longer handled since offline/local tracking was removed
        },
    })

    const { setSyncIsActive } = useSyncIsActive()

    React.useEffect(() => {
        setSyncIsActive(false)
    }, [])
}
