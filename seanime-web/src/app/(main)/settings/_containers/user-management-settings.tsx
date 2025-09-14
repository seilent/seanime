'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { TextInput } from '@/components/ui/text-input'
import { Alert } from '@/components/ui/alert'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { Modal } from '@/components/ui/modal'
import { useServerStatus } from '@/app/(main)/_hooks/use-server-status'
import { SettingsCard } from '@/app/(main)/settings/_components/settings-card'
import { toast } from 'sonner'

interface AniListWhitelistUser {
    username: string
}

export function UserManagementSettings() {
    const [whitelistUsers, setWhitelistUsers] = useState<AniListWhitelistUser[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [error, setError] = useState('')
    const [isAddModalOpen, setIsAddModalOpen] = useState(false)
    const [isAdding, setIsAdding] = useState(false)
    const [newUsername, setNewUsername] = useState('')
    const serverStatus = useServerStatus()

    // Fetch whitelist
    const fetchWhitelist = async () => {
        try {
            const response = await fetch('/api/v1/admin/whitelist', {
                credentials: 'include'
            })

            if (response.ok) {
                const data = await response.json() as { data?: string[] }
                const whitelist = (data.data || []).map((username) => ({
                    username
                }))
                setWhitelistUsers(whitelist)
            } else {
                setError('Failed to fetch whitelist')
            }
        } catch (err) {
            setError('Network error fetching whitelist')
        } finally {
            setIsLoading(false)
        }
    }

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
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username: newUsername.trim() }),
                credentials: 'include'
            })

            if (response.ok) {
                toast.success('User added to whitelist')
                setNewUsername('')
                setIsAddModalOpen(false)
                fetchWhitelist()
            } else {
                const errorData = await response.json() as { error?: string }
                setError(errorData.error || 'Failed to add user to whitelist')
            }
        } catch (err) {
            setError('Failed to add user to whitelist')
        } finally {
            setIsAdding(false)
        }
    }

    // Remove user from whitelist
    const handleRemoveFromWhitelist = async (username: string) => {
        if (!confirm(`Are you sure you want to remove ${username} from the whitelist?`)) {
            return
        }

        try {
            const response = await fetch(`/api/v1/admin/whitelist/${encodeURIComponent(username)}`, {
                method: 'DELETE',
                credentials: 'include'
            })

            if (response.ok) {
                toast.success('User removed from whitelist')
                fetchWhitelist()
            } else {
                const errorData = await response.json() as { error?: string }
                setError(errorData.error || 'Failed to remove user from whitelist')
            }
        } catch (err) {
            setError('Failed to remove user from whitelist')
        }
    }

    useEffect(() => {
        fetchWhitelist()
    }, [])

    if (isLoading) {
        return (
            <SettingsCard>
                <div className="flex items-center justify-center py-8">
                    <LoadingSpinner className="w-8 h-8" />
                </div>
            </SettingsCard>
        )
    }

    return (
        <>
            <div className="space-y-4">
                <div className="flex justify-between items-center">
                    <Button onClick={() => setIsAddModalOpen(true)}>
                        Add to Whitelist
                    </Button>
                </div>

                {error && (
                    <Alert intent="alert">
                        {error}
                    </Alert>
                )}

                <SettingsCard title="Whitelisted Users">
                    <div className="space-y-4">
                        <p className="text-sm text-[--muted]">
                            Only AniList users in this whitelist can access the server.
                        </p>

                        {whitelistUsers.length === 0 ? (
                            <p className="text-[--muted]">No users in whitelist</p>
                        ) : (
                            <div className="space-y-3">
                                {whitelistUsers.map((user) => (
                                    <div
                                        key={user.username}
                                        className="flex items-center justify-between p-4 border rounded-[--radius-md]"
                                    >
                                        <div>
                                            <span className="font-medium">{user.username}</span>
                                        </div>

                                        <Button
                                            intent="alert-subtle"
                                            size="sm"
                                            onClick={() => handleRemoveFromWhitelist(user.username)}
                                            disabled={isAdding}
                                        >
                                            Remove
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </SettingsCard>
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
                        <p className="text-sm text-[--muted]">
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
        </>
    )
}