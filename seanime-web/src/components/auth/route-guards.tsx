"use client"

import React from 'react'
import { useAuth } from '@/contexts/auth-context'
import { LoadingOverlayWithLogo } from '@/components/shared/loading-overlay-with-logo'
import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

// Protected route component - requires authentication
export function ProtectedRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth()
    const router = useRouter()

    useEffect(() => {
        if (!isLoading && !isAuthenticated) {
            router.push('/login')
        }
    }, [isAuthenticated, isLoading, router])

    if (isLoading) {
        return <LoadingOverlayWithLogo />
    }

    if (!isAuthenticated) {
        return null // Will redirect to login
    }

    return <>{children}</>
}

// Admin route component - requires admin role
export function AdminRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isAdmin, isLoading } = useAuth()
    const router = useRouter()

    useEffect(() => {
        if (!isLoading) {
            if (!isAuthenticated) {
                router.push('/login')
            } else if (!isAdmin) {
                router.push('/') // Redirect non-admin users to home
            }
        }
    }, [isAuthenticated, isAdmin, isLoading, router])

    if (isLoading) {
        return <LoadingOverlayWithLogo />
    }

    if (!isAuthenticated || !isAdmin) {
        return null // Will redirect
    }

    return <>{children}</>
}

// Public route component - redirects authenticated users
export function PublicRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth()
    const router = useRouter()

    useEffect(() => {
        if (!isLoading && isAuthenticated) {
            router.push('/')
        }
    }, [isAuthenticated, isLoading, router])

    if (isLoading) {
        return <LoadingOverlayWithLogo />
    }

    if (isAuthenticated) {
        return null // Will redirect to home
    }

    return <>{children}</>
}

// Setup route component - only accessible when no users exist
export function SetupRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth()
    const router = useRouter()

    useEffect(() => {
        if (!isLoading && isAuthenticated) {
            router.push('/')
        }
    }, [isAuthenticated, isLoading, router])

    if (isLoading) {
        return <LoadingOverlayWithLogo />
    }

    if (isAuthenticated) {
        return null // Will redirect to home
    }

    return <>{children}</>
}
