import { Nullish } from "@/api/generated/types"
import { atom } from "jotai"
import { useAtom } from "jotai/react"
import { FaShareFromSquare } from "react-icons/fa6"
import { PiVideoFill } from "react-icons/pi"
import { useUserPreferences, useUserScopedAtom } from "./user-scoped-atoms"

export const enum ElectronPlaybackMethod {
    NativePlayer = "nativePlayer", // Desktop media player or Integrated player (media streaming)
    Default = "default", // Desktop media player, media streaming or external player link
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

export const enum PlaybackDownloadedMedia {
    Default = "default", // Desktop media player or Integrated player (media streaming)
    ExternalPlayerLink = "externalPlayerLink", // External player link
}


export const playbackDownloadedMediaOptions = [
    {
        label: <div className="flex items-center gap-4 md:gap-2 w-full">
            <PiVideoFill className="text-2xl flex-none" />
            <p className="max-w-[90%]">Desktop media player or Transcoding / Direct Play</p>
        </div>, value: PlaybackDownloadedMedia.Default,
    },
    {
        label: <div className="flex items-center gap-4 md:gap-2 w-full">
            <FaShareFromSquare className="text-2xl flex-none" />
            <p className="max-w-[90%]">External player link</p>
        </div>, value: PlaybackDownloadedMedia.ExternalPlayerLink,
    },
]

// Removed atomWithStorage - now using server-side user preferences

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

export const enum PlaybackTorrentStreaming {
    Default = "default", // Desktop media player
    ExternalPlayerLink = "externalPlayerLink",
}

export const playbackTorrentStreamingOptions = [
    {
        label: <div className="flex items-center gap-4 md:gap-2 w-full">
            <PiVideoFill className="text-2xl flex-none" />
            <p className="max-w-[90%]">Desktop media player</p>
        </div>, value: PlaybackTorrentStreaming.Default,
    },
    {
        label: <div className="flex items-center gap-4 md:gap-2 w-full">
            <FaShareFromSquare className="text-2xl flex-none" />
            <p className="max-w-[90%]">External player link</p>
        </div>, value: PlaybackTorrentStreaming.ExternalPlayerLink,
    },
]



//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

export function useCurrentDevicePlaybackSettings() {

    const [downloadedMediaPlayback, setDownloadedMediaPlayback] = useUserPreferences("playback_downloaded_media", PlaybackDownloadedMedia.Default)
    const [electronPlaybackMethod, setElectronPlaybackMethod] = useUserPreferences("playback_electron_method", ElectronPlaybackMethod.NativePlayer)
    return {
        downloadedMediaPlayback,
        setDownloadedMediaPlayback,
        electronPlaybackMethod,
        setElectronPlaybackMethod,
    }
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

export function useExternalPlayerLink() {
    const [externalPlayerLink, setExternalPlayerLink] = useUserPreferences("playback_external_player_link", "")
    const [encodePath, setEncodePath] = useUserPreferences("playback_external_player_encode_path", false)
    return {
        externalPlayerLink,
        setExternalPlayerLink,
        encodePath,
        setEncodePath,
    }
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const __playback_playNextBaseAtom = atom<number | null>(null)

export function usePlayNext() {
    const [playNext, _setPlayNext] = useUserScopedAtom(__playback_playNextBaseAtom)

    function setPlayNext(ep: Nullish<number>, callback: () => void) {
        if (!ep) return
        _setPlayNext(ep)
        callback()
    }

    return {
        playNext,
        setPlayNext,
        resetPlayNext: () => _setPlayNext(null),
    }
}
