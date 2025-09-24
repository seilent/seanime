import { useMediastreamCurrentFile } from "@/app/(main)/mediastream/_lib/mediastream.atoms"
import { clientIdAtom } from "@/app/websocket-provider"
import { logger } from "@/lib/helpers/debug"
import { useAtomValue } from "jotai"
import { useRouter } from "next/navigation"
import React from "react"

export function useHandlePlayMedia() {
    const router = useRouter()
    const clientId = useAtomValue(clientIdAtom)
    const { setFilePath: setMediastreamFilePath } = useMediastreamCurrentFile()

    function playMediaFile({ path, mediaId }: { path: string, mediaId: number }) {
        logger("PLAY MEDIA").info("Playing media file using web player", path)

        // Always use media streaming (web player)
        setMediastreamFilePath(path)
        React.startTransition(() => {
            router.push(`/mediastream?id=${mediaId}`)
        })
    }

    return {
        playMediaFile,
    }
}
