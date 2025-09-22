import { useEffect } from "react"

interface SSEEvent {
    type: string
    payload: any
}

interface UseSSEEventsOptions {
    enabled?: boolean
    onEvent?: (event: SSEEvent) => void
}

// Singleton/shared EventSource and subscriber registry
let __sse_source: EventSource | null = null
const __sse_subscribers = new Set<(evt: SSEEvent) => void>()
let __sse_refCount = 0

function ensureEventSource() {
    if (__sse_source) return __sse_source
    const src = new EventSource("/api/v1/sse/events")
    src.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data) as SSEEvent
            // Fan out to all subscribers
            __sse_subscribers.forEach(cb => {
                try { cb(data) } catch (e) { /* no-op */ }
            })
        } catch (error) {
            console.error("Failed to parse SSE event:", error)
        }
    }
    src.onerror = (error) => {
        console.error("SSE connection error:", error)
        // EventSource auto-reconnects; keep it alive
    }
    __sse_source = src
    return src
}

export function useSSEEvents(options: UseSSEEventsOptions = {}) {
    const { enabled = true, onEvent } = options

    useEffect(() => {
        if (!enabled) return
        ensureEventSource()
        __sse_refCount += 1
        const subscriber = (evt: SSEEvent) => {
            try {
                onEvent?.(evt)
            } catch (e) { /* no-op */ }
        }
        __sse_subscribers.add(subscriber)

        return () => {
            __sse_subscribers.delete(subscriber)
            __sse_refCount = Math.max(0, __sse_refCount - 1)
            if (__sse_refCount === 0 && __sse_source) {
                __sse_source.close()
                __sse_source = null
            }
        }
    }, [enabled, onEvent])
}

export default useSSEEvents
