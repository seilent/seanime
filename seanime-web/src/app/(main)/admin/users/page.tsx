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

interface User {
    id: number
    username: string
    isAdmin: boolean
    createdAt: string
    updatedAt: string
}

export default function UsersManagementPage() {
    const [users, setUsers] = useState<User[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [error, setError] = useState('')
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
    const [isCreating, setIsCreating] = useState(false)
    const [newUser, setNewUser] = useState({
        username: '',
        password: '',
        isAdmin: false
    })

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

    // Create user
    const handleCreateUser = async (e: React.FormEvent) => {
        e.preventDefault()
        setIsCreating(true)
        setError('')

        try {
            const response = await fetch('/api/v1/admin/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(newUser),
                credentials: 'include'
            })

            if (response.ok) {
                setIsCreateModalOpen(false)
                setNewUser({ username: '', password: '', isAdmin: false })
                fetchUsers() // Refresh the list
            } else {
                const errorData = await response.json() as { error?: string }
                setError(errorData.error || 'Failed to create user')
            }
        } catch (err) {
            setError('Network error')
        } finally {
            setIsCreating(false)
        }
    }

    // Delete user
    const handleDeleteUser = async (userId: number) => {
        if (!confirm('Are you sure you want to delete this user?')) {
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
                            Manage user accounts and permissions
                        </p>
                    </div>
                    <Button onClick={() => setIsCreateModalOpen(true)}>
                        Create User
                    </Button>
                </div>

                {error && (
                    <Alert intent="alert">
                        {error}
                    </Alert>
                )}

                <Card className="p-6">
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold">Users</h2>
                        
                        {users.length === 0 ? (
                            <p className="text-gray-500">No users found</p>
                        ) : (
                            <div className="space-y-3">
                                {users.map((user) => (
                                    <div
                                        key={user.id}
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
                                            Delete
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </Card>
            </div>

            {/* Create User Modal */}
            <Modal
                open={isCreateModalOpen}
                onOpenChange={setIsCreateModalOpen}
                title="Create New User"
            >
                <form onSubmit={handleCreateUser} className="space-y-4">
                    <TextInput
                        label="Username"
                        placeholder="Enter username"
                        value={newUser.username}
                        onValueChange={(value) => setNewUser(prev => ({ ...prev, username: value }))}
                        required
                        disabled={isCreating}
                    />
                    
                    <TextInput
                        label="Password"
                        type="password"
                        placeholder="Enter password"
                        value={newUser.password}
                        onValueChange={(value) => setNewUser(prev => ({ ...prev, password: value }))}
                        required
                        disabled={isCreating}
                    />
                    
                    <Checkbox
                        label="Admin User"
                        value={newUser.isAdmin}
                        onValueChange={(checked: boolean | "indeterminate") => setNewUser(prev => ({ ...prev, isAdmin: !!checked }))}
                        disabled={isCreating}
                    />
                    
                    <div className="flex gap-2 pt-4">
                        <Button
                            type="button"
                            intent="gray-subtle"
                            onClick={() => setIsCreateModalOpen(false)}
                            disabled={isCreating}
                        >
                            Cancel
                        </Button>
                        <Button
                            type="submit"
                            disabled={isCreating}
                            leftIcon={isCreating ? <LoadingSpinner className="w-4 h-4" /> : undefined}
                        >
                            {isCreating ? 'Creating...' : 'Create User'}
                        </Button>
                    </div>
                </form>
            </Modal>
        </PageWrapper>
    )
}
