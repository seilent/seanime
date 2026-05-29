import { API_ENDPOINTS } from "@/api/generated/endpoints"
import {
    usePlaybackSyncCurrentProgress,
} from "@/api/hooks/playback_manager.hooks"
import { AutoplayCountdownModal } from "@/app/(main)/_features/progress-tracking/_components/autoplay-countdown-modal"
import { useAutoplay, useNextEpisodeResolver } from "@/app/(main)/_features/progress-tracking/_lib/autoplay"
import { PlaybackManager_PlaybackState, PlaybackManager_PlaylistState } from "@/app/(main)/_features/progress-tracking/_lib/playback-manager.types"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { Button, IconButton } from "@/components/ui/button"
import { cn } from "@/components/ui/core/styling"
import { Modal } from "@/components/ui/modal"
import { ProgressBar } from "@/components/ui/progress-bar"
import { logger } from "@/lib/helpers/debug"
import { WSEvents } from "@/lib/server/ws-events"
import { useQueryClient } from "@tanstack/react-query"
import { atom, useAtomValue } from "jotai"
import { useAtom } from "jotai/react"
import mousetrap from "mousetrap"
import Image from "next/image"
import React from "react"
import { PiPopcornFill } from "react-icons/pi"
import { toast } from "sonner"
import { useSSEEvents } from "@/hooks/use-sse-events"

const __pt_showModalAtom = atom(false)
const __pt_isTrackingAtom = atom(false)
const __pt_isCompletedAtom = atom(false)

type Props = {
    asSidebarButton?: boolean
}

export function PlaybackManagerProgressTrackingButton({ asSidebarButton }: Props) {
    const [showModal, setShowModal] = useAtom(__pt_showModalAtom)

    const isTracking = useAtomValue(__pt_isTrackingAtom)

    const isCompleted = useAtomValue(__pt_isCompletedAtom)

    // \/ Modal can be displayed when progress tracking or video is completed
    // Basically, keep the modal visible if there's no more tracking but the video is completed
    const shouldBeDisplayed = isTracking || isCompleted

    return (
        <>
            {shouldBeDisplayed && (
                <>
                    {asSidebarButton ? (
                        <IconButton
                            data-progress-tracking-button
                            intent="primary-subtle"
                            className={cn("animate-pulse")}
                            icon={<PiPopcornFill />}
                            onClick={() => setShowModal(true)}
                        />
                    ) : (
                        <Button
                            data-progress-tracking-button
                            intent="primary"
                            className={cn("animate-pulse")}
                            leftIcon={<PiPopcornFill />}
                            onClick={() => setShowModal(true)}
                        >
                            Currently watching
                        </Button>)}
                </>
            )}
        </>
    )
}

