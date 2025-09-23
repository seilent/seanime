import React, { createContext, useContext, useEffect } from 'react'
import { useAuth } from './auth-context'
import { clearUserScope } from '@/app/(main)/_atoms/user-scoped-atoms'

interface UserJotaiContextType {
    isReady: boolean
}

const UserJotaiContext = createContext<UserJotaiContextType>({
    isReady: false,
})

export function UserJotaiProvider({ children }: { children: React.ReactNode }) {
    const { user, isAuthenticated, isLoading } = useAuth()
    const [isReady, setIsReady] = React.useState(false)

    // Clear user scope when user logs out
    useEffect(() => {
        if (!isAuthenticated && !isLoading && !user) {
            // Clear any existing user scopes
            // Note: We can't clear all scopes without knowing the user IDs
            // This will be handled individually on logout
            setIsReady(false)
        } else if (isAuthenticated && user) {
            setIsReady(true)
        }
    }, [user, isAuthenticated, isLoading])

    // Initialize user scope when user is authenticated
    useEffect(() => {
        if (isAuthenticated && user && isReady) {
            // User scope will be created when first needed
            // This is just to ensure the provider is ready
        }
    }, [isAuthenticated, user, isReady])

    const value: UserJotaiContextType = {
        isReady: isReady && !isLoading,
    }

    return (
        <UserJotaiContext.Provider value={value}>
            {children}
        </UserJotaiContext.Provider>
    )
}

export function useUserJotai() {
    const context = useContext(UserJotaiContext)
    if (context === undefined) {
        throw new Error('useUserJotai must be used within a UserJotaiProvider')
    }
    return context
}

// Hook to ensure user context is available before using scoped atoms
export function useRequireUserContext() {
    const { user } = useAuth()
    const { isReady } = useUserJotai()

    if (!isReady || !user) {
        throw new Error('User context not ready. This hook requires an authenticated user.')
    }

    return { user, isReady }
}