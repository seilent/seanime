import { useGetClientMediaSettings, useSaveClientMediaSettings } from "@/api/hooks/mediastream.hooks"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { useMediastreamActiveOnDevice } from "@/app/(main)/mediastream/_lib/mediastream.atoms"
import { SettingsSubmitButton } from "@/app/(main)/settings/_components/settings-submit-button"
import { Checkbox } from "@/components/ui/checkbox"
import { defineSchema, Field, Form } from "@/components/ui/form"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import React from "react"
import { MdOutlineDevices } from "react-icons/md"

const mediastreamSchema = defineSchema(({ z }) => z.object({
    directPlayOnly: z.boolean(),
}))

type MediastreamSettingsProps = {
    children?: React.ReactNode
}

export function MediastreamSettings(props: MediastreamSettingsProps) {

    const {
        children,
        ...rest
    } = props

    const serverStatus = useServerStatus()

    const { data: clientMedia, isLoading: loadingClientMedia } = useGetClientMediaSettings(true)
    const { mutate: saveClientMedia, isPending: savingClientMedia } = useSaveClientMediaSettings()

    const { activeOnDevice, setActiveOnDevice } = useMediastreamActiveOnDevice()

    if (!clientMedia || loadingClientMedia) return <LoadingSpinner />

    return (
        <>
            <Form
                schema={mediastreamSchema}
                onSubmit={data => {
                    // Save client media settings (per-user)
                    saveClientMedia({
                        settings: {
                            directPlayOnly: data.directPlayOnly,
                        },
                    })
                }}
                defaultValues={{
                    directPlayOnly: clientMedia?.directPlayOnly ?? false,
                }}
                stackClass="space-y-6"
            >
                {(f) => (
                    <>
                        <div className="flex gap-4 items-center border rounded-md p-2 lg:p-4">
                            <MdOutlineDevices className="text-4xl" />
                            <div className="space-y-1">
                                <Checkbox
                                    value={activeOnDevice ?? false}
                                    onValueChange={v => setActiveOnDevice((prev) => typeof v === "boolean" ? v : prev)}
                                    label="Use media streaming on this device"
                                    help="Enable this option if you want to use media streaming on this device."
                                />
                                <p className="text-gray-200">
                                    Current client: {serverStatus?.clientDevice}, {serverStatus?.clientPlatform}
                                </p>
                            </div>
                        </div>

                        {activeOnDevice && (
                            <div className="text-gray-300">
                                <p>Your downloaded media files will be played back using the built-in player on this device.</p>
                                <p className="text-sm text-gray-400 mt-2">Note: Transcoding has been disabled for better performance. Most modern browsers support direct playback of common video formats.</p>
                            </div>
                        )}

                        <Field.Switch
                            name="directPlayOnly"
                            label="Direct play only"
                            help="Only allow direct play. This is now the default behavior."
                        />

                        <SettingsSubmitButton isPending={savingClientMedia} />
                    </>
                )}
            </Form>

        </>
    )
}