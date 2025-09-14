"use client"

import React, { useState } from "react"
import { Models_UnmappedFile } from "@/api/generated/types"
import { useGetIgnoredFiles, useUnignoreFile } from "@/api/hooks/global-mapping"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { DataGrid } from "@/components/ui/datagrid"
import { TextInput } from "@/components/ui/text-input"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { cn } from "@/components/ui/core/styling"
import { formatDistanceToNow } from "date-fns"
import { Badge } from "@/components/ui/badge"
import { FiEyeOff as EyeOff, FiEye as Eye } from "react-icons/fi"
import { AiOutlineExclamationCircle as AlertCircle } from "react-icons/ai"

interface IgnoredFilesTableProps {
    className?: string
}

export function IgnoredFilesTable({ className }: IgnoredFilesTableProps) {
    const { data: ignoredFiles, isLoading, error } = useGetIgnoredFiles()
    const unignoreFileMutation = useUnignoreFile()

    const [searchTerm, setSearchTerm] = useState("")

    const filteredFiles = React.useMemo(() => {
        if (!ignoredFiles) return []
        if (!searchTerm) return ignoredFiles

        return ignoredFiles.filter(file =>
            file.localFilePath.toLowerCase().includes(searchTerm.toLowerCase()) ||
            file.detectedTitle.toLowerCase().includes(searchTerm.toLowerCase())
        )
    }, [ignoredFiles, searchTerm])

    const handleUnignoreFile = async (file: Models_UnmappedFile) => {
        await unignoreFileMutation.mutateAsync({
            filePath: file.localFilePath,
        })
    }

    const handleBulkUnignore = async () => {
        if (filteredFiles.length === 0) return

        for (const file of filteredFiles) {
            try {
                await unignoreFileMutation.mutateAsync({
                    filePath: file.localFilePath,
                })
            } catch (error) {
                console.error("Failed to unignore file:", file.localFilePath, error)
            }
        }
    }

    if (isLoading) {
        return (
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <EyeOff className="h-5 w-5" />
                        Ignored Files
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
                        Error Loading Ignored Files
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground">
                        Failed to load ignored files: {error?.message}
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <Card className={className}>
            <CardHeader>
                <div className="flex items-center justify-between">
                    <CardTitle className="flex items-center gap-2">
                        <EyeOff className="h-5 w-5" />
                        Ignored Files
                        {ignoredFiles && (
                            <Badge>
                                {ignoredFiles.length}
                            </Badge>
                        )}
                    </CardTitle>
                    <div className="flex gap-2">
                        {filteredFiles.length > 0 && (
                            <Button
                                intent="gray-outline"
                                size="sm"
                                onClick={handleBulkUnignore}
                                disabled={unignoreFileMutation.isPending}
                            >
                                <Eye className="h-4 w-4 mr-2" />
                                Unignore All Filtered
                            </Button>
                        )}
                    </div>
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                <TextInput
                    placeholder="Search files by path or detected title..."
                    value={searchTerm}
                    onValueChange={setSearchTerm}
                />

                {filteredFiles.length === 0 ? (
                    <div className="text-center py-8">
                        <EyeOff className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                        <p className="text-lg font-medium">No ignored files</p>
                        <p className="text-sm text-muted-foreground">
                            {searchTerm ? "No files match your search." : "No files have been ignored."}
                        </p>
                    </div>
                ) : (
                    <DataGrid
                        data={filteredFiles}
                        rowCount={filteredFiles.length}
                        columns={[
                            {
                                accessorKey: "detectedTitle",
                                header: "Detected Title",
                                cell: ({ row }) => (
                                    <div>
                                        <p className="font-medium">{row.original.detectedTitle}</p>
                                        <p className="text-sm text-muted-foreground truncate max-w-[300px]" title={row.original.localFilePath}>
                                            {row.original.localFilePath}
                                        </p>
                                    </div>
                                ),
                            },
                            {
                                accessorKey: "fileSize",
                                header: "Size",
                                cell: ({ row }) => {
                                        const bytes = row.original.fileSize
                                        if (bytes === 0) return "0 B"
                                        const k = 1024
                                        const sizes = ["B", "KB", "MB", "GB", "TB"]
                                        const i = Math.floor(Math.log(bytes) / Math.log(k))
                                        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
                                    },
                            },
                            {
                                accessorKey: "ignoredAt",
                                header: "Ignored At",
                                cell: ({ row }) => {
                                    if (!row.original.ignoredAt) return "Unknown"
                                    return formatDistanceToNow(new Date(row.original.ignoredAt), { addSuffix: true })
                                },
                            },
                            {
                                id: "actions",
                                header: "Actions",
                                cell: ({ row }) => (
                                    <Button
                                        intent="gray-outline"
                                        size="sm"
                                        onClick={() => handleUnignoreFile(row.original)}
                                        disabled={unignoreFileMutation.isPending}
                                    >
                                        <Eye className="h-4 w-4 mr-2" />
                                        Unignore
                                    </Button>
                                ),
                            },
                        ]}
                    />
                )}
            </CardContent>
        </Card>
    )
}