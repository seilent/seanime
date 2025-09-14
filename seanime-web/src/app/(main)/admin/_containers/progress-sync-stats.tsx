"use client"

import React from "react"
import { useGetProgressSyncStats, useRetryFailedSync } from "@/api/hooks/global-mapping"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { Badge } from "@/components/ui/badge"
import { ProgressBar } from "@/components/ui/progress-bar"
import {
    FiClock as Clock,
    FiCheckCircle as CheckCircle,
    FiXCircle as XCircle,
    FiRefreshCcw as RefreshCcw,
    FiTrendingUp as TrendingUp,
    FiActivity as Activity
} from "react-icons/fi"
import { AiOutlineExclamationCircle as AlertCircle } from "react-icons/ai"
import { RiBarChart2Line as BarChart3 } from "react-icons/ri"

interface ProgressSyncStatsProps {
    className?: string
}

export function ProgressSyncStats({ className }: ProgressSyncStatsProps) {
    const { data: stats, isLoading, error } = useGetProgressSyncStats()
    const retryFailedMutation = useRetryFailedSync()

    const handleRetryFailed = async () => {
        await retryFailedMutation.mutateAsync()
    }

    if (isLoading) {
        return (
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <BarChart3 className="h-5 w-5" />
                        Progress Sync Statistics
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <LoadingSpinner />
                </CardContent>
            </Card>
        )
    }

    if (error) {
        return (
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-red-500">
                        <AlertCircle className="h-5 w-5" />
                        Error Loading Stats
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground">
                        Failed to load progress sync statistics: {error?.message}
                    </p>
                </CardContent>
            </Card>
        )
    }

    if (!stats) {
        return (
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <BarChart3 className="h-5 w-5" />
                        Progress Sync Statistics
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground">No statistics available</p>
                </CardContent>
            </Card>
        )
    }

    const pending = stats.pending || 0
    const synced = stats.synced || 0
    const failed = stats.failed || 0
    const total = pending + synced + failed

    const successRate = total > 0 ? Math.round((synced / total) * 100) : 0
    const failureRate = total > 0 ? Math.round((failed / total) * 100) : 0

    return (
        <Card className={className}>
            <CardHeader>
                <div className="flex items-center justify-between">
                    <CardTitle className="flex items-center gap-2">
                        <BarChart3 className="h-5 w-5" />
                        Progress Sync Statistics
                    </CardTitle>
                    {failed > 0 && (
                        <Button
                            intent="gray-outline"
                            size="sm"
                            onClick={handleRetryFailed}
                            disabled={retryFailedMutation.isPending}
                        >
                            <RefreshCcw className="h-4 w-4 mr-2" />
                            Retry Failed
                        </Button>
                    )}
                </div>
            </CardHeader>
            <CardContent className="space-y-6">
                {/* Overview Cards */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="bg-muted/50 p-4 rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                            <Clock className="h-4 w-4 text-yellow-500" />
                            <span className="text-sm font-medium">Pending</span>
                        </div>
                        <div className="text-2xl font-bold">{pending}</div>
                        <p className="text-xs text-muted-foreground">Items waiting to sync</p>
                    </div>

                    <div className="bg-muted/50 p-4 rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                            <CheckCircle className="h-4 w-4 text-green-500" />
                            <span className="text-sm font-medium">Synced</span>
                        </div>
                        <div className="text-2xl font-bold">{synced}</div>
                        <p className="text-xs text-muted-foreground">Successfully synced</p>
                    </div>

                    <div className="bg-muted/50 p-4 rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                            <XCircle className="h-4 w-4 text-red-500" />
                            <span className="text-sm font-medium">Failed</span>
                        </div>
                        <div className="text-2xl font-bold">{failed}</div>
                        <p className="text-xs text-muted-foreground">Failed to sync</p>
                    </div>
                </div>

                {/* Progress Bar */}
                {total > 0 && (
                    <div className="space-y-2">
                        <div className="flex justify-between text-sm">
                            <span>Sync Progress</span>
                            <span>{Math.round(((synced + failed) / total) * 100)}% processed</span>
                        </div>
                        <ProgressBar value={((synced + failed) / total) * 100} size="sm" />
                        <div className="flex justify-between text-xs text-muted-foreground">
                            <span>{synced + failed} / {total} items processed</span>
                            <span>{pending} remaining</span>
                        </div>
                    </div>
                )}

                {/* Success/Failure Rates */}
                {total > 0 && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="bg-green-50 dark:bg-green-950/20 p-4 rounded-lg border border-green-200 dark:border-green-800">
                            <div className="flex items-center gap-2 mb-2">
                                <TrendingUp className="h-4 w-4 text-green-600" />
                                <span className="text-sm font-medium text-green-800 dark:text-green-200">
                                    Success Rate
                                </span>
                            </div>
                            <div className="text-2xl font-bold text-green-800 dark:text-green-200">
                                {successRate}%
                            </div>
                            <p className="text-xs text-green-600 dark:text-green-400">
                                {synced} out of {synced + failed} processed
                            </p>
                        </div>

                        <div className="bg-red-50 dark:bg-red-950/20 p-4 rounded-lg border border-red-200 dark:border-red-800">
                            <div className="flex items-center gap-2 mb-2">
                                <Activity className="h-4 w-4 text-red-600" />
                                <span className="text-sm font-medium text-red-800 dark:text-red-200">
                                    Failure Rate
                                </span>
                            </div>
                            <div className="text-2xl font-bold text-red-800 dark:text-red-200">
                                {failureRate}%
                            </div>
                            <p className="text-xs text-red-600 dark:text-red-400">
                                {failed} out of {synced + failed} processed
                            </p>
                        </div>
                    </div>
                )}

                {/* Status Messages */}
                <div className="space-y-2">
                    {pending > 0 && (
                        <div className="flex items-center gap-2 text-sm text-yellow-600 dark:text-yellow-400">
                            <Clock className="h-4 w-4" />
                            <span>{pending} items are queued for sync</span>
                        </div>
                    )}

                    {failed > 0 && (
                        <div className="flex items-center gap-2 text-sm text-red-600 dark:text-red-400">
                            <XCircle className="h-4 w-4" />
                            <span>{failed} items failed to sync and may need manual attention</span>
                        </div>
                    )}

                    {total === 0 && (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <CheckCircle className="h-4 w-4" />
                            <span>No progress sync items in queue</span>
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}