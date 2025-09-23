import { atom } from "jotai"
import { createContext } from "react"
import { useUserScopedAtom } from "./user-scoped-atoms"

export const WebSocketContext = createContext<WebSocket | null>(null)

// Base atom for WebSocket connection
const websocketBaseAtom = atom<WebSocket | null>(null)

// Hook to use user-scoped WebSocket atom
export function useWebsocket() {
    return useUserScopedAtom(websocketBaseAtom)
}

// Keep the original atom for backward compatibility
export const websocketAtom = websocketBaseAtom

