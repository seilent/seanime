"use client"

import React, { createContext, useContext, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useServerStatus } from '@/app/(main)/_hooks/use-server-status'
import { logger } from '@/lib/helpers/debug'

// Types
interface User {
    id: number
    username: string
    displayName: string
    role: string
    isActive: boolean
    createdAt: string
    updatedAt: string
}

interface AuthContextType {
    user: User | null
    isAuthenticated: boolean
    isLoading: boolean
    isAdmin: boolean
    login: (username: string, password: string) => Promise<{ success: boolean; error?: string }>
    logout: () => Promise<void>
    checkAuth: () => Promise<void>
}

// Create context
const AuthContext = createContext<AuthContextType | undefined>(undefined)

// Auth provider component
export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = useState<User | null>(null)
    const [isLoading, setIsLoading] = useState(true)
    const router = useRouter()
    const serverStatus = useServerStatus()

    const isAuthenticated = !!user
    const isAdmin = user?.role === 'admin'

    // Check authentication status
    const checkAuth = async () => {
        try {
            const response = await fetch('/api/v1/users/profile', {
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json',
                },
            })

            if (response.ok) {
                const userData = await response.json() as { data: User }
                setUser(userData.data)
            } else {
                setUser(null)
            }
        } catch (error) {
            console.error('Auth check failed:', error)
            setUser(null)
        } finally {
            setIsLoading(false)
        }
    }

    // Login function
    const login = async (username: string, password: string): Promise<{ success: boolean; error?: string }> => {
        try {
            const response = await fetch('/api/v1/users/login', {
                method: 'POST',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password }),
            })

            const data = await response.json() as { data?: { user: User }; error?: string }

            if (response.ok && data.data) {
                setUser(data.data.user)
                return { success: true }
            } else {
                return { success: false, error: data.error || 'Login failed' }
            }
        } catch (error) {
            console.error('Login error:', error)
            return { success: false, error: 'Network error occurred' }
        }
    }

    // Logout function
    const logout = async () => {
        try {
            await fetch('/api/v1/users/logout', {
                method: 'POST',
                credentials: 'include',
            })
        } catch (error) {
            console.error('Logout error:', error)
        } finally {
            setUser(null)
            router.push('/login')
        }
    }

    // Check auth on mount
    useEffect(() => {
        logger("DBG").info("AuthProvider mounted, checking auth")
        checkAuth()
    }, [])

    // Auto-redirect logic
    useEffect(() => {
        if (!isLoading && serverStatus !== null) {
            const currentPath = window.location.pathname
            
            logger("DBG").info("Auth redirect logic", {
                currentPath,
                isAuthenticated,
                isLoading,
                hasSettings: !!serverStatus?.settings,
                serverStatusKeys: serverStatus ? Object.keys(serverStatus) : null,
                settingsObject: serverStatus?.settings
            })
            
            // If user is authenticated and on login page, redirect to home
            if (isAuthenticated && currentPath === '/login') {
                logger("DBG").info("Redirecting authenticated user from login to home")
                router.push('/')
                return
            }

            // If user is authenticated and on getting-started, redirect to home
            if (isAuthenticated && currentPath === '/getting-started') {
                logger("DBG").info("Redirecting authenticated user from getting-started to home")
                router.push('/')
                return
            }

            // If user is not authenticated and trying to access protected routes
            const protectedRoutes = [
                '/',
                '/settings',
                '/admin',
                '/anilist',
                '/auto-downloader',
                '/debrid',
                '/discover',
                '/extensions',
                '/manga',
                '/mediastream',
                '/offline',
                '/onlinestream',
                '/schedule',
                '/search',
                '/sync',
                '/torrent-list'
            ]

            const isProtectedRoute = protectedRoutes.some(route => 
                currentPath === route || currentPath.startsWith(route + '/')
            )

            if (!isAuthenticated && isProtectedRoute) {
                logger("DBG").info("Redirecting unauthenticated user to login from protected route", currentPath)
                router.push('/login')
                return
            }
        } else {
            logger("DBG").info("Auth redirect logic waiting", {
                isLoading,
                hasServerStatus: serverStatus !== null
            })
        }
    }, [isAuthenticated, isLoading, router, serverStatus?.settings])

    const value: AuthContextType = {
        user,
        isAuthenticated,
        isLoading,
        isAdmin,
        login,
        logout,
        checkAuth,
    }

    return (
        <AuthContext.Provider value={value}>
            {children}
        </AuthContext.Provider>
    )
}

// Custom hook to use auth context
export function useAuth() {
    const context = useContext(AuthContext)
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider')
    }
    return context
}
