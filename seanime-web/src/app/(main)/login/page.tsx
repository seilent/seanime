'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { PublicRoute } from '@/components/auth/route-guards'
import { useAuth } from '@/contexts/auth-context'
import { ANILIST_PIN_URL } from '@/lib/server/config'
import Link from 'next/link'

function LoginPageContent() {
    const router = useRouter()
    const { loginWithToken } = useAuth()
    const [token, setToken] = useState("")
    const [isLoading, setIsLoading] = useState(false)
    const [error, setError] = useState("")

    const handleTokenChange = async (value: string) => {
        setToken(value)
        setError("")

        // Only attempt login if token looks valid (has some length)
        if (value.trim().length > 50) { // AniList tokens are typically longer
            setIsLoading(true)
            try {
                const result = await loginWithToken(value.trim())
                if (result.success) {
                    router.push('/')
                } else {
                    setError(result.error || "Invalid token")
                }
            } catch (err) {
                setError("Failed to validate token")
            } finally {
                setIsLoading(false)
            }
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
            <Card className="w-full max-w-md p-6">
                <div className="space-y-6">
                    <div className="text-center space-y-2">
                        <div className="mb-4 flex justify-center w-full">
                            <img src="/logo.png" alt="logo" className="w-24 h-auto" />
                        </div>
                        <h1 className="text-2xl font-bold">
                            Sign in to Seanime
                        </h1>
                        <p className="text-gray-600 dark:text-gray-400">
                            Use your AniList account to access Seanime
                        </p>
                    </div>

                    <div className="space-y-4">
                        <Link
                            href={ANILIST_PIN_URL}
                            target="_blank"
                        >
                            <Button
                                leftIcon={<svg
                                    xmlns="http://www.w3.org/2000/svg" fill="currentColor" width="24" height="24"
                                    viewBox="0 0 24 24" role="img"
                                >
                                    <path
                                        d="M6.361 2.943 0 21.056h4.942l1.077-3.133H11.4l1.052 3.133H22.9c.71 0 1.1-.392 1.1-1.101V17.53c0-.71-.39-1.101-1.1-1.101h-6.483V4.045c0-.71-.392-1.102-1.101-1.102h-2.422c-.71 0-1.101.392-1.101 1.102v1.064l-.758-2.166zm2.324 5.948 1.688 5.018H7.144z"
                                    />
                                </svg>}
                                intent="white"
                                size="md"
                                className="w-full"
                            >Get AniList token</Button>
                        </Link>

                        <div className="space-y-2">
                            <label className="text-base font-semibold">AniList Token</label>
                            <textarea
                                value={token}
                                onChange={(e) => handleTokenChange(e.target.value)}
                                placeholder="Paste your AniList token here"
                                className="w-full h-32 p-3 rounded-lg bg-[--paper] border border-[--border] placeholder-gray-400 dark:placeholder-gray-500 focus:border-brand focus:ring-1 focus:ring-[--ring] outline-0 transition duration-150 resize-none"
                                disabled={isLoading}
                            />
                            {error && (
                                <div className="text-sm text-red-500">
                                    {error}
                                </div>
                            )}
                            {isLoading && (
                                <div className="text-sm text-[--muted]">
                                    Validating token and logging you in...
                                </div>
                            )}
                        </div>

                        <div className="text-center text-sm text-gray-500 dark:text-gray-400">
                            Only whitelisted AniList accounts can access this server
                        </div>
                    </div>
                </div>
            </Card>
        </div>
    )
}

export default function LoginPage() {
    return (
        <PublicRoute>
            <LoginPageContent />
        </PublicRoute>
    )
}