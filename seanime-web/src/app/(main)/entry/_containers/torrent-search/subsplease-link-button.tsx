import { Anime_Entry } from "@/api/generated/types"
import { useGetSubspleaseShows, useGetSubspleaseStatus, useLinkSubsplease } from "@/api/hooks/subsplease.hooks"
import { AnimeMetaActionButton } from "@/app/(main)/entry/_components/meta-section"
import { Button } from "@/components/ui/button"
import { Combobox } from "@/components/ui/combobox"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { Modal } from "@/components/ui/modal"
import React, { useState } from "react"
import { FiLink } from "react-icons/fi"

export function SubspleaseLinkButton({ entry }: { entry: Anime_Entry }) {

    const [open, setOpen] = useState(false)
    const [slug, setSlug] = useState("")

    const enabled = !!(entry.media?.id && entry.media?.status && entry.media?.format)
    const { data: status } = useGetSubspleaseStatus(entry.media, enabled)
    const linked = !!status?.slug

    const { data: shows, isLoading: showsLoading } = useGetSubspleaseShows(open)

    const { mutate: link, isPending } = useLinkSubsplease(entry.mediaId, () => {
        setOpen(false)
        setSlug("")
    })

    const handleOpen = () => {
        setSlug(status?.slug ?? "")
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
                description="Select the SubsPlease show to sync episodes from."
            >
                <div className="space-y-3">
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
