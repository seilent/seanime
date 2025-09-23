import { AL_BaseAnime, Anime_EntryListData, Manga_EntryListData } from "@/api/generated/types"
import { atom } from "jotai/index"
import { useUserScopedAtom } from "./user-scoped-atoms"

// Base atoms for user-specific AniList data
const __anilist_userAnimeMediaBaseAtom = atom<AL_BaseAnime[] | undefined>(undefined)
const __anilist_userAnimeListDataBaseAtom = atom<Record<string, Anime_EntryListData>>({})
const __anilist_userMangaListDataBaseAtom = atom<Record<string, Manga_EntryListData>>({})

// Hooks to use user-scoped AniList atoms
export function useAnilistUserAnimeMedia() {
    return useUserScopedAtom(__anilist_userAnimeMediaBaseAtom)
}

export function useAnilistUserAnimeListData() {
    return useUserScopedAtom(__anilist_userAnimeListDataBaseAtom)
}

export function useAnilistUserMangaListData() {
    return useUserScopedAtom(__anilist_userMangaListDataBaseAtom)
}

// For backward compatibility during migration - these will be removed later
export const __anilist_userAnimeMediaAtom = __anilist_userAnimeMediaBaseAtom
export const __anilist_userAnimeListDataAtom = __anilist_userAnimeListDataBaseAtom
export const __anilist_userMangaListDataAtom = __anilist_userMangaListDataBaseAtom
