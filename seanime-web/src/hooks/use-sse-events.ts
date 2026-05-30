import { useEffect } from "react"
import { useAuth } from "@/contexts/auth-context"

interface SSEEvent {
    type: string
    payload: any
    userId?: number // Add user ID for user-specific events
}

interface UseSSEEventsOptions {
    enabled?: boolean
    onEvent?: (event: SSEEvent) => void
}

// Singleton/shared EventSource and subscriber registry
let __sse_source: EventSource | null = null
const __sse_subscribers = new Map<number, Set<(evt: SSEEvent) => void>>() // User-specific subscribers
const __global_subscribers = new Set<(evt: SSEEvent) => void>() // Global subscribers
let __sse_refCount = 0

function ensureEventSource() {
    if (__sse_source) return __sse_source
    const src = new EventSource("/api/v1/sse/events")
    src.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data) as SSEEvent

            // Route events based on user ID
            if (data.userId !== undefined) {
                // User-specific event
                const userSubscribers = __sse_subscribers.get(data.userId)
                if (userSubscribers) {
                    userSubscribers.forEach(cb => {
                        try { cb(data) } catch (e) { /* no-op */ }
                    })
                }
            } else {
                // Global event - this deployment has only registered (whitelisted)
                // users, so "global" means every connected user: notify the global
                // subscribers AND all per-user subscribers.
                __global_subscribers.forEach(cb => {
                    try { cb(data) } catch (e) { /* no-op */ }
                })
                __sse_subscribers.forEach(set => set.forEach(cb => {
                    try { cb(data) } catch (e) { /* no-op */ }
                }))
            }
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
    const { user } = useAuth()

    useEffect(() => {
        if (!enabled) return
        ensureEventSource()
        __sse_refCount += 1

        const subscriber = (evt: SSEEvent) => {
            try {
                // Only process events meant for this user or global events
                if (evt.userId === undefined || evt.userId === user?.id) {
                    onEvent?.(evt)
                }
            } catch (e) { /* no-op */ }
        }

        // Add subscriber to appropriate set
        if (user) {
            if (!__sse_subscribers.has(user.id)) {
                __sse_subscribers.set(user.id, new Set())
            }
            __sse_subscribers.get(user.id)!.add(subscriber)
        } else {
            __global_subscribers.add(subscriber)
        }

        return () => {
            // Remove subscriber from appropriate set
            if (user) {
                const userSubscribers = __sse_subscribers.get(user.id)
                if (userSubscribers) {
                    userSubscribers.delete(subscriber)
                    if (userSubscribers.size === 0) {
                        __sse_subscribers.delete(user.id)
                    }
                }
            } else {
                __global_subscribers.delete(subscriber)
            }

            __sse_refCount = Math.max(0, __sse_refCount - 1)
            if (__sse_refCount === 0 && __sse_source) {
                __sse_source.close()
                __sse_source = null
            }
        }
    }, [enabled, onEvent, user])
}

export default useSSEEvents
