import { Anime_Entry, HibikeTorrent_AnimeTorrent } from "@/api/generated/types"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import {
    TorrentResolutionBadge,
    TorrentSeedersBadge,
} from "@/app/(main)/entry/_containers/torrent-search/_components/torrent-item-badges"
import { TorrentPreviewItem } from "@/app/(main)/entry/_containers/torrent-search/_components/torrent-preview-item"
import { TorrentPreviewList } from "@/app/(main)/entry/_containers/torrent-search/_components/torrent-preview-list"
import { TorrentTable } from "@/app/(main)/entry/_containers/torrent-search/_components/torrent-table"
import { Torrent_SearchType, useHandleTorrentSearch } from "@/app/(main)/entry/_containers/torrent-search/_lib/handle-torrent-search"
import { useTorrentSearchSelectedStreamEpisode } from "@/app/(main)/entry/_containers/torrent-search/_lib/handle-torrent-selection"
import { TorrentConfirmationModal } from "@/app/(main)/entry/_containers/torrent-search/torrent-confirmation-modal"
import { __torrentSearch_selectionAtom, TorrentSelectionType } from "@/app/(main)/entry/_containers/torrent-search/torrent-search-drawer"
import { LuffyError } from "@/components/shared/luffy-error"
import { Alert } from "@/components/ui/alert"
import { AppLayoutStack } from "@/components/ui/app-layout"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/components/ui/core/styling"
import { DataGridSearchInput } from "@/components/ui/datagrid"
import { NumberInput } from "@/components/ui/number-input"
import { Select } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { formatDistanceToNowSafe } from "@/lib/helpers/date"
import { TORRENT_PROVIDER } from "@/lib/server/settings"
import { subDays, subMonths } from "date-fns"
import { atom, useSetAtom } from "jotai"
import { useAtom } from "jotai/react"
import React, { startTransition } from "react"
import { BiCalendarAlt } from "react-icons/bi"
import { LuCornerLeftDown, LuGem } from "react-icons/lu"
import { RiFolderDownloadFill } from "react-icons/ri"

export const __torrentSearch_selectedTorrentsAtom = atom<HibikeTorrent_AnimeTorrent[]>([])

