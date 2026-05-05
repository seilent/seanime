import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { toast } from "sonner"
import { useMemo } from "react"
import { ErrorDeduplicator } from "@/lib/error-deduplicator"

// Singleton instance
const errorDeduplicator = new ErrorDeduplicator(
	5000,  // 5 second dedupe window
	100    // max 100 cached errors
)

export function useMiscEventListeners() {
	// Optional: Expose deduplicator controls for debugging/settings
	const deduplicator = useMemo(() => errorDeduplicator, [])

	useSSEEvents({
		enabled: true,
		onEvent: (evt: any) => {
			const t = evt.type
			const data = evt.payload as string
			if (!data) return
			switch (t) {
				case WSEvents.INFO_TOAST: toast.info(data); break
				case WSEvents.SUCCESS_TOAST: toast.success(data); break
				case WSEvents.WARNING_TOAST: toast.warning(data); break
				case WSEvents.ERROR_TOAST:
					// Check if we should show this error
					if (deduplicator.shouldShowError(data)) {
						toast.error(data)
					} else {
						// Optional: Log to console that error was deduped
						const count = deduplicator.getErrorCount(data)
						console.log(`[ErrorDeduplicator] Suppressed duplicate error (count: ${count}):`, data)
					}
					break
				case WSEvents.CONSOLE_LOG: console.log(data); break
				case WSEvents.CONSOLE_WARN: console.warn(data); break
			}
		},
	})

	return { deduplicator }
}
