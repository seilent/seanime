import { useSSEEvents } from "@/hooks/use-sse-events"
import { ProgressBar } from "@/components/ui/progress-bar"
import { logger } from "@/lib/helpers/debug"
import { WSEvents } from "@/lib/server/ws-events"
import React from "react"

const log = logger("IndefiniteLoader")

export function TopIndefiniteLoader() {

    const [showStack, setShowStack] = React.useState<string[]>([])

    // Empty after 3 minutes
    // timeout resets each time a new loader is shown
    React.useEffect(() => {
        const timeout = setTimeout(() => {
            setShowStack([])
        }, 3 * 60 * 1000)
        return () => clearTimeout(timeout)
    }, [showStack])

    useSSEEvents({
        enabled: true,
        onEvent: (evt: any) => {
            const type = evt.type
            const data = evt.payload as string
            if (type === WSEvents.SHOW_INDEFINITE_LOADER && data) {
                log.info("Showing indefinite loader", data)
                setShowStack(prev => (prev.includes(data) ? prev : [...prev, data]))
            }
            if (type === WSEvents.HIDE_INDEFINITE_LOADER && data) {
                log.info("Hiding indefinite loader", data)
                setShowStack(prev => prev.filter(item => item !== data))
            }
        },
    })

    return (
        <>
            {showStack.length > 0 && <div className="w-full bg-gray-950 fixed top-0 left-0 z-[100]">
                <ProgressBar size="xs" isIndeterminate />
            </div>}
        </>
    )
}
