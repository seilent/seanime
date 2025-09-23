import { getServerBaseUrl } from "@/api/client/server-url"
import { Anime_Episode, Mediastream_StreamType, Nullish } from "@/api/generated/types"
import { useHandleContinuityWithMediaPlayer, useHandleCurrentMediaContinuity } from "@/api/hooks/continuity.hooks"
import { useContinuityWithCache } from "@/app/(main)/_features/progress-tracking/_lib/use-continuity-with-cache"
import { useContinuityWithCacheLoading } from "@/app/(main)/_features/progress-tracking/_lib/use-continuity-with-cache-loading"
import { useGetClientMediaSettings, useRequestMediastreamMediaContainer } from "@/api/hooks/mediastream.hooks"
import { useWebsocketMessageListener } from "@/app/(main)/_hooks/handle-websockets"
import { useMediastreamCurrentFile, useMediastreamJassubOffscreenRender } from "@/app/(main)/mediastream/_lib/mediastream.atoms"
import { clientIdAtom } from "@/app/websocket-provider"
import { logger } from "@/lib/helpers/debug"
import { legacy_getAssetUrl } from "@/lib/server/assets"
import { WSEvents } from "@/lib/server/ws-events"
import {
    LibASSTextRenderer,
    MediaCanPlayDetail,
    MediaPlayerInstance,
    MediaProviderAdapter,
    MediaProviderChangeEvent,
    MediaProviderSetupEvent,
} from "@vidstack/react"
import { useAtomValue } from "jotai"
import { useRouter } from "next/navigation"
import React from "react"
import { toast } from "sonner"

function uuidv4(): string {
    // @ts-ignore
    return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, (c) =>
        (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16),
    )
}

let cId = typeof window === "undefined" ? "-" : uuidv4()

type HandleMediastreamProps = {
    playerRef: React.RefObject<MediaPlayerInstance>
    episodes: Anime_Episode[]
    mediaId: Nullish<string | number>
}