export function PlaybackManagerProgressTracking() {
    const serverStatus = useServerStatus()
    const qc = useQueryClient()

    const [showModal, setShowModal] = useAtom(__pt_showModalAtom)

    /**
     * Progress tracking states
     * - 'True' when tracking has started
     * - 'False' when tracking has stopped
     */
    const [isTracking, setIsTracking] = useAtom(__pt_isTrackingAtom)
    /**
     * Video completion state
     * - 'True' when the video has been completed
     * - 'False' by default
     */
    const [isCompleted, setIsCompleted] = useAtom(__pt_isCompletedAtom)

    // \/ Modal can be displayed when progress tracking or video is completed
    // Basically, keep the modal visible if there's no more tracking but the video is completed
    const shouldBeDisplayed = isTracking || isCompleted

    const [state, setState] = React.useState<PlaybackManager_PlaybackState | null>(null)
    const [playlistState, setPlaylistState] = React.useState<PlaybackManager_PlaylistState | null>(null)

    const { state: autoplayState, startAutoplay, cancelAutoplay } = useAutoplay()

    // Get next episode for local playback
    const nextEpisodeToPlay = useNextEpisodeResolver(
        state?.mediaId || 0,
        state?.episodeNumber || 0,
    )

    const { mutate: syncProgress, isPending } = usePlaybackSyncCurrentProgress()

    // Convert to SSE-based event handling for Playback Manager
    const handlePmEvent = React.useCallback((type: string, payload: any) => {
        switch (type) {
            case WSEvents.PLAYBACK_MANAGER_PROGRESS_TRACKING_STARTED: {
                const data = payload as PlaybackManager_PlaybackState | null
                logger("PlaybackManagerProgressTracking").info("Tracking started (SSE)", data)
                setIsTracking(true)
                setIsCompleted(false)
                setShowModal(true)
                setState(data)
                break
            }
            case WSEvents.PLAYBACK_MANAGER_PROGRESS_PLAYBACK_STATE: {
                const data = payload as PlaybackManager_PlaybackState | null
                if (!isTracking) setIsTracking(true)
                setState(data)
                break
            }
            case WSEvents.PLAYBACK_MANAGER_PROGRESS_VIDEO_COMPLETED: {
                const data = payload as PlaybackManager_PlaybackState | null
                logger("PlaybackManagerProgressTracking").info("Video completed (SSE)", data)
                setIsCompleted(true)
                setState(data)
                break
            }
            case WSEvents.PLAYBACK_MANAGER_PROGRESS_TRACKING_STOPPED: {
                const reason = payload as string
                logger("PlaybackManagerProgressTracking").info("Tracking stopped (SSE)", reason, "Completion:", state?.completionPercentage)
                setIsTracking(false)
                if (state?.progressUpdated) {
                    logger("PlaybackManagerProgressTracking").info("Progress updated, setting isCompleted to false")
                    setIsCompleted(false)
                }
                if (reason === "Player closed") {
                    toast.info("Player closed")
                    if ((state?.completionPercentage || 0) <= 0.8) setIsCompleted(false)
                } else if (reason === "Tracking stopped") {
                    toast.info("Tracking stopped")
                } else if (reason) {
                    toast.error(reason)
                }
                qc.invalidateQueries({ queryKey: [API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistory.key] }).then()
                if (state && state.completionPercentage && state.completionPercentage > 0.7) {
                    if (!autoplayState.isActive) startAutoplay(state, nextEpisodeToPlay || undefined, "local")
                }
                setState(null)
                break
            }
            case WSEvents.PLAYBACK_MANAGER_PROGRESS_UPDATED: {
                const data = payload as PlaybackManager_PlaybackState | null
                if (data) {
                    qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key, String(data.mediaId)] })
                    qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.key] })
                    qc.invalidateQueries({ queryKey: [API_ENDPOINTS.ANILIST.GetAnimeCollection.key] })
                    setState(data)
                    toast.success("Progress updated")
                }
                break
            }
            case WSEvents.PLAYBACK_MANAGER_PLAYLIST_STATE: {
                setPlaylistState(payload)
                break
            }
        }
    }, [isTracking, state, autoplayState.isActive, startAutoplay, nextEpisodeToPlay, qc])

    useSSEEvents({
        enabled: true,
        onEvent: (evt) => {
            switch (evt.type) {
                case WSEvents.PLAYBACK_MANAGER_PROGRESS_TRACKING_STARTED:
                case WSEvents.PLAYBACK_MANAGER_PROGRESS_PLAYBACK_STATE:
                case WSEvents.PLAYBACK_MANAGER_PROGRESS_VIDEO_COMPLETED:
                case WSEvents.PLAYBACK_MANAGER_PROGRESS_TRACKING_STOPPED:
                case WSEvents.PLAYBACK_MANAGER_PROGRESS_UPDATED:
                case WSEvents.PLAYBACK_MANAGER_PLAYLIST_STATE:
                    handlePmEvent(evt.type, evt.payload)
                    break
            }
        },
    })


    // Progress update keyboard shortcuts
    React.useEffect(() => {
        mousetrap.bind("u", () => {
            if (!isPending && state?.completionPercentage && state?.completionPercentage > 0.7) {
                syncProgress()
            }
        })

        return () => {
            mousetrap.unbind("u")
        }
    }, [state?.completionPercentage && state?.completionPercentage > 0.7, cancelAutoplay])


    function handleUpdateProgress() {
        syncProgress()
    }

    // React.useEffect(() => {
    //     mousetrap.bind("esc", () => {
    //         cancelAutoPlay()
    //         setShowModal(false)
    //         setShowAutoPlayCountdownModal(false)
    //         setIsTracking(false)
    //         setIsCompleted(false)
    //         setState(null)
    //         setPlaylistState(null)
    //         setWillAutoPlay(false)
    //         resetTorrentstreamAutoplayInfo()
    //         resetDebridstreamAutoplayInfo()
    //         clearTimers()
    //     })

    //     return () => {
    //         mousetrap.unbind("esc")
    //     }
    // }, [])

    return (
        <>
            <Modal
                data-progress-tracking-modal
                open={showModal && shouldBeDisplayed}
                onOpenChange={v => setShowModal(v)}
                titleClass="text-center"
                contentClass="!space-y-0 relative max-w-2xl overflow-hidden"
            >
                {!!state?.completionPercentage && <div data-progress-tracking-modal-progress-bar className="absolute left-0 top-0 w-full">
                    <ProgressBar className="h-2 rounded-lg" value={state.completionPercentage * 100} />
                </div>}
                {state && <div data-progress-tracking-main-content className="text-center relative overflow-hidden py-2 space-y-2">
                    {state.mediaCoverImage && <div className="size-16 rounded-full relative mx-auto overflow-hidden mb-3">
                        <Image src={state.mediaCoverImage} alt="cover image" fill className="object-cover object-center" />
                    </div>}
                    {/*<p className="text-[--muted]">Currently watching</p>*/}
                    <div data-progress-tracking-title>
                        <h3 className="text-lg font-medium line-clamp-1">{state?.mediaTitle}</h3>
                        <p className="text-2xl font-bold">Episode {state?.episodeNumber}
                            <span className="text-[--muted]">{" / "}{state?.mediaTotalEpisodes || "-"}</span>
                        </p>
                    </div>
                    {(serverStatus?.settings?.autoUpdateProgress && !state?.progressUpdated) && (
                        <p data-progress-tracking-auto-update-progress className="text-[--muted] text-center text-sm">
                            Your progress will be automatically updated
                        </p>
                    )}
                    {(state?.progressUpdated) && (
                        <p data-progress-tracking-progress-updated className="text-green-300 text-center">
                            Progress updated
                        </p>
                    )}
                </div>}

                {(
                    !!state?.completionPercentage
                    && state?.completionPercentage > 0.7
                    && !state.progressUpdated
                ) && <div data-progress-tracking-update-progress-button className="flex gap-2 justify-center items-center">
                    <Button
                        intent="primary-subtle"
                        disabled={isPending || state?.progressUpdated}
                        onClick={handleUpdateProgress}
                        className="w-full animate-pulse"
                        loading={isPending}
                    >
                        Update progress now
                    </Button>
                </div>}

            </Modal>

            <AutoplayCountdownModal
                autoplayState={autoplayState}
                onCancel={cancelAutoplay}
            />
        </>
    )

}
