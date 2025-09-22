import { useSSEEvents } from "@/hooks/use-sse-events"
import { WSEvents } from "@/lib/server/ws-events"
import { toast } from "sonner"

export function useMiscEventListeners() {

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
                case WSEvents.ERROR_TOAST: toast.error(data); break
                case WSEvents.CONSOLE_LOG: console.log(data); break
                case WSEvents.CONSOLE_WARN: console.warn(data); break
            }
        },
    })

}
