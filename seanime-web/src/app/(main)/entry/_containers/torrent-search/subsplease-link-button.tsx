import { Anime_Entry } from "@/api/generated/types"
import { SubspleaseScheduleShow, useGetSubspleaseSchedule, useGetSubspleaseShows, useGetSubspleaseStatus, useLinkSubsplease } from "@/api/hooks/subsplease.hooks"
import { AnimeMetaActionButton } from "@/app/(main)/entry/_components/meta-section"
import { imageShimmer } from "@/components/shared/image-helpers"
import { Button } from "@/components/ui/button"
import { Combobox } from "@/components/ui/combobox"
import { cn } from "@/components/ui/core/styling"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { Modal } from "@/components/ui/modal"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import Image from "next/image"
import React, { useState } from "react"
import { FiLink } from "react-icons/fi"

function ScheduleCard({ show, selected, onSelect }: {
    show: SubspleaseScheduleShow
    selected: boolean
    onSelect: () => void
}) {
    return (
        <button
            type="button"
            aria-pressed={selected}
            className={cn(
                "col-span-1 aspect-[6/7] rounded-[--radius-md] overflow-hidden relative cursor-pointer transition-all group",
                selected && "ring-2 ring-[--brand]",
            )}
            onClick={onSelect}
        >
            <Image
                src={show.imageUrl}
                placeholder={imageShimmer(700, 475)}
                sizes="10rem"
                fill
                alt=""
                className="object-cover object-center"
            />
            <div className="absolute bottom-0 w-full h-[80%] bg-gradient-to-t from-black/80 to-transparent z-[5]" />
            <div className="absolute bottom-0 m-2 z-[10]">
                <p className="line-clamp-2 text-sm font-semibold text-white drop-shadow-lg">
                    {show.title}
                </p>
                <p className="text-xs text-white/70">
                    {show.day} {show.time}
                </p>
            </div>
        </button>
    )
}

export function SubspleaseLinkButton({ entry }: { entry: Anime_Entry }) {

    const [open, setOpen] = useState(false)
    const [slug, setSlug] = useState("")
    const [activeTab, setActiveTab] = useState("airing")

    const enabled = !!(entry.media?.id && entry.media?.status && entry.media?.format)
    const { data: status } = useGetSubspleaseStatus(entry.media, enabled)
    const linked = !!status?.slug

    const { data: schedule, isLoading: scheduleLoading } = useGetSubspleaseSchedule(open)
    const { data: shows, isLoading: showsLoading } = useGetSubspleaseShows(open && activeTab === "all")

    const { mutate: link, isPending } = useLinkSubsplease(entry.mediaId, () => {
        setOpen(false)
        setSlug("")
    })

    const handleOpen = () => {
        setSlug(status?.slug ?? "")
        setActiveTab("airing")
        setOpen(true)
    }

    return (
        <>
            <AnimeMetaActionButton
                intent={linked ? "white-subtle" : "gray-subtle"}
                size="md"
                leftIcon={<FiLink />}
                iconClass="text-2xl"
                onClick={handleOpen}
                data-subsplease-link-button
            >
                {linked ? "Linked to SubsPlease" : "Link SubsPlease"}
            </AnimeMetaActionButton>

            <Modal
                open={open}
                onOpenChange={setOpen}
                title="Link to SubsPlease"
                description="Pick from airing shows or search the full list."
                contentClass="max-w-4xl"
            >
                <div className="space-y-3">
                    <Tabs value={activeTab} onValueChange={setActiveTab}>
                        <TabsList>
                            <TabsTrigger value="airing">Airing</TabsTrigger>
                            <TabsTrigger value="all">All shows</TabsTrigger>
                        </TabsList>
                        <TabsContent value="airing">
                            {scheduleLoading ? (
                                <LoadingSpinner />
                            ) : (schedule && schedule.length > 0) ? (
                                <ScrollArea className="max-h-[60vh]">
                                    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-2">
                                        {schedule.map(show => (
                                            <ScheduleCard
                                                key={show.slug}
                                                show={show}
                                                selected={slug === show.slug}
                                                onSelect={() => setSlug(show.slug)}
                                            />
                                        ))}
                                    </div>
                                </ScrollArea>
                            ) : (
                                <p className="text-[--muted] text-sm py-4">No airing shows found. Use the All shows tab.</p>
                            )}
                        </TabsContent>
                        <TabsContent value="all">
                            {showsLoading ? (
                                <LoadingSpinner />
                            ) : (
                                <Combobox
                                    value={slug ? [slug] : []}
                                    onValueChange={v => setSlug(v[0] ?? "")}
                                    options={(shows ?? []).map(s => ({ value: s.slug, label: s.title, textValue: s.title }))}
                                    emptyMessage="No shows found"
                                    placeholder="Select a show"
                                    disabled={!shows}
                                />
                            )}
                        </TabsContent>
                    </Tabs>
                    <Button
                        intent="primary"
                        loading={isPending}
                        disabled={!slug}
                        onClick={() => link({ mediaId: entry.mediaId, url: slug })}
                    >
                        {linked ? "Update" : "Link"}
                    </Button>
                </div>
            </Modal>
        </>
    )
}
