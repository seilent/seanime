import { useScanLocalFiles } from "@/api/hooks/scan.hooks"
import { useGetAnimeCollection } from "@/api/hooks/anilist.hooks"
import { atom } from "jotai"
import { useSetAtom } from "jotai/react"
import { useState } from "react"

export const __scanner_isScanningAtom = atom(false)

/**
 * Smart library scan that waits for AniList data before scanning local files
 * This ensures AniList is always the source of truth
 */
export function useSmartLibraryScan() {
    const [isWaitingForAnilist, setIsWaitingForAnilist] = useState(false)
    const setScannerIsScanning = useSetAtom(__scanner_isScanningAtom)
    const { data: anilistCollection, isLoading: anilistLoading, refetch: refetchAnilist } = useGetAnimeCollection()
    const { mutate: scanLocalFiles, isPending: scanPending } = useScanLocalFiles(() => {
        setIsWaitingForAnilist(false)
        setScannerIsScanning(false)
    })

    const scanLibrary = async () => {
        setIsWaitingForAnilist(true)
        setScannerIsScanning(true)

        // If AniList data is not loaded yet, wait for it
        if (!anilistCollection && !anilistLoading) {
            // Trigger AniList refetch if data is not loading and not available
            await refetchAnilist()
        }

        // Wait for AniList data to be available
        const checkAnilistData = () => {
            if (anilistCollection || !anilistLoading) {
                // AniList data is available (or finished loading), proceed with local scan
                scanLocalFiles({
                    skipIgnoredFiles: true,
                })
                setIsWaitingForAnilist(false)
            } else {
                // Still loading, check again in a short interval
                setTimeout(checkAnilistData, 500)
            }
        }

        checkAnilistData()
    }

    const refreshLibrary = () => {
        // For refresh, skip AniList data waiting - just scan local files immediately
        setScannerIsScanning(true)
        scanLocalFiles({
            skipIgnoredFiles: true,
        })
    }

    return {
        scanLibrary, // For initial/empty library - waits for AniList data
        refreshLibrary, // For refresh - uses existing AniList data
        isPending: isWaitingForAnilist || anilistLoading || scanPending,
        isWaitingForAnilist,
    }
}