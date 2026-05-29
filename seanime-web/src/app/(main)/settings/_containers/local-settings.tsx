import { SettingsCard, SettingsPageHeader } from "@/app/(main)/settings/_components/settings-card"
import { SettingsSubmitButton } from "@/app/(main)/settings/_components/settings-submit-button"
import { Field } from "@/components/ui/form"
import React from "react"
import { LuUserCog } from "react-icons/lu"

type Props = {
    isPending: boolean
    children?: React.ReactNode
}

export function LocalSettings(props: Props) {

    const {
        isPending,
        children,
        ...rest
    } = props

    return (
        <div className="space-y-4">

            <SettingsPageHeader
                title="Local Account"
                description="Local anime and manga list managed by Seanime"
                icon={LuUserCog}
            />

            <SettingsCard
                title="AniList"
            >
                <div>
                    <Field.Switch
                        side="right"
                        name="autoSyncToLocalAccount"
                        label="Auto sync from AniList"
                        help="Periodically update your local collection by using your AniList data."
                    />
                </div>
            </SettingsCard>

            <SettingsSubmitButton isPending={isPending} />

        </div>
    )
}
