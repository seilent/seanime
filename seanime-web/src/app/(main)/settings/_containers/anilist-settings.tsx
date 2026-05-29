import { SettingsPageHeader } from "@/app/(main)/settings/_components/settings-card"
import { SettingsSubmitButton } from "@/app/(main)/settings/_components/settings-submit-button"
import React from "react"
import { SiAnilist } from "react-icons/si"

type Props = {
    isPending: boolean
    children?: React.ReactNode
}

export function AnilistSettings(props: Props) {

    const {
        isPending,
        children,
        ...rest
    } = props

    return (
        <div className="space-y-4">

            <SettingsPageHeader
                title="AniList"
                description="Manage your AniList account"
                icon={SiAnilist}
            />


            <SettingsSubmitButton isPending={isPending} />

        </div>
    )
}
