"use client"

import React, { useState } from "react"
import { Models_UnmappedFile } from "@/api/generated/types"
import { useMapFileToAniList } from "@/api/hooks/global-mapping"
import { useDebounce } from "@/hooks/use-debounce"
import { Button } from "@/components/ui/button"
import { Modal } from "@/components/ui/modal"
import { TextInput } from "@/components/ui/text-input"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/components/ui/core/styling"
import { FiSearch, FiFile } from "react-icons/fi"
import { useAnilistListAnime } from "@/api/hooks/anilist.hooks"

interface FileMapModalProps {
    isOpen: boolean
    onClose: () => void
    file: Models_UnmappedFile | null
}

export function FileMapModal({ isOpen, onClose, file }: FileMapModalProps) {
    const mapFileMutation = useMapFileToAniList()

    const [searchQuery, setSearchQuery] = useState("")
    const [selectedAnime, setSelectedAnime] = useState<any>(null)
    const [episodeNumber, setEpisodeNumber] = useState("")
    const [customTitle, setCustomTitle] = useState("")
    const [customYear, setCustomYear] = useState("")

    const debouncedSearchQuery = useDebounce(searchQuery, 300)

    // Use the existing AniList search hook
    const { data: searchResults, isLoading: isSearching } = useAnilistListAnime(
        { search: debouncedSearchQuery, perPage: 10 },
        !!debouncedSearchQuery && debouncedSearchQuery.length > 2
    )

    React.useEffect(() => {
        if (file && isOpen) {
            setSearchQuery(file.detectedTitle)
            setCustomTitle(file.detectedTitle)
            setEpisodeNumber("")
            setCustomYear("")
            setSelectedAnime(null)
        }
    }, [file, isOpen])

    const handleMapFile = async () => {
        if (!file || !selectedAnime) return

        const title = customTitle || selectedAnime.title?.english || selectedAnime.title?.romaji || ""
        const year = customYear ? parseInt(customYear) : selectedAnime.startDate?.year || 0
        const episode = episodeNumber ? parseInt(episodeNumber) : 1

        await mapFileMutation.mutateAsync({
            filePath: file.localFilePath,
            anilistId: selectedAnime.id,
            title,
            year,
            episodeNumber: episode,
        })

        onClose()
    }

    const handleClose = () => {
        setSearchQuery("")
        setSelectedAnime(null)
        setEpisodeNumber("")
        setCustomTitle("")
        setCustomYear("")
        onClose()
    }

    if (!file) return null

    return (
        <Modal
            open={isOpen}
            onOpenChange={handleClose}
            title="Map File to AniList Entry"
            description="Map this file to an AniList anime entry for global access."
            contentClass="max-w-2xl max-h-[80vh] flex flex-col"
        >

                <div className="flex-1 space-y-4 overflow-hidden">
                    {/* File Info */}
                    <div className="bg-muted/50 p-3 rounded-lg space-y-2">
                        <div className="flex items-center gap-2 text-sm">
                            <FiFile className="h-4 w-4" />
                            <span className="font-medium">File:</span>
                        </div>
                        <p className="text-sm break-all">{file.localFilePath}</p>
                        <div className="flex items-center gap-4 text-xs text-muted-foreground">
                            <span>Detected: {file.detectedTitle}</span>
                            <span>Size: {Math.round(file.fileSize / (1024 * 1024))} MB</span>
                        </div>
                    </div>

                    {/* Search */}
                    <div className="space-y-2">
                        <label htmlFor="search" className="text-sm font-medium">Search AniList</label>
                        <div>
                            <TextInput
                                placeholder="Search anime on AniList..."
                                value={searchQuery}
                                onValueChange={setSearchQuery}
                                leftIcon={<FiSearch className="h-4 w-4" />}
                            />
                        </div>
                    </div>

                    {/* Search Results */}
                    <div className="space-y-2">
                        <label className="text-sm font-medium">Search Results</label>
                        <ScrollArea className="h-[200px] border rounded-lg">
                            {isSearching ? (
                                <div className="flex items-center justify-center p-4">
                                    <LoadingSpinner />
                                </div>
                            ) : searchResults?.Page?.media?.length ? (
                                <div className="p-2 space-y-2">
                                    {searchResults.Page.media.map((anime: any) => (
                                        <div
                                            key={anime.id}
                                            className={cn(
                                                "p-3 rounded-lg border cursor-pointer transition-colors",
                                                selectedAnime?.id === anime.id
                                                    ? "bg-primary/10 border-primary"
                                                    : "hover:bg-muted/50"
                                            )}
                                            onClick={() => {
                                                setSelectedAnime(anime)
                                                setCustomTitle(anime.title?.english || anime.title?.romaji || "")
                                                setCustomYear(anime.startDate?.year?.toString() || "")
                                            }}
                                        >
                                            <div className="flex gap-3">
                                                {anime.coverImage?.medium && (
                                                    <img
                                                        src={anime.coverImage.medium}
                                                        alt={anime.title?.english || anime.title?.romaji}
                                                        className="w-12 h-16 object-cover rounded"
                                                    />
                                                )}
                                                <div className="flex-1 space-y-1">
                                                    <p className="font-medium text-sm">
                                                        {anime.title?.english || anime.title?.romaji}
                                                    </p>
                                                    {anime.title?.english && anime.title?.romaji && (
                                                        <p className="text-xs text-muted-foreground">
                                                            {anime.title.romaji}
                                                        </p>
                                                    )}
                                                    <div className="flex gap-2 text-xs text-muted-foreground">
                                                        <span>{anime.startDate?.year}</span>
                                                        <span>•</span>
                                                        <span>{anime.format}</span>
                                                        {anime.episodes && (
                                                            <>
                                                                <span>•</span>
                                                                <span>{anime.episodes} episodes</span>
                                                            </>
                                                        )}
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            ) : searchQuery.length > 2 ? (
                                <div className="flex items-center justify-center p-4 text-sm text-muted-foreground">
                                    No results found
                                </div>
                            ) : (
                                <div className="flex items-center justify-center p-4 text-sm text-muted-foreground">
                                    Type to search for anime
                                </div>
                            )}
                        </ScrollArea>
                    </div>

                    {/* Mapping Details */}
                    {selectedAnime && (
                        <div className="space-y-4 pt-2 border-t">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label htmlFor="title" className="text-sm font-medium">Title</label>
                                    <TextInput
                                        placeholder="Custom title"
                                        value={customTitle}
                                        onValueChange={setCustomTitle}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label htmlFor="year" className="text-sm font-medium">Year</label>
                                    <TextInput
                                        placeholder="Year"
                                        value={customYear}
                                        onValueChange={setCustomYear}
                                        type="number"
                                    />
                                </div>
                            </div>
                            <div className="space-y-2">
                                <label htmlFor="episode" className="text-sm font-medium">Episode Number</label>
                                <TextInput
                                    placeholder="Episode number (default: 1)"
                                    value={episodeNumber}
                                    onValueChange={setEpisodeNumber}
                                    type="number"
                                />
                            </div>
                        </div>
                    )}
                </div>

                <div className="flex justify-end gap-2 pt-4">
                    <Button intent="gray-outline" onClick={handleClose}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleMapFile}
                        disabled={!selectedAnime || mapFileMutation.isPending}
                    >
                        {mapFileMutation.isPending ? (
                            <>
                                <LoadingSpinner className="mr-2" />
                                Mapping...
                            </>
                        ) : (
                            "Map File"
                        )}
                    </Button>
                </div>
        </Modal>
    )
}