import { serverStatusAtom } from "@/app/(main)/_atoms/server-status.atoms"
import { TORRENT_PROVIDER } from "@/lib/server/settings"
import { useAtomValue } from "jotai"
import { useSetAtom } from "jotai/react"
import React from "react"
import { useAuth } from "@/contexts/auth-context"

export function useServerStatus() {
    return useAtomValue(serverStatusAtom)
}

export function useSetServerStatus() {
    return useSetAtom(serverStatusAtom)
}

export function useCurrentUser() {
    const { user } = useAuth()
    return React.useMemo(() => user, [user])
}

export function useHasTorrentProvider() {
    const serverStatus = useServerStatus()
    return {
        hasTorrentProvider: React.useMemo(() => !!serverStatus?.settings?.library?.torrentProvider && serverStatus?.settings?.library?.torrentProvider !== TORRENT_PROVIDER.NONE,
            [serverStatus?.settings?.library?.torrentProvider]),
    }
}

// Debrid feature removed: useHasDebridService deleted

