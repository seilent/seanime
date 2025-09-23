import React, { useCallback, useEffect, useMemo, useState } from "react"
import { atom, Atom, WritableAtom, PrimitiveAtom, useAtom } from "jotai"
import { useAuth } from "@/contexts/auth-context"

// User values container - simple Map-based implementation
const userValues = new Map<string, Map<string, any>>()

/**
 * Get or create user-specific values map
 */
function getUserValues(userId: number): Map<string, any> {
    const userKey = `user_${userId}`
    if (!userValues.has(userKey)) {
        userValues.set(userKey, new Map())
    }
    return userValues.get(userKey)!
}

/**
 * Clear user-specific values (for logout)
 */
export function clearUserScope(userId: number): void {
    const userKey = `user_${userId}`
    userValues.delete(userKey)
}

/**
 * Create a user-scoped atom that automatically uses current user context
 */
export function createUserScopedAtom<T>(
    initialValue: T,
    options?: { key?: string }
): PrimitiveAtom<T> {
    return atom<T>(initialValue)
}

/**
 * Hook to use atom with user-specific scope
 */
export function useUserScopedAtom<T>(baseAtom: PrimitiveAtom<T>): [T, (value: T) => void] {
    const { user } = useAuth()

    if (!user) {
        throw new Error("useUserScopedAtom requires authenticated user")
    }

    // Use global user values map
    const userSpecificValues = getUserValues(user.id)
    const atomKey = baseAtom.toString()

    const [internalValue, setInternalValue] = useState<T>(() => {
        return userSpecificValues.get(atomKey) ?? (baseAtom as any).init
    })

    const value = useMemo(() => {
        return userSpecificValues.get(atomKey) ?? internalValue
    }, [userSpecificValues, atomKey, internalValue])

    const setValue = useCallback((newValue: T) => {
        userSpecificValues.set(atomKey, newValue)
        setInternalValue(newValue)
    }, [atomKey, userSpecificValues])

    return [value, setValue]
}

/**
 * Hook to use atom value with user-specific scope
 */
export function useUserScopedAtomValue<T>(baseAtom: Atom<T>): T {
    const [value] = useUserScopedAtom(baseAtom as PrimitiveAtom<T>)
    return value
}

/**
 * Hook to set atom value with user-specific scope
 */
export function useUserScopedSetAtom<T>(baseAtom: WritableAtom<T, [T], void>): (value: T) => void {
    const [, setValue] = useUserScopedAtom(baseAtom as PrimitiveAtom<T>)
    return setValue
}

/**
 * Create a user-scoped derived atom
 */
export function createUserScopedDerivedAtom<T>(
    deps: readonly Atom<unknown>[],
    fn: (get: (atom: Atom<unknown>) => unknown) => T
): Atom<T> {
    return atom<T>(get => fn(get))
}

/**
 * Hook to manage user-specific preferences with server storage
 */
export function useUserPreferences<T>(key: string, defaultValue: T) {
    const { user, isLoading } = useAuth()

    const [value, setValue] = useState<T>(defaultValue)

    // Load preferences from server on mount or when user changes
    useEffect(() => {
        if (!user) return

        const loadPreferences = async () => {
            try {
                const response = await fetch(`/api/v1/users/preferences/${key}`, {
                    credentials: 'include',
                })
                if (response.ok) {
                    const data = await response.json() as { value: T }
                    setValue(data.value || defaultValue)
                }
            } catch (error) {
                console.error(`Failed to load user preference ${key}:`, error)
            }
        }

        loadPreferences()
    }, [key, defaultValue, user])

    // Save preferences to server when they change (only if user is authenticated)
    const setValueAndSave = useCallback(async (newValue: T) => {
        setValue(newValue)

        if (!user) return

        try {
            await fetch(`/api/v1/users/preferences/${key}`, {
                method: 'PUT',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ value: newValue }),
            })
        } catch (error) {
            console.error(`Failed to save user preference ${key}:`, error)
        }
    }, [key, user, setValue])

    return [value, setValueAndSave] as const
}