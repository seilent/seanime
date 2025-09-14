"use client"

import React, { useState } from "react"
import { Models_GlobalAnimeFileMapping } from "@/api/generated/types"
import { useGetGlobalMappings, useRemoveMapping } from "@/api/hooks/global-mapping"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { DataGrid } from "@/components/ui/datagrid"
import { TextInput } from "@/components/ui/text-input"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { cn } from "@/components/ui/core/styling"
import { formatDistanceToNow } from "date-fns"
import { Badge } from "@/components/ui/badge"
import { FiMap as Map, FiTrash2 as Trash2, FiFile as FileText, FiHash as Hash, FiCalendar as Calendar } from "react-icons/fi"
import { AiOutlineExclamationCircle as AlertCircle } from "react-icons/ai"
import { Modal } from "@/components/ui/modal"

interface GlobalMappingsTableProps {
    className?: string
}

export function GlobalMappingsTable({ className }: GlobalMappingsTableProps) {
    const { data: globalMappings, isLoading, error } = useGetGlobalMappings()
    const removeMappingMutation = useRemoveMapping()

    const [searchTerm, setSearchTerm] = useState("")
    const [mappingToRemove, setMappingToRemove] = useState<Models_GlobalAnimeFileMapping | null>(null)

    const filteredMappings = React.useMemo(() => {
        if (!globalMappings) return []
        if (!searchTerm) return globalMappings

        return globalMappings.filter(mapping =>
            mapping.localFilePath.toLowerCase().includes(searchTerm.toLowerCase()) ||
            mapping.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
            mapping.anilistId.toString().includes(searchTerm)
        )
    }, [globalMappings, searchTerm])

    const handleRemoveMapping = async (mapping: Models_GlobalAnimeFileMapping) => {
        setMappingToRemove(mapping)
    }

    const confirmRemoveMapping = async () => {
        if (!mappingToRemove) return

        await removeMappingMutation.mutateAsync({
            filePath: mappingToRemove.localFilePath,
        })
        setMappingToRemove(null)
    }

    if (isLoading) {
        return (
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Map className="h-5 w-5" />
                        Global Mappings
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
                        Error Loading Global Mappings
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground">
                        Failed to load global mappings: {error?.message}
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <>
            <Card className={className}>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Map className="h-5 w-5" />
                        Global Mappings
                        {globalMappings && (
                            <Badge>
                                {globalMappings.length}
                            </Badge>
                        )}
                    </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <TextInput
                        placeholder="Search mappings by path, title, or AniList ID..."
                        value={searchTerm}
                        onValueChange={setSearchTerm}
                    />

                    {filteredMappings.length === 0 ? (
                        <div className="text-center py-8">
                            <Map className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                            <p className="text-lg font-medium">No global mappings</p>
                            <p className="text-sm text-muted-foreground">
                                {searchTerm ? "No mappings match your search." : "No files have been mapped yet."}
                            </p>
                        </div>
                    ) : (
                        <DataGrid
                            data={filteredMappings}
                            rowCount={filteredMappings.length}
                            columns={[
                                {
                                    accessorKey: "title",
                                    header: "Anime Title",
                                    cell: ({ row }) => (
                                        <div>
                                            <p className="font-medium">{row.original.title}</p>
                                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                <Hash className="h-3 w-3" />
                                                <span>{row.original.anilistId}</span>
                                                <Calendar className="h-3 w-3" />
                                                <span>{row.original.year}</span>
                                            </div>
                                        </div>
                                    ),
                                },
                                {
                                    accessorKey: "localFilePath",
                                    header: "File Path",
                                    cell: ({ row }) => (
                                        <div>
                                            <p className="text-sm truncate max-w-[300px]" title={row.original.localFilePath}>
                                                {row.original.localFilePath}
                                            </p>
                                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                <FileText className="h-3 w-3" />
                                                <span>{(() => {
                                                    const bytes = row.original.fileSize
                                                    if (bytes === 0) return "0 B"
                                                    const k = 1024
                                                    const sizes = ["B", "KB", "MB", "GB", "TB"]
                                                    const i = Math.floor(Math.log(bytes) / Math.log(k))
                                                    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
                                                })()}</span>
                                                <span>•</span>
                                                <span>Episode {row.original.episodeNumber}</span>
                                            </div>
                                        </div>
                                    ),
                                },
                                {
                                    accessorKey: "mappedAt",
                                    header: "Mapped",
                                    cell: ({ row }) => {
                                        if (!row.original.createdAt) return "Unknown"
                                        return formatDistanceToNow(new Date(row.original.createdAt), { addSuffix: true })
                                    },
                                },
                                {
                                    id: "actions",
                                    header: "Actions",
                                    cell: ({ row }) => (
                                        <Button
                                            intent="gray-outline"
                                            size="sm"
                                            onClick={() => handleRemoveMapping(row.original)}
                                            disabled={removeMappingMutation.isPending}
                                        >
                                            <Trash2 className="h-4 w-4 mr-2" />
                                            Remove
                                        </Button>
                                    ),
                                },
                            ]}
                        />
                    )}
                </CardContent>
            </Card>

            <Modal
                open={!!mappingToRemove}
                onOpenChange={() => setMappingToRemove(null)}
                title="Remove Global Mapping"
                description="Are you sure you want to remove this global mapping? This will disconnect the file from AniList entry."
            >
                {mappingToRemove && (
                    <div className="space-y-4">
                        <div className="p-3 bg-muted rounded-lg">
                            <p className="font-medium">{mappingToRemove.title}</p>
                            <p className="text-sm text-muted-foreground break-all">
                                {mappingToRemove.localFilePath}
                            </p>
                        </div>
                        <div className="flex justify-end gap-2">
                            <Button intent="gray-outline" onClick={() => setMappingToRemove(null)}>
                                Cancel
                            </Button>
                            <Button onClick={confirmRemoveMapping}>
                                Remove Mapping
                            </Button>
                        </div>
                    </div>
                )}
            </Modal>
        </>
    )
}