import { useAtom } from "jotai/react"
import { atomWithStorage } from "jotai/utils"

const __mediastream_filePath = atomWithStorage<string | undefined>("sea-mediastream-filepath", undefined, undefined, { getOnInit: true })

export function useMediastreamCurrentFile() {
    const [filePath, setFilePath] = useAtom(__mediastream_filePath)

    return {
        filePath,
        setFilePath,
    }
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const __mediastream_jassubOffscreenRender = atomWithStorage<boolean>("sea-mediastream-jassub-offscreen-render", false, undefined, { getOnInit: true })

export function useMediastreamJassubOffscreenRender() {
    const [jassubOffscreenRender, setJassubOffscreenRender] = useAtom(__mediastream_jassubOffscreenRender)

    return {
        jassubOffscreenRender,
        setJassubOffscreenRender,
    }
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * Always use media streaming for all devices
 */
export function useMediastreamActiveOnDevice() {
    return {
        activeOnDevice: true,
        setActiveOnDevice: () => {}, // No-op since it's always enabled
    }
}