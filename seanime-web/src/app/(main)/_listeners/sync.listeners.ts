import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { Local_QueueState } from "@/api/generated/types"
import { useSyncIsActive } from "@/app/(main)/_atoms/sync.atoms"
import { useSSEEvents } from "@/hooks/use-sse-events"
import { logger } from "@/lib/helpers/debug"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"
import React from "react"

export function useSyncListener() {
    const qc = useQueryClient()

    useSSEEvents({
        enabled: true,
        onEvent: (evt: any) => {
            if (evt.type === WSEvents.SYNC_LOCAL_FINISHED) {
                qc.invalidateQueries({ queryKey: [API_ENDPOINTS.LOCAL.LocalGetTrackedMediaItems.key] })
            }
        },
    })

    const [queueState, setQueueState] = React.useState<Local_QueueState | null>(null)
    useSSEEvents({
        enabled: true,
        onEvent: (evt: any) => {
            if (evt.type === WSEvents.SYNC_LOCAL_QUEUE_STATE) {
                const data = evt.payload as Local_QueueState
                logger("SYNC").info("Queue state", queueState)
                setQueueState(data)
            }
        },
    })

    const { setSyncIsActive } = useSyncIsActive()

    React.useEffect(() => {
        setSyncIsActive(!!queueState && (Object.keys(queueState.animeTasks!).length > 0 || Object.keys(queueState.mangaTasks!).length > 0))
    }, [queueState])
}
