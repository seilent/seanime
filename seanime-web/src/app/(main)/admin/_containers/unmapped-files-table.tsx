"use client"

import React, { useState } from "react"
import { Models_UnmappedFile } from "@/api/generated/types"
import { useGetUnmappedFiles, useIgnoreFile } from "@/api/hooks/global-mapping"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { DataGrid } from "@/components/ui/datagrid"
import { TextInput } from "@/components/ui/text-input"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { cn } from "@/components/ui/core/styling"
import { formatDistanceToNow } from "date-fns"
import { FileMapModal } from "./file-map-modal"
import { Badge } from "@/components/ui/badge"
import { AiOutlineExclamationCircle as AlertCircle } from "react-icons/ai"
import { FiEye as Eye, FiEyeOff as EyeOff, FiMap as Map } from "react-icons/fi"

interface UnmappedFilesTableProps {
    className?: string
}

export function UnmappedFilesTable({ className }: UnmappedFilesTableProps) {
    const { data: unmappedFiles, isLoading, error } = useGetUnmappedFiles()
    const ignoreFileMutation = useIgnoreFile()

    const [searchTerm, setSearchTerm] = useState("")
    const [selectedFile, setSelectedFile] = useState<Models_UnmappedFile | null>(null)
    const [isMapModalOpen, setIsMapModalOpen] = useState(false)

    const filteredFiles = React.useMemo(() => {
        if (!unmappedFiles) return []
        if (!searchTerm) return unmappedFiles

        return unmappedFiles.filter(file =>
            file.localFilePath.toLowerCase().includes(searchTerm.toLowerCase()) ||
            file.detectedTitle.toLowerCase().includes(searchTerm.toLowerCase())
        )
    }, [unmappedFiles, searchTerm])

    const handleMapFile = (file: Models_UnmappedFile) => {
        setSelectedFile(file)
        setIsMapModalOpen(true)
    }

    const handleIgnoreFile = async (file: Models_UnmappedFile) => {
        await ignoreFileMutation.mutateAsync({
            filePath: file.localFilePath,
        })
    }

    const handleBulkIgnore = async () => {
        // For now, we'll implement this as ignoring all filtered files
        // In a more sophisticated implementation, we'd have checkboxes for selection
        if (filteredFiles.length === 0) return

        for (const file of filteredFiles) {
            try {
                await ignoreFileMutation.mutateAsync({
                    filePath: file.localFilePath,
                })
            } catch (error) {
                console.error("Failed to ignore file:", file.localFilePath, error)
            }
        }
    }

    if (isLoading) {
        return (
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <AlertCircle className="h-5 w-5" />
                        Unmapped Files
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
                        Error Loading Unmapped Files
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground">
                        Failed to load unmapped files: {error?.message}
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <>
            <Card className={className}>
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <CardTitle className="flex items-center gap-2">
                            <AlertCircle className="h-5 w-5" />
                            Unmapped Files
                            {unmappedFiles && (
                                <Badge>
                                    {unmappedFiles.length}
                                </Badge>
                            )}
                        </CardTitle>
                        <div className="flex gap-2">
                            {filteredFiles.length > 0 && (
                                <Button
                                    intent="gray-outline"
                                    size="sm"
                                    onClick={handleBulkIgnore}
                                    disabled={ignoreFileMutation.isPending}
                                >
                                    <EyeOff className="h-4 w-4 mr-2" />
                                    Ignore All Filtered
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
                            <AlertCircle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                            <p className="text-lg font-medium">No unmapped files</p>
                            <p className="text-sm text-muted-foreground">
                                {searchTerm ? "No files match your search." : "All files have been mapped or ignored."}
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
                                        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
                                    },
                                },
                                {
                                    accessorKey: "lastDetected",
                                    header: "Last Detected",
                                    cell: ({ row }) => {
                                        if (!row.original.lastDetected) return "Unknown"
                                        return formatDistanceToNow(new Date(row.original.lastDetected), { addSuffix: true })
                                    },
                                },
                                {
                                    id: "actions",
                                    header: "Actions",
                                    cell: ({ row }) => (
                                        <div className="flex gap-2">
                                            <Button
                                                intent="gray-outline"
                                                size="sm"
                                                onClick={() => handleMapFile(row.original)}
                                            >
                                                <Map className="h-4 w-4 mr-2" />
                                                Map
                                            </Button>
                                            <Button
                                                intent="gray-outline"
                                                size="sm"
                                                onClick={() => handleIgnoreFile(row.original)}
                                                disabled={ignoreFileMutation.isPending}
                                            >
                                                <EyeOff className="h-4 w-4 mr-2" />
                                                Ignore
                                            </Button>
                                        </div>
                                    ),
                                },
                            ]}
                        />
                    )}
                </CardContent>
            </Card>

            <FileMapModal
                isOpen={isMapModalOpen}
                onClose={() => {
                    setIsMapModalOpen(false)
                    setSelectedFile(null)
                }}
                file={selectedFile}
            />
        </>
    )
}