'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { TextInput } from '@/components/ui/text-input'
import { Card } from '@/components/ui/card'
import { Alert } from '@/components/ui/alert'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { Modal } from '@/components/ui/modal'
import { Checkbox } from '@/components/ui/checkbox'
import { PageWrapper } from '@/components/shared/page-wrapper'
import { useSaveSettings } from '@/api/hooks/settings.hooks'

interface User {
    id: number
    username: string
    role: string
    isActive: boolean
    displayName: string
    createdAt: string
    updatedAt: string
}

interface AniListWhitelistUser {
    username: string
    isAdmin: boolean
}

export default function UsersManagementPage() {
    const [users, setUsers] = useState<User[]>([])
    const [whitelistUsers, setWhitelistUsers] = useState<AniListWhitelistUser[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [error, setError] = useState('')
    const [isAddModalOpen, setIsAddModalOpen] = useState(false)
    const [isAdding, setIsAdding] = useState(false)
    const [newUsername, setNewUsername] = useState('')
    const { mutate: saveSettings, isPending: isSavingSettings } = useSaveSettings()

    // Fetch users
    const fetchUsers = async () => {
        try {
            const response = await fetch('/api/v1/admin/users', {
                credentials: 'include'
            })

            if (response.ok) {
                const data = await response.json() as { data?: User[] }
                setUsers(data.data || [])
            } else {
                setError('Failed to fetch users')
            }
        } catch (err) {
            setError('Network error')
        } finally {
            setIsLoading(false)
        }
    }

    // Fetch whitelist from API
    const fetchWhitelist = async () => {
        try {
            const response = await fetch('/api/v1/admin/whitelist', {
                credentials: 'include'
            })

            if (response.ok) {
                const data = await response.json() as { data?: string[] }
                const whitelist = (data.data || []).map((username, index) => ({
                    username,
                    isAdmin: index === 0 // First user in whitelist is admin
                }))
                setWhitelistUsers(whitelist)
            }
        } catch (err) {
            console.error('Failed to fetch whitelist:', err)
        }
    }

    useEffect(() => {
        fetchWhitelist()
    }, [])

    // Add user to whitelist
    const handleAddToWhitelist = async (e: React.FormEvent) => {
        e.preventDefault()
        setIsAdding(true)
        setError('')

        if (!newUsername.trim()) {
            setError('AniList username is required')
            setIsAdding(false)
            return
        }

        // Check if user already in whitelist
        if (whitelistUsers.some(u => u.username === newUsername.trim())) {
            setError('User already in whitelist')
            setIsAdding(false)
            return
        }

        try {
            const response = await fetch('/api/v1/admin/whitelist', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include',
                body: JSON.stringify({ username: newUsername.trim() })
            })

            if (response.ok) {
                setNewUsername('')
                setIsAddModalOpen(false)
                await fetchWhitelist() // Refresh whitelist
            } else {
                const errorData = await response.json() as { error?: string }
                setError(errorData.error || 'Failed to add user to whitelist')
            }
        } catch (err) {
            setError('Network error')
        } finally {
            setIsAdding(false)
        }
    }

    // Remove user from whitelist
    const handleRemoveFromWhitelist = async (username: string, isAdmin: boolean) => {
        if (isAdmin && !confirm('Removing the admin user will revoke their admin access. Are you sure?')) {
            return
        }

        if (!confirm(`Are you sure you want to remove ${username} from the whitelist?`)) {
            return
        }

        try {
            const response = await fetch(`/api/v1/admin/whitelist/${encodeURIComponent(username)}`, {
                method: 'DELETE',
                credentials: 'include'
            })

            if (response.ok) {
                await fetchWhitelist() // Refresh whitelist
            } else {
                const errorData = await response.json() as { error?: string }
                setError(errorData.error || 'Failed to remove user from whitelist')
            }
        } catch (err) {
            setError('Network error')
        }
    }

    // Delete user account
    const handleDeleteUser = async (userId: number) => {
        if (!confirm('Are you sure you want to delete this user account? This will remove all their data.')) {
            return
        }

        try {
            const response = await fetch(`/api/v1/admin/users/${userId}`, {
                method: 'DELETE',
                credentials: 'include'
            })

            if (response.ok) {
                fetchUsers() // Refresh the list
            } else {
                setError('Failed to delete user')
            }
        } catch (err) {
            setError('Network error')
        }
    }

    useEffect(() => {
        fetchUsers()
    }, [])

    if (isLoading) {
        return (
            <PageWrapper className="flex items-center justify-center">
                <LoadingSpinner className="w-8 h-8" />
            </PageWrapper>
        )
    }

    return (
        <PageWrapper>
            <div className="space-y-6">
                <div className="flex justify-between items-center">
                    <div>
                        <h1 className="text-2xl font-bold">User Management</h1>
                        <p className="text-gray-600 dark:text-gray-400">
                            Manage AniList users and permissions. Users must be in the whitelist to access the server.
                        </p>
                    </div>
                    <Button onClick={() => setIsAddModalOpen(true)}>
                        Add to Whitelist
                    </Button>
                </div>

                {error && (
                    <Alert intent="alert">
                        {error}
                    </Alert>
                )}

                <Card className="p-6">
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">AniList Whitelist</h2>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                            Only AniList users in this whitelist can access the server. The first user is automatically an admin.
                        </p>

                        {whitelistUsers.length === 0 ? (
                            <p className="text-gray-500">No users in whitelist</p>
                        ) : (
                            <div className="space-y-3">
                                {whitelistUsers.map((user, index) => (
                                    <div
                                        key={user.username}
                                        className="flex items-center justify-between p-4 border rounded-lg"
                                    >
                                        <div className="space-y-1">
                                            <div className="flex items-center gap-2">
                                                <span className="font-medium">{user.username}</span>
                                                {user.isAdmin && (
                                                    <span className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded">
                                                        Admin
                                                    </span>
                                                )}
                                                {index === 0 && (
                                                    <span className="px-2 py-1 text-xs bg-green-100 text-green-800 rounded">
                                                        First User
                                                    </span>
                                                )}
                                            </div>
                                            <p className="text-sm text-gray-500">
                                                AniList Username
                                            </p>
                                        </div>

                                        <Button
                                            intent="alert-subtle"
                                            size="sm"
                                            onClick={() => handleRemoveFromWhitelist(user.username, user.isAdmin)}
                                            disabled={isSavingSettings}
                                        >
                                            Remove
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </Card>

                {users.length > 0 && (
                    <Card className="p-6">
                        <div className="space-y-4">
                            <h2 className="text-lg font-semibold">Registered Users</h2>
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                                Users who have logged in at least once. You can delete their accounts and data here.
                            </p>

                            <div className="space-y-3">
                                {users.map((user) => (
                                    <div
                                        key={user.id}
                                        className="flex items-center justify-between p-4 border rounded-lg"
                                    >
                                        <div className="space-y-1">
                                            <div className="flex items-center gap-2">
                                                <span className="font-medium">{user.displayName || user.username}</span>
                                                <span className="text-sm text-gray-500">@{user.username}</span>
                                                {user.role === 'admin' && (
                                                    <span className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded">
                                                        Admin
                                                    </span>
                                                )}
                                                {!user.isActive && (
                                                    <span className="px-2 py-1 text-xs bg-red-100 text-red-800 rounded">
                                                        Inactive
                                                    </span>
                                                )}
                                            </div>
                                            <p className="text-sm text-gray-500">
                                                Created: {new Date(user.createdAt).toLocaleDateString()}
                                            </p>
                                        </div>

                                        <Button
                                            intent="alert-subtle"
                                            size="sm"
                                            onClick={() => handleDeleteUser(user.id)}
                                        >
                                            Delete Account
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </Card>
                )}
            </div>

            {/* Add to Whitelist Modal */}
            <Modal
                open={isAddModalOpen}
                onOpenChange={setIsAddModalOpen}
                title="Add User to Whitelist"
            >
                <form onSubmit={handleAddToWhitelist} className="space-y-4">
                    <div className="space-y-2">
                        <TextInput
                            label="AniList Username"
                            placeholder="Enter AniList username"
                            value={newUsername}
                            onValueChange={setNewUsername}
                            required
                            disabled={isAdding}
                        />
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                            The user must have this exact username on AniList to be able to log in.
                        </p>
                    </div>

                    <div className="flex gap-2 pt-4">
                        <Button
                            type="button"
                            intent="gray-subtle"
                            onClick={() => setIsAddModalOpen(false)}
                            disabled={isAdding}
                        >
                            Cancel
                        </Button>
                        <Button
                            type="submit"
                            disabled={isAdding}
                            leftIcon={isAdding ? <LoadingSpinner className="w-4 h-4" /> : undefined}
                        >
                            {isAdding ? 'Adding...' : 'Add to Whitelist'}
                        </Button>
                    </div>
                </form>
            </Modal>
        </PageWrapper>
    )
}
