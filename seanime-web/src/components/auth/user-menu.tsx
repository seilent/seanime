"use client"

import React from 'react'
import { useAuth } from '@/contexts/auth-context'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Avatar } from '@/components/ui/avatar'
import { useRouter } from 'next/navigation'
import { BiUser, BiLogOut, BiCog } from 'react-icons/bi'
import { HiOutlineServerStack } from 'react-icons/hi2'

export function UserMenu() {
    const { user, viewer, logout, isAdmin } = useAuth()
    const router = useRouter()

    if (!user) return null

    const handleLogout = async () => {
        await logout()
    }

    const handleSettings = () => {
        router.push('/settings')
    }

    const handleUserManagement = () => {
        router.push('/admin/users')
    }

    return (
        <DropdownMenu
            trigger={
                <Button
                    intent="gray-subtle"
                    size="sm"
                    className="flex items-center gap-2 px-2"
                >
                    <Avatar className="w-6 h-6" src={viewer?.avatar?.medium}>
                        {!viewer?.avatar?.medium && (
                            <div className="w-full h-full bg-gradient-to-br from-brand-500 to-purple-600 flex items-center justify-center text-white text-xs font-medium">
                                {user.displayName?.charAt(0)?.toUpperCase() || user.username?.charAt(0)?.toUpperCase() || 'U'}
                            </div>
                        )}
                    </Avatar>
                    <span className="hidden sm:inline text-sm font-medium">
                        {user.displayName || user.username}
                    </span>
                </Button>
            }
        >
            <div className="px-2 py-1.5">
                <p className="text-sm font-medium">{user.displayName || user.username}</p>
                <p className="text-xs text-[--muted]">@{user.username}</p>
                {user.role === 'admin' && (
                    <p className="text-xs text-brand-500 font-medium">Administrator</p>
                )}
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleSettings}>
                <BiCog className="mr-2 h-4 w-4" />
                Settings
            </DropdownMenuItem>
            {isAdmin && (
                <DropdownMenuItem onClick={handleUserManagement}>
                    <HiOutlineServerStack className="mr-2 h-4 w-4" />
                    User Management
                </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout} className="text-red-600 focus:text-red-600">
                <BiLogOut className="mr-2 h-4 w-4" />
                Sign out
            </DropdownMenuItem>
        </DropdownMenu>
    )
}
