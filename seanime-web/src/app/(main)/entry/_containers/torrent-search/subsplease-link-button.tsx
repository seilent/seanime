import { Anime_Entry } from "@/api/generated/types"
import { useGetSubspleaseStatus, useLinkSubsplease } from "@/api/hooks/subsplease.hooks"
import { AnimeMetaActionButton } from "@/app/(main)/entry/_components/meta-section"
import { Button } from "@/components/ui/button"
import { Modal } from "@/components/ui/modal"
import { TextInput } from "@/components/ui/text-input"
import React, { useState } from "react"
import { FiLink } from "react-icons/fi"

export function SubspleaseLinkButton({ entry }: { entry: Anime_Entry }) {

    const [open, setOpen] = useState(false)
    const [url, setUrl] = useState("")

    const enabled = !!(entry.media?.id && entry.media?.status && entry.media?.format)
    const { data: status } = useGetSubspleaseStatus(entry.media, enabled)
    const linked = !!status?.slug

    const { mutate: link, isPending } = useLinkSubsplease(entry.mediaId, () => {
        setOpen(false)
        setUrl("")
    })

    const handleOpen = () => {
        setUrl(status?.slug ? `https://subsplease.org/shows/${status.slug}/` : "")
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
                description="Paste the SubsPlease show URL to sync episodes directly, bypassing title matching."
            >
                <div className="space-y-3">
                    <TextInput
                        value={url}
                        onValueChange={setUrl}
                        placeholder="https://subsplease.org/shows/anime-title/"
                    />
                    <Button
                        intent="primary"
                        loading={isPending}
                        disabled={!url.trim()}
                        onClick={() => link({ mediaId: entry.mediaId, url })}
                    >
                        {linked ? "Update" : "Link"}
                    </Button>
                </div>
            </Modal>
        </>
    )
}
