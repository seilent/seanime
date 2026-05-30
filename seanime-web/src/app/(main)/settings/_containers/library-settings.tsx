import { SettingsCard } from "@/app/(main)/settings/_components/settings-card"
import { SettingsSubmitButton } from "@/app/(main)/settings/_components/settings-submit-button"
import { Field } from "@/components/ui/form"
import React from "react"
import { FcFolder } from "react-icons/fc"

type LibrarySettingsProps = {
    isPending: boolean
}

export function LibrarySettings(props: LibrarySettingsProps) {

    const {
        isPending,
        ...rest
    } = props


    return (
        <div className="space-y-4">

            <SettingsCard>
                <Field.DirectorySelector
                    name="libraryPath"
                    label="Library directory"
                    leftIcon={<FcFolder />}
                    help="Path of the directory where your media files ared located. (Keep the casing consistent)"
                    shouldExist
                />

                <Field.MultiDirectorySelector
                    name="libraryPaths"
                    label="Additional library directories"
                    leftIcon={<FcFolder />}
                    help="Include additional directory paths if your library is spread across multiple locations."
                    shouldExist
                />
            </SettingsCard>

            <SettingsCard>

                <Field.Switch
                    side="right"
                    name="autoScan"
                    label="Automatically refresh library"
                    moreHelp={<p>
                        When adding batches, not all files are guaranteed to be picked up.
                    </p>}
                />

                <Field.Switch
                    side="right"
                    name="refreshLibraryOnStart"
                    label="Refresh library on startup"
                />

            </SettingsCard>

            <SettingsSubmitButton isPending={isPending} />

        </div>
    )
}
