import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { useSmartLibraryScan } from "@/app/(main)/(library)/_hooks/use-smart-library-scan"

import { useSSEEvents } from "@/hooks/use-sse-events"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { PageWrapper } from "@/components/shared/page-wrapper"
import { Button, CloseButton } from "@/components/ui/button"
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Spinner } from "@/components/ui/loading-spinner"
import { useBoolean } from "@/hooks/use-disclosure"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"
import React, { useState } from "react"
import { BiSolidBinoculars } from "react-icons/bi"
import { FiSearch } from "react-icons/fi"
import { toast } from "sonner"

type LibraryWatcherProps = {
    children?: React.ReactNode
}

export function LibraryWatcher(props: LibraryWatcherProps) {

    const {
        children,
        ...rest
    } = props

    const qc = useQueryClient()
    const serverStatus = useServerStatus()
    const [fileEvent, setFileEvent] = useState<string | null>(null)
    const fileAdded = useBoolean(false)
    const fileRemoved = useBoolean(false)
    const autoScanning = useBoolean(false)
    const [progress, setProgress] = useState(0)

    const { scanLibrary, isPending: isScanning } = useSmartLibraryScan()

    useSSEEvents({
        enabled: true,
        onEvent: (evt: any) => {
            const type = evt.type
            const data = evt.payload
            switch (type) {
                case WSEvents.LIBRARY_WATCHER_FILE_ADDED: {
                    if (!serverStatus?.settings?.library?.autoScan) {
                        fileAdded.on(); setFileEvent(data as string)
                    }
                    break
                }
                case WSEvents.LIBRARY_WATCHER_FILE_REMOVED: {
                    if (!serverStatus?.settings?.library?.autoScan) {
                        fileRemoved.on(); setFileEvent(data as string)
                    }
                    break
                }
                case WSEvents.SCAN_PROGRESS: {
                    setFileEvent(null); fileAdded.off(); fileRemoved.off()
                    const p = data as number
                    setProgress(p)
                    if (p === 100) setTimeout(() => setProgress(0), 2000)
                    break
                }
                case WSEvents.AUTO_SCAN_STARTED: {
                    autoScanning.on(); break
                }
                case WSEvents.AUTO_SCAN_COMPLETED: {
                    autoScanning.off(); toast.success("Library scanned")
                    qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.key] })
                    qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetMissingEpisodes.key] })
                    qc.invalidateQueries({ queryKey: [API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderItems.key] })
                    break
                }
            }
        },
    })

    function handleCancel() {
        setFileEvent(null)
        fileAdded.off()
        fileRemoved.off()
    }

    if (autoScanning.active && progress > 0) {
        return (
            <div className="z-50 fixed bottom-4 right-4">
                <PageWrapper>
                    <Card className="w-fit max-w-[400px]">
                        <CardHeader>
                            <CardDescription className="flex items-center gap-2 text-base">
                                <Spinner className="size-6" /> {progress}% Refreshing your library...
                            </CardDescription>
                        </CardHeader>
                    </Card>
                </PageWrapper>
            </div>
        )
    } else if (!!fileEvent) {
        return (
            <div className="z-50 fixed bottom-4 right-4">
                <PageWrapper>
                    <Card className="w-full max-w-[400px] min-h-[150px] relative">
                        <CardHeader>
                            <CardTitle className="text-lg flex items-center gap-2">
                                <BiSolidBinoculars className="text-brand-400" />
                                Library watcher
                            </CardTitle>
                            <CardDescription className="flex items-center gap-2 text-base">
                                A change has been detected in your library, refresh your entries.
                            </CardDescription>
                        </CardHeader>
                        <CardFooter>
                            <Button
                                intent="primary-outline"
                                leftIcon={<FiSearch />}
                                size="sm"
                                onClick={() => scanLibrary()}
                                loading={isScanning}
                                className="rounded-full"
                            >
                                Scan your library
                            </Button>
                        </CardFooter>
                        <CloseButton className="absolute top-2 right-2" onClick={handleCancel} />
                    </Card>
                </PageWrapper>
            </div>
        )
    }

    return null
}
