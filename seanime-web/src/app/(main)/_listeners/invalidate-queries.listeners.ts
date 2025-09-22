import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"
import { useSSEEvents } from "@/hooks/use-sse-events"

export function useInvalidateQueriesListener() {

    const queryClient = useQueryClient()

    useSSEEvents({
        enabled: true,
        onEvent: async (evt: any) => {
            if (evt.type === WSEvents.INVALIDATE_QUERIES) {
                const data = (evt.payload as string[]) || []
                await Promise.all(data.map(async (queryKey) => {
                    await queryClient.invalidateQueries({ queryKey: [queryKey] })
                }))
            }
        },
    })

}
