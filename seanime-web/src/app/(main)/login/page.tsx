'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { TextInput } from '@/components/ui/text-input'
import { Card } from '@/components/ui/card'
import { Alert } from '@/components/ui/alert'
import { LoadingSpinner } from '@/components/ui/loading-spinner'

export default function LoginPage() {
    const [username, setUsername] = useState('')
    const [password, setPassword] = useState('')
    const [isLoading, setIsLoading] = useState(false)
    const [error, setError] = useState('')
    const router = useRouter()

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault()
        setIsLoading(true)
        setError('')

        try {
            const response = await fetch('/api/v1/users/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password }),
                credentials: 'include', // Important for cookies
            })

            if (response.ok) {
                // Login successful, redirect to main page
                router.push('/')
                router.refresh()
            } else {
                const errorData = await response.json() as { error?: string }
                setError(errorData.error || 'Login failed')
            }
        } catch (err) {
            setError('Network error. Please try again.')
        } finally {
            setIsLoading(false)
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
            <Card className="w-full max-w-md p-6">
                <div className="space-y-6">
                    <div className="text-center space-y-2">
                        <h1 className="text-2xl font-bold">
                            Sign in to Seanime
                        </h1>
                        <p className="text-gray-600 dark:text-gray-400">
                            Enter your credentials to access your account
                        </p>
                    </div>
                    
                    <form onSubmit={handleLogin} className="space-y-4">
                        <TextInput
                            label="Username"
                            placeholder="Enter your username"
                            value={username}
                            onValueChange={setUsername}
                            required
                            disabled={isLoading}
                        />
                        
                        <TextInput
                            label="Password"
                            type="password"
                            placeholder="Enter your password"
                            value={password}
                            onValueChange={setPassword}
                            required
                            disabled={isLoading}
                        />
                        
                        {error && (
                            <Alert intent="alert">
                                {error}
                            </Alert>
                        )}
                        
                        <Button
                            type="submit"
                            className="w-full"
                            disabled={isLoading}
                            leftIcon={isLoading ? <LoadingSpinner className="w-4 h-4" /> : undefined}
                        >
                            {isLoading ? 'Signing in...' : 'Sign in'}
                        </Button>
                    </form>
                </div>
            </Card>
        </div>
    )
}
