import { atomWithStorage } from "jotai/utils"

export const __seaMediaPlayer_autoPlayAtom = atomWithStorage("sea-media-player-autoplay", false, undefined, { getOnInit: true })

export const __seaMediaPlayer_autoNextAtom = atomWithStorage("sea-media-player-autonext", false, undefined, { getOnInit: true })

export const __seaMediaPlayer_autoSkipIntroOutroAtom = atomWithStorage("sea-media-player-autoskip-intro-outro", false, undefined, { getOnInit: true })

export const __seaMediaPlayer_discreteControlsAtom = atomWithStorage("sea-media-player-discrete-controls", false, undefined, { getOnInit: true })

export const __seaMediaPlayer_volumeAtom = atomWithStorage("sea-media-player-volume", 1, undefined, { getOnInit: true })

export const __seaMediaPlayer_mutedAtom = atomWithStorage("sea-media-player-muted", false, undefined, { getOnInit: true })

export const __seaMediaPlayer_playbackRateAtom = atomWithStorage("sea-media-playback-rate", 1, undefined, { getOnInit: true })

export const __seaMediaPlayer_audioBoostAtom = atomWithStorage("sea-media-player-audio-boost", 1, undefined, { getOnInit: true })

export const seaMediaPlayer_audioBoostOptions: { label: string; value: number }[] = [
    { label: "Off", value: 1 },
    { label: "+3 dB", value: 1.41 },
    { label: "+6 dB", value: 2 },
    { label: "+9 dB", value: 2.83 },
    { label: "+12 dB", value: 4 },
]
