import { useGetStatus } from "@/api/hooks/status.hooks"
import { GettingStartedPage } from "@/app/(main)/_features/getting-started/getting-started-page"
import { useServerStatus, useSetServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { LoadingOverlayWithLogo } from "@/components/shared/loading-overlay-with-logo"
import { LuffyError } from "@/components/shared/luffy-error"
import { logger } from "@/lib/helpers/debug"
import { WSEvents } from "@/lib/server/ws-events"
import { usePathname } from "next/navigation"
import React from "react"
import { useWebsocketMessageListener } from "./_hooks/handle-websockets"
import { useSSEEvents } from "@/hooks/use-sse-events"

type ServerDataWrapperProps = {
    host: string
    children?: React.ReactNode
}

export function ServerDataWrapper(props: ServerDataWrapperProps) {

    const {
        host,
        children,
        ...rest
    } = props

    const pathname = usePathname()
    const serverStatus = useServerStatus()
    const setServerStatus = useSetServerStatus()
    const { data: _serverStatus, isLoading, refetch } = useGetStatus()

    React.useEffect(() => {
        if (_serverStatus) {
            // logger("SERVER").info("Server status", _serverStatus)
            setServerStatus(_serverStatus)
        }
    }, [_serverStatus])

    useWebsocketMessageListener({
        type: WSEvents.ANILIST_DATA_LOADED,
        onMessage: () => {
            logger("Data Wrapper").info("Anilist data loaded, refetching server status")
            refetch()
        },
    })

    // Enable global SSE for real-time updates across all pages
    useSSEEvents({
        enabled: true,
        onEvent: (event) => {
            logger("SSE Global").info("Received SSE event:", event.type)
        }
    })


    // Refetch the server status every 2 seconds if serverReady is false
    // This is a fallback to the websocket
    const intervalId = React.useRef<NodeJS.Timeout | null>(null)
    React.useEffect(() => {
        if (!serverStatus?.serverReady) {
            intervalId.current = setInterval(() => {
                logger("Data Wrapper").info("Refetching server status")
                refetch()
            }, 2000)
        }
        return () => {
            logger("Data Wrapper").info("Clearing interval")
            if (intervalId.current) {
                clearInterval(intervalId.current)
                intervalId.current = null
            }
        }
    }, [serverStatus?.serverReady])

    /**
     * If the server status is loading or doesn't exist, show the loading overlay
     */
    if (isLoading || !serverStatus) return <LoadingOverlayWithLogo />
    if (!serverStatus?.serverReady) return <LoadingOverlayWithLogo title="L o a d i n g" />

    /**
     * If the pathname is /auth/callback, show the callback page
     */
    if (pathname.startsWith("/auth/callback")) return children

    /**
     * Show getting started page if setup is not completed
     */
    if (!serverStatus?.settings?.setupCompleted) {
        return <GettingStartedPage status={serverStatus} />
    }

    /**
     * If the app is updating, show a different screen
     */
    if (serverStatus?.updating) {
        return <div className="container max-w-3xl py-10">
            <div className="mb-4 flex justify-center w-full">
                <img src="/logo_2.png" alt="logo" className="w-36 h-auto" />
            </div>
            <p className="text-center text-lg">
                Seanime is currently updating. Refresh the page once the update is complete and the connection has been reestablished.
            </p>
        </div>
    }

    /**
     * Check feature flag routes
     */

    if (!serverStatus?.mediastreamSettings?.transcodeEnabled && pathname.startsWith("/mediastream")) {
        return <LuffyError title="Transcoding not enabled" />
    }


    return children
}