export function useHandleMediastream(props: HandleMediastreamProps) {

    const {
        playerRef,
        episodes,
        mediaId,
    } = props
    const router = useRouter()
    const { filePath, setFilePath } = useMediastreamCurrentFile()

    const { data: mediastreamSettings, isFetching: mediastreamSettingsLoading } = useGetClientMediaSettings(true)

    /**
     * Stream URL
     */
    const prevUrlRef = React.useRef<string | undefined>(undefined)
    const definedUrlRef = React.useRef<string | undefined>(undefined)
    const [url, setUrl] = React.useState<string | undefined>(undefined)
    const [streamType] = React.useState<Mediastream_StreamType>("direct")

    // Refs
    const previousCurrentTimeRef = React.useRef(0)
    const previousIsPlayingRef = React.useRef(false)

    const sessionId = useAtomValue(clientIdAtom)

    /**
     * Current episode
     */
    const episode = React.useMemo(() => {
        return episodes.find(ep => !!ep.localFile?.path && ep.localFile?.path === filePath)
    }, [episodes, filePath])

    /**
     * Watch history with cache
     */
    const { waitForWatchHistory, getEpisodeContinuitySeekTo } = useContinuityWithCacheLoading({ mediaId, episodeNumber: episode?.episodeNumber })

    /**
     * Fetch media container containing stream URL
     */
    const { data: _mediaContainer, isError: isMediaContainerError, isPending, isFetching, refetch } = useRequestMediastreamMediaContainer({
        path: filePath,
        streamType: streamType,
        clientId: sessionId ?? uuidv4(),
    }, !!mediastreamSettings && !mediastreamSettingsLoading && !waitForWatchHistory)

    const mediaContainer = React.useMemo(() => (!isPending && !isFetching) ? _mediaContainer : undefined, [_mediaContainer, isPending, isFetching])

    // Whether the playback has errored
    const [playbackErrored, setPlaybackErrored] = React.useState<boolean>(false)

    // Duration
    const [duration, setDuration] = React.useState<number>(0)

    React.useEffect(() => {
        if (isPending) {
            logger("MEDIASTREAM").info("Loading media container")
            changeUrl(undefined)
            logger("MEDIASTREAM").info("Setting URL to undefined")
        }
    }, [isPending])

    /**
     * This error happens when the media container is available but the URL has been set to undefined
     */
    const isStreamError = !!mediaContainer && !url

    /**
     * Effect triggered when media container is available
     * - Set URL when media container is available
     */
    React.useEffect(() => {
        logger("MEDIASTREAM").info("Media container changed, running effect", mediaContainer)

        if (mediaContainer?.streamUrl) {
            logger("MEDIASTREAM").info("Stream URL available", mediaContainer.streamUrl)

            const _newUrl = `${getServerBaseUrl()}${mediaContainer.streamUrl}`

            logger("MEDIASTREAM").info("Changing URL", _newUrl, "streamType:", mediaContainer.streamType)

            changeUrl(_newUrl)
        } else {
            changeUrl(undefined)
            logger("MEDIASTREAM").info("Setting URL to undefined")
        }

    }, [mediaContainer?.streamUrl])

    //////////////////////////////////////////////////////////////
    // JASSUB
    //////////////////////////////////////////////////////////////

    const { jassubOffscreenRender } = useMediastreamJassubOffscreenRender()

    /**
     * Effect used to set LibASS renderer
     * Add subtitle renderer
     */
    React.useEffect(() => {
        if (playerRef.current && !!mediaContainer?.mediaInfo?.fonts?.length) {
            logger("MEDIASTREAM").info("Adding JASSUB renderer to player", mediaContainer?.mediaInfo?.fonts?.length, "fonts")
            const legacyWasmUrl = process.env.NODE_ENV === "development"
                ? "/jassub/jassub-worker.wasm.js" : legacy_getAssetUrl("/jassub/jassub-worker.wasm.js")

            logger("MEDIASTREAM").info("Loading JASSUB renderer")

            const fonts = mediaContainer?.mediaInfo?.fonts?.map(name => `${getServerBaseUrl()}/api/v1/mediastream/att/${name}`) || []

            // Extracted fonts
            let availableFonts: Record<string, string> = {}
            let firstFont = ""
            if (!!fonts?.length) {
                for (const font of fonts) {
                    const name = font.split("/").pop()?.split(".")[0]
                    if (name) {
                        if (!firstFont) {
                            firstFont = name.toLowerCase()
                        }
                        availableFonts[name.toLowerCase()] = font
                    }
                }
            }

            // Fallback font if no fonts are available
            if (!firstFont) {
                firstFont = "liberation sans"
            }
            if (Object.keys(availableFonts).length === 0) {
                availableFonts = {
                    "liberation sans": getServerBaseUrl() + `/jassub/default.woff2`,
                }
            }

            logger("MEDIASTREAM").info("Available fonts:", availableFonts)
            logger("MEDIASTREAM").info("Fallback font:", firstFont)

            // @ts-expect-error
            const renderer = new LibASSTextRenderer(() => import("jassub"), {
                wasmUrl: "/jassub/jassub-worker.wasm",
                workerUrl: "/jassub/jassub-worker.js",
                legacyWasmUrl: legacyWasmUrl,
                // Both parameters needed for subs to work on iOS, ref: jellyfin-vue
                offscreenRender: jassubOffscreenRender, // should be false for iOS
                prescaleFactor: 0.8,
                onDemandRender: false,
                fonts: fonts,
                availableFonts: availableFonts,
                fallbackFont: firstFont,
            })
            playerRef.current!.textRenderers.add(renderer)

            logger("MEDIASTREAM").info("JASSUB renderer added to player")

            return () => {
                playerRef.current!.textRenderers.remove(renderer)
            }
        }
    }, [
        playerRef.current,
        mediaContainer?.streamUrl,
        mediaContainer?.mediaInfo?.fonts,
        jassubOffscreenRender,
    ])

    /**
     * Changes the stream URL
     * @param newUrl
     */
    function changeUrl(newUrl: string | undefined) {
        logger("MEDIASTREAM").info("[changeUrl] called,", "request url:", newUrl)
        if (prevUrlRef.current !== newUrl) {
            logger("MEDIASTREAM").info("Resetting playback error status")
            setPlaybackErrored(false)
        }
        setUrl(prevUrl => {
            if (prevUrl === newUrl) {
                logger("MEDIASTREAM").info("[changeUrl] URL has not changed")
                return prevUrl
            }
            prevUrlRef.current = prevUrl
            logger("MEDIASTREAM").info("[changeUrl] URL updated")
            return newUrl
        })
        if (newUrl) {
            definedUrlRef.current = newUrl
        }
    }

    //////////////////////////////////////////////////////////////
    // Media player
    //////////////////////////////////////////////////////////////

    function onProviderChange(provider: MediaProviderAdapter | null, nativeEvent: MediaProviderChangeEvent) {
        logger("MEDIASTREAM").info("[onProviderChange] Provider changed to native")
    }

    function onProviderSetup(provider: MediaProviderAdapter, nativeEvent: MediaProviderSetupEvent) {
        logger("MEDIASTREAM").info("[onProviderSetup] Provider setup - native")
    }

    /**
     * Continuity with cache
     */
    const { handleUpdateWatchHistory, manualSync, isCacheEnabled } = useContinuityWithCache({ playerRef, episodeNumber: episode?.episodeNumber, mediaId })

    const preloadedNextFileForRef = React.useRef<string | undefined>(undefined) // unused

    const onCanPlay = (e: MediaCanPlayDetail) => {
        logger("MEDIASTREAM").info("[onCanPlay] called", e)
        preloadedNextFileForRef.current = undefined
        setDuration(e.duration)
    }

    const playNextEpisode = () => {
        logger("MEDIASTREAM").info("[playNextEpisode] called")
        const currentEpisodeIndex = episodes.findIndex(ep => !!ep.localFile?.path && ep.localFile?.path === filePath)
        if (currentEpisodeIndex !== -1) {
            const nextFile = episodes[currentEpisodeIndex + 1]
            if (nextFile?.localFile?.path) {
                onPlayFile(nextFile.localFile.path)
            }
        }
    }

    const onPlayFile = (filepath: string) => {
        logger("MEDIASTREAM").info("Playing file", filepath)
        playerRef.current?.destroy?.()
        previousCurrentTimeRef.current = 0
        setFilePath(filepath)
    }

    //////////////////////////////////////////////////////////////
    // Events
    //////////////////////////////////////////////////////////////

    /**
     * Listen for shutdown stream event
     * - This event is sent when something goes wrong internally
     * - Settings the URL to undefined will unmount the player and thus avoid spamming the server
     */
    useWebsocketMessageListener<string | null>({
        type: WSEvents.MEDIASTREAM_SHUTDOWN_STREAM,
        onMessage: log => {
            if (log) {
                toast.error(log)
            }
            logger("MEDIASTREAM").warning("Shutdown stream event received")
            changeUrl(undefined)
        },
    })

    //////////////////////////////////////////////////////////////

    // Subtitle endpoint URI
    const subtitleEndpointUri = React.useMemo(() => {
        if (mediaContainer?.streamUrl && mediaContainer?.streamType) {
            return `${getServerBaseUrl()}/api/v1/mediastream/subs`
        }
        return ""
    }, [mediaContainer?.streamUrl, mediaContainer?.streamType])

    return {
        url,
        streamType,
        subtitles: mediaContainer?.mediaInfo?.subtitles,
        isMediaContainerLoading: isPending,
        isError: isMediaContainerError || isStreamError,
        subtitleEndpointUri,
        mediaContainer: _mediaContainer,
        onPlayFile,
        filePath,
        episode,
        duration,
        setStreamType: (type: Mediastream_StreamType) => {
            logger("MEDIASTREAM").info("[setStreamType] Setting stream type", type)
            // Only allow direct stream type now
            if (type === "direct") {
                playerRef.current?.destroy?.()
                changeUrl(undefined)
            }
        },
        onCanPlay,
        playNextEpisode,
        onProviderChange,
        onProviderSetup,
        isCodecSupported: () => true, // Always return true since we only do direct streaming
        handleUpdateWatchHistory,
        manualSync,
        isCacheEnabled,
    }

}