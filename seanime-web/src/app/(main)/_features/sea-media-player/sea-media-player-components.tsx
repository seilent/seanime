// Local definitions replacing deleted onlinestream components
const submenuClass = "vds-menu-content text-sm font-medium outline-none data-[focus]:ring-4 data-[hocus]:bg-white/10 data-[open]:bg-white/10 data-[focus]:ring-media-focus rounded-sm"

interface VdsSubmenuButtonProps {
    label: string
    hint?: string
    icon: React.ComponentType<any>
    disabled?: boolean
}

function VdsSubmenuButton({ label, hint, icon: Icon, disabled }: VdsSubmenuButtonProps) {
    return (
        <Menu.Button
            className="vds-menu-button vds-menu-item flex w-full cursor-pointer select-none items-center justify-start rounded-sm p-2.5 text-left text-sm font-medium outline-none data-[focus]:ring-4 data-[hocus]:bg-white/10 data-[focus]:ring-media-focus aria-hidden:hidden"
            disabled={disabled}
        >
            <Icon className="vds-icon h-5 w-5" />
            <span className="vds-menu-label ml-2">{label}</span>
            {hint && <span className="vds-menu-hint ml-auto text-xs opacity-60">{hint}</span>}
        </Menu.Button>
    )
}
import { Switch } from "@/components/ui/switch"
import { Menu } from "@vidstack/react"
import { useAtom } from "jotai/react"
import React from "react"
import { AiFillPlayCircle } from "react-icons/ai"
import { MdPlaylistPlay } from "react-icons/md"
import { RxSlider } from "react-icons/rx"
import { LuCheck, LuVolume2 } from "react-icons/lu"
import {
    __seaMediaPlayer_audioBoostAtom,
    __seaMediaPlayer_autoNextAtom,
    __seaMediaPlayer_autoPlayAtom,
    __seaMediaPlayer_autoSkipIntroOutroAtom,
    __seaMediaPlayer_discreteControlsAtom,
    seaMediaPlayer_audioBoostOptions,
} from "./sea-media-player.atoms"

export function SeaMediaPlayerPlaybackSubmenu() {

    const [autoPlay, setAutoPlay] = useAtom(__seaMediaPlayer_autoPlayAtom)
    const [autoNext, setAutoNext] = useAtom(__seaMediaPlayer_autoNextAtom)
    const [autoSkipIntroOutro, setAutoSkipIntroOutro] = useAtom(__seaMediaPlayer_autoSkipIntroOutroAtom)
    const [discreteControls, setDiscreteControls] = useAtom(__seaMediaPlayer_discreteControlsAtom)

    const [audioBoost, setAudioBoost] = useAtom(__seaMediaPlayer_audioBoostAtom)
    const currentAudioBoostLabel = seaMediaPlayer_audioBoostOptions.find(o => o.value === audioBoost)?.label ?? "Off"

    return (
        <>
            <Menu.Root>
                <VdsSubmenuButton
                    label={`Audio Boost`}
                    hint={currentAudioBoostLabel}
                    disabled={false}
                    icon={LuVolume2}
                />
                <Menu.Content className={submenuClass}>
                    {seaMediaPlayer_audioBoostOptions.map(option => (
                        <button
                            key={option.value}
                            role="menuitemradio"
                            aria-checked={audioBoost === option.value}
                            className="vds-menu-item flex w-full cursor-pointer select-none items-center justify-between rounded-sm p-2.5 text-left text-sm font-medium outline-none data-[hocus]:bg-white/10 hover:bg-white/10"
                            onClick={() => setAudioBoost(option.value)}
                        >
                            <span>{option.label}</span>
                            {audioBoost === option.value && <LuCheck className="h-4 w-4" />}
                        </button>
                    ))}
                </Menu.Content>
            </Menu.Root>
            <Menu.Root>
                <VdsSubmenuButton
                    label={`Auto Play`}
                    hint={autoPlay ? "On" : "Off"}
                    disabled={false}
                    icon={AiFillPlayCircle}
                />
                <Menu.Content className={submenuClass}>
                    <Switch
                        label="Auto play"
                        fieldClass="py-2 px-2"
                        value={autoPlay}
                        onValueChange={setAutoPlay}
                    />
                </Menu.Content>
            </Menu.Root>
            <Menu.Root>
                <VdsSubmenuButton
                    label={`Auto Play Next Episode`}
                    hint={autoNext ? "On" : "Off"}
                    disabled={false}
                    icon={MdPlaylistPlay}
                />
                <Menu.Content className={submenuClass}>
                    <Switch
                        label="Auto play next episode"
                        fieldClass="py-2 px-2"
                        value={autoNext}
                        onValueChange={setAutoNext}
                    />
                </Menu.Content>
            </Menu.Root>
            <Menu.Root>
                <VdsSubmenuButton
                    label={`Skip Intro/Outro`}
                    hint={autoSkipIntroOutro ? "On" : "Off"}
                    disabled={false}
                    icon={MdPlaylistPlay}
                />
                <Menu.Content className={submenuClass}>
                    <Switch
                        label="Skip intro/outro"
                        fieldClass="py-2 px-2"
                        value={autoSkipIntroOutro}
                        onValueChange={setAutoSkipIntroOutro}
                    />
                </Menu.Content>
            </Menu.Root>
            <Menu.Root>
                <VdsSubmenuButton
                    label={`Discrete Controls`}
                    hint={discreteControls ? "On" : "Off"}
                    disabled={false}
                    icon={RxSlider}
                />
                <Menu.Content className={submenuClass}>
                    <Switch
                        label="Discrete controls"
                        help="Only show the controls when the mouse is over the bottom part. (Large screens only)"
                        fieldClass="py-2 px-2"
                        value={discreteControls}
                        onValueChange={setDiscreteControls}
                        fieldHelpTextClass="max-w-xs"
                    />
                </Menu.Content>
            </Menu.Root>
        </>
    )
}