export function TorrentSearchContainer({ type, entry }: { type: TorrentSelectionType, entry: Anime_Entry }) {
    const downloadInfo = React.useMemo(() => entry.downloadInfo, [entry.downloadInfo])
    const serverStatus = useServerStatus()

    const shouldLookForBatches = React.useMemo(() => {
        const endedDate = entry.media?.endDate?.year ? new Date(entry.media?.endDate?.year,
            entry.media?.endDate?.month ? entry.media?.endDate?.month - 1 : 0,
            entry.media?.endDate?.day || 0) : null
        const now = new Date()
        let flag = true

        if (type === "download") {
            if (endedDate && subDays(now, 6) < endedDate) {
                flag = false
            }
            return !!downloadInfo?.canBatch && !!downloadInfo?.episodesToDownload?.length && flag
        } else {
            if (endedDate && subMonths(now, 1) < endedDate) {
                flag = false
            }
            return !!downloadInfo?.canBatch && flag
        }
    }, [downloadInfo?.canBatch, downloadInfo?.episodesToDownload?.length, type, entry.media?.endDate])

    const hasEpisodesToDownload = React.useMemo(() => !!downloadInfo?.episodesToDownload?.length, [downloadInfo?.episodesToDownload?.length])
    const [isAdult, setIsAdult] = React.useState(entry.media?.isAdult === true)

    const {
        warnings,
        hasOneWarning,
        selectedProviderExtension,
        selectedProviderExtensionId,
        setSelectedProviderExtensionId,
        providerExtensions,
        globalFilter,
        setGlobalFilter,
        selectedTorrents,
        setSelectedTorrents,
        searchType,
        setSearchType,
        smartSearchBatch,
        setSmartSearchBatch,
        smartSearchEpisode,
        setSmartSearchEpisode,
        smartSearchResolution,
        setSmartSearchResolution,
        smartSearchBest,
        setSmartSearchBest,
        data,
        isLoading,
        isFetching,
        soughtEpisode,
    } = useHandleTorrentSearch({
        isAdult,
        hasEpisodesToDownload,
        shouldLookForBatches,
        downloadInfo,
        entry,
        type,
    })

    React.useEffect(() => {
        setSelectedTorrents([])
    }, [])

    React.useLayoutEffect(() => {
        if (searchType === Torrent_SearchType.SMART) {
            setGlobalFilter("")
        } else if (searchType === Torrent_SearchType.SIMPLE) {
            const title = entry.media?.title?.romaji || entry.media?.title?.english || entry.media?.title?.userPreferred
            setGlobalFilter(title?.replaceAll(":", "").replaceAll("-", "") || "")
        }
    }, [searchType, entry.media?.title])

    const torrents = React.useMemo(() => data?.torrents ?? [], [data?.torrents])
    const previews = React.useMemo(() => data?.previews ?? [], [data?.previews])

    /**
     * Select torrent
     * - Download: Select multiple torrents, but only ONE batch torrent at a time
     * - Select: Select only one torrent
     */
    const handleToggleTorrent = React.useCallback((t: HibikeTorrent_AnimeTorrent) => {
        if (type === "download") {
            setSelectedTorrents(prev => {
                const idx = prev.findIndex(n => n.link === t.link)

                // Deselecting
                if (idx !== -1) {
                    return prev.filter(n => n.link !== t.link)
                }

                // Selecting - check if trying to select multiple batches
                const isBatchTorrent = t.isBatch
                const hasBatchInSelection = prev.some(torrent => torrent.isBatch)

                if (isBatchTorrent && hasBatchInSelection) {
                    // Prevent selecting multiple batches
                    // TODO: Show toast/notification to user
                    console.warn("Only one batch can be selected at a time")
                    return prev // Don't add the new batch
                }

                return [...prev, t]
            })
        } else {
            setSelectedTorrents(prev => {
                const idx = prev.findIndex(n => n.link === t.link)
                if (idx !== -1) {
                    return []
                }
                return [t]
            })
        }
    }, [setSelectedTorrents, smartSearchBest, type])

    /**
     * Handle streams
     */


    return (
        <>
                <TorrentSearchTorrentStreamBatchHistory
                    type={type}
                    entry={entry}
                />

            <AppLayoutStack className="Sea-TorrentSearchContainer__root space-y-4" data-torrent-search-container>

                <div
                    className="Sea-TorrentSearchContainer__paramContainer flex flex-wrap gap-3 items-center"
                    data-torrent-search-container-param-container
                >
                    <div className="w-[200px]" data-torrent-search-container-param-container-provider-select-container>
                        <Select
                            name="torrentProvider"
                            // leftAddon="Torrent Provider"
                            value={selectedProviderExtension?.id ?? TORRENT_PROVIDER.NONE}
                            onValueChange={setSelectedProviderExtensionId}
                            leftIcon={<RiFolderDownloadFill className="text-[--brand]" />}
                            options={[
                                ...(providerExtensions?.map(ext => ({
                                    label: ext.name,
                                    value: ext.id,
                                })) ?? []).sort((a, b) => a?.label?.localeCompare(b?.label) ?? 0),
                                { label: "None", value: TORRENT_PROVIDER.NONE },
                            ]}
                        />
                    </div>

                    {(selectedProviderExtensionId !== "none" && selectedProviderExtensionId !== "") && <>
                        <div
                            className="h-10 rounded-[--radius] px-2 flex items-center"
                            data-torrent-search-container-param-container-smart-search-switch-container
                        >
                            <Switch
                                // side="right"
                                label="Smart search"
                                moreHelp={selectedProviderExtension?.settings?.canSmartSearch
                                    ? "Automatically search based on given parameters"
                                    : "This provider does not support smart search"}
                                value={searchType === Torrent_SearchType.SMART}
                                onValueChange={v => setSearchType(v ? Torrent_SearchType.SMART : Torrent_SearchType.SIMPLE)}
                                disabled={!selectedProviderExtension?.settings?.canSmartSearch}
                                containerClass="flex-row-reverse gap-1"
                            />
                        </div>

                        {entry.media?.isAdult === false && <div
                            className="h-10 rounded-[--radius] px-2 flex items-center"
                            data-torrent-search-container-param-container-adult-switch-container
                        >
                            <Switch
                                // side="right"
                                label="Adult"
                                moreHelp="If enabled, the adult content flag will be passed to the provider."
                                value={isAdult}
                                onValueChange={setIsAdult}
                                containerClass="flex-row-reverse gap-1"
                            />
                        </div>}
                    </>}
                </div>

                {(selectedProviderExtensionId !== "none" && selectedProviderExtensionId !== "") && Object.keys(warnings)?.map((key) => {
                    if ((warnings as any)[key]) {
                        return <Alert
                            data-torrent-search-container-warning
                            key={key}
                            intent="warning"
                            description={<>
                                {key === "extensionDoesNotSupportAdult" && "This provider does not support adult content"}
                                {key === "extensionDoesNotSupportSmartSearch" && "This provider does not support smart search"}
                                {key === "extensionDoesNotSupportBestRelease" && "This provider does not support best release search"}
                            </>}
                        />
                    }
                    return null
                })}

                {(selectedProviderExtensionId !== "none" && selectedProviderExtensionId !== "") ? (
                    <>
                        {(searchType === Torrent_SearchType.SMART) &&
                            <AppLayoutStack className="Sea-TorrentSearchContainer__smartSearchContainer" data-torrent-search-smart-search-container>
                                <div
                                    data-torrent-search-smart-search-provider-param-container
                                    className={cn(
                                        "Sea-TorrentSearchContainer__providerParamContainer flex flex-col items-center flex-wrap justify-around gap-3 md:flex-row w-full border rounded-[--radius] p-3",
                                        {
                                            "hidden": !selectedProviderExtension?.settings?.smartSearchFilters?.includes("episodeNumber") &&
                                                !selectedProviderExtension?.settings?.smartSearchFilters?.includes("resolution")
                                                && !selectedProviderExtension?.settings?.smartSearchFilters?.includes("batch")
                                                && !selectedProviderExtension?.settings?.smartSearchFilters?.includes("bestReleases")
                                                && !selectedProviderExtension?.settings?.smartSearchFilters?.includes("search"),
                                        },
                                    )}
                                >

                                    {selectedProviderExtension?.settings?.smartSearchFilters?.includes("episodeNumber") && <NumberInput
                                        data-torrent-search-smart-search-episode-number-input
                                        label="Episode number"
                                        value={smartSearchEpisode}
                                        disabled={entry?.media?.format === "MOVIE" || smartSearchBest}
                                        onValueChange={(value) => {
                                            startTransition(() => {
                                                setSmartSearchEpisode(value)
                                            })
                                        }}
                                        formatOptions={{ useGrouping: false }}
                                        // hideControls
                                        size="sm"
                                        fieldClass={cn(
                                            "flex flex-none w-fit md:justify-end gap-3 space-y-0",
                                            { "opacity-50 cursor-not-allowed pointer-events-none": (smartSearchBatch || searchType != Torrent_SearchType.SMART) },
                                        )}
                                        fieldLabelClass={cn(
                                            "flex-none self-center font-normal !text-md sm:text-md lg:text-md",
                                        )}
                                        className="max-w-[4rem]"
                                    />}

                                    {selectedProviderExtension?.settings?.smartSearchFilters?.includes("resolution") && <Select
                                        data-torrent-search-smart-search-resolution-select
                                        label="Resolution"
                                        value="1080"
                                        onValueChange={() => {}}
                                        options={[
                                            { value: "1080", label: "1080p" },
                                        ]}
                                        disabled={true}
                                        size="sm"
                                        fieldClass={cn(
                                            "flex flex-none w-fit md:justify-center gap-3 space-y-0",
                                            "opacity-50 cursor-not-allowed pointer-events-none",
                                        )}
                                        fieldLabelClass="flex-none self-center font-normal !text-md sm:text-md lg:text-md"
                                        className="w-[6rem]"
                                    />}

                                    {selectedProviderExtension?.settings?.smartSearchFilters?.includes("batch") && <Switch
                                        data-torrent-search-smart-search-batch-switch
                                        label="Batches"
                                        value={smartSearchBatch}
                                        onValueChange={setSmartSearchBatch}
                                        disabled={smartSearchBest || !downloadInfo?.canBatch}
                                        fieldClass={cn(
                                            "flex flex-none w-fit",
                                            { "opacity-50 cursor-not-allowed pointer-events-none": !downloadInfo?.canBatch || smartSearchBest },
                                        )}
                                        size="sm"
                                        containerClass="flex-row-reverse gap-1"
                                    />}

                                    {selectedProviderExtension?.settings?.smartSearchFilters?.includes("bestReleases") && <Switch
                                        data-torrent-search-smart-search-best-releases-switch
                                        label="Best releases"
                                        value={smartSearchBest}
                                        onValueChange={setSmartSearchBest}
                                        fieldClass={cn(
                                            "flex flex-none w-fit",
                                            { "opacity-50 cursor-not-allowed pointer-events-none": !downloadInfo?.canBatch },
                                        )}
                                        size="sm"
                                        containerClass="flex-row-reverse gap-1"
                                    />}

                                </div>

                                {!hasOneWarning && (
                                    <>
                                        {selectedProviderExtension?.settings?.smartSearchFilters?.includes("query") &&
                                            <div className="py-1" data-torrent-search-smart-search-query-input-container>
                                                <DataGridSearchInput
                                                    value={globalFilter ?? ""}
                                                    onChange={v => setGlobalFilter(v)}
                                                    placeholder={searchType === Torrent_SearchType.SMART
                                                        ? `Refine the title (${entry.media?.title?.romaji})`
                                                        : "Search"}
                                                    fieldClass="md:max-w-full w-full"
                                                />
                                            </div>}

                                        <TorrentPreviewList
                                            entry={entry}
                                            previews={previews}
                                            isLoading={isLoading}
                                            selectedTorrents={selectedTorrents}
                                            onToggleTorrent={handleToggleTorrent}
                                                                    type={type}
                                            torrentMetadata={data?.torrentMetadata}
                                            // animeMetadata={data?.animeMetadata}
                                        />
                                    </>
                                )}
                            </AppLayoutStack>}

                        {hasOneWarning && <LuffyError />}

                        {((searchType !== Torrent_SearchType.SMART) && !hasOneWarning && !previews?.length) && (
                            <>
                                <TorrentTable
                                    entry={entry}
                                    torrents={torrents}
                                    globalFilter={globalFilter}
                                    setGlobalFilter={setGlobalFilter}
                                    smartSearch={false}
                                    supportsQuery
                                    isLoading={isLoading}
                                    isFetching={isFetching}
                                    selectedTorrents={selectedTorrents}
                                    onToggleTorrent={handleToggleTorrent}
                                                    animeMetadata={data?.animeMetadata}
                                    torrentMetadata={data?.torrentMetadata}
                                />
                            </>
                        )}

                    </>
                ) : (!!providerExtensions) ? <LuffyError title="No extension selected" /> : <div className="space-y-2">
                    <Skeleton className="h-[96px]" />
                    <Skeleton className="h-[96px]" />
                    <Skeleton className="h-[96px]" />
                    <Skeleton className="h-[96px]" />
                </div>}
            </AppLayoutStack>

            {type === "download" && <TorrentConfirmationModal
                onToggleTorrent={handleToggleTorrent}
                media={entry.media!!}
                entry={entry}
            />}

        </>
    )

}

function TorrentSearchTorrentStreamBatchHistory({ entry, type }: {
    entry: Anime_Entry | undefined,
    type: TorrentSelectionType
}) {
    // Torrent stream batch history functionality has been removed
    return null
}
