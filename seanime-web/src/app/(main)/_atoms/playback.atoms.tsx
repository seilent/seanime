import { Nullish } from "@/api/generated/types"
import { atom } from "jotai"
import { useUserScopedAtom } from "./user-scoped-atoms"

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