//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// admin_system_scan
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useStartSystemScan() {
//     return useServerMutation<Scanner_SystemScanResult, StartSystemScan_Variables>({
//         endpoint: API_ENDPOINTS.ADMIN_SYSTEM_SCAN.StartSystemScan.endpoint,
//         method: API_ENDPOINTS.ADMIN_SYSTEM_SCAN.StartSystemScan.methods[0],
//         mutationKey: [API_ENDPOINTS.ADMIN_SYSTEM_SCAN.StartSystemScan.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetSystemScanStatus() {
//     return useServerQuery<Record<string, interface{}>>({
//         endpoint: API_ENDPOINTS.ADMIN_SYSTEM_SCAN.GetSystemScanStatus.endpoint,
//         method: API_ENDPOINTS.ADMIN_SYSTEM_SCAN.GetSystemScanStatus.methods[0],
//         queryKey: [API_ENDPOINTS.ADMIN_SYSTEM_SCAN.GetSystemScanStatus.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// anilist
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetAnimeCollection() {
//     return useServerQuery<AL_AnimeCollection>({
//         endpoint: API_ENDPOINTS.ANILIST.GetAnimeCollection.endpoint,
//         method: API_ENDPOINTS.ANILIST.GetAnimeCollection.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST.GetAnimeCollection.key],
//         enabled: true,
//     })
// }

// export function useGetAnimeCollection() {
//     return useServerMutation<AL_AnimeCollection>({
//         endpoint: API_ENDPOINTS.ANILIST.GetAnimeCollection.endpoint,
//         method: API_ENDPOINTS.ANILIST.GetAnimeCollection.methods[1],
//         mutationKey: [API_ENDPOINTS.ANILIST.GetAnimeCollection.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetRawAnimeCollection() {
//     return useServerQuery<AL_AnimeCollection>({
//         endpoint: API_ENDPOINTS.ANILIST.GetRawAnimeCollection.endpoint,
//         method: API_ENDPOINTS.ANILIST.GetRawAnimeCollection.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST.GetRawAnimeCollection.key],
//         enabled: true,
//     })
// }

// export function useGetRawAnimeCollection() {
//     return useServerMutation<AL_AnimeCollection>({
//         endpoint: API_ENDPOINTS.ANILIST.GetRawAnimeCollection.endpoint,
//         method: API_ENDPOINTS.ANILIST.GetRawAnimeCollection.methods[1],
//         mutationKey: [API_ENDPOINTS.ANILIST.GetRawAnimeCollection.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useEditAnilistListEntry() {
//     return useServerMutation<true, EditAnilistListEntry_Variables>({
//         endpoint: API_ENDPOINTS.ANILIST.EditAnilistListEntry.endpoint,
//         method: API_ENDPOINTS.ANILIST.EditAnilistListEntry.methods[0],
//         mutationKey: [API_ENDPOINTS.ANILIST.EditAnilistListEntry.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetAnilistAnimeDetails(id: number) {
//     return useServerQuery<AL_AnimeDetailsById_Media>({
//         endpoint: API_ENDPOINTS.ANILIST.GetAnilistAnimeDetails.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.ANILIST.GetAnilistAnimeDetails.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST.GetAnilistAnimeDetails.key],
//         enabled: true,
//     })
// }

// export function useGetAnilistStudioDetails(id: number) {
//     return useServerQuery<AL_StudioDetails>({
//         endpoint: API_ENDPOINTS.ANILIST.GetAnilistStudioDetails.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.ANILIST.GetAnilistStudioDetails.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST.GetAnilistStudioDetails.key],
//         enabled: true,
//     })
// }

// export function useDeleteAnilistListEntry() {
//     return useServerMutation<boolean, DeleteAnilistListEntry_Variables>({
//         endpoint: API_ENDPOINTS.ANILIST.DeleteAnilistListEntry.endpoint,
//         method: API_ENDPOINTS.ANILIST.DeleteAnilistListEntry.methods[0],
//         mutationKey: [API_ENDPOINTS.ANILIST.DeleteAnilistListEntry.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useAnilistListAnime() {
//     return useServerMutation<AL_ListAnime, AnilistListAnime_Variables>({
//         endpoint: API_ENDPOINTS.ANILIST.AnilistListAnime.endpoint,
//         method: API_ENDPOINTS.ANILIST.AnilistListAnime.methods[0],
//         mutationKey: [API_ENDPOINTS.ANILIST.AnilistListAnime.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useAnilistListRecentAiringAnime() {
//     return useServerMutation<AL_ListRecentAnime, AnilistListRecentAiringAnime_Variables>({
//         endpoint: API_ENDPOINTS.ANILIST.AnilistListRecentAiringAnime.endpoint,
//         method: API_ENDPOINTS.ANILIST.AnilistListRecentAiringAnime.methods[0],
//         mutationKey: [API_ENDPOINTS.ANILIST.AnilistListRecentAiringAnime.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useAnilistListMissedSequels() {
//     return useServerQuery<Array<AL_BaseAnime>>({
//         endpoint: API_ENDPOINTS.ANILIST.AnilistListMissedSequels.endpoint,
//         method: API_ENDPOINTS.ANILIST.AnilistListMissedSequels.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST.AnilistListMissedSequels.key],
//         enabled: true,
//     })
// }

// export function useGetAniListStats() {
//     return useServerQuery<AL_Stats>({
//         endpoint: API_ENDPOINTS.ANILIST.GetAniListStats.endpoint,
//         method: API_ENDPOINTS.ANILIST.GetAniListStats.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST.GetAniListStats.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// anilist_connection
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useAnilistConnect() {
//     return useServerMutation<Status, AnilistConnect_Variables>({
//         endpoint: API_ENDPOINTS.ANILIST_CONNECTION.AnilistConnect.endpoint,
//         method: API_ENDPOINTS.ANILIST_CONNECTION.AnilistConnect.methods[0],
//         mutationKey: [API_ENDPOINTS.ANILIST_CONNECTION.AnilistConnect.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useAnilistDisconnect() {
//     return useServerMutation<Status>({
//         endpoint: API_ENDPOINTS.ANILIST_CONNECTION.AnilistDisconnect.endpoint,
//         method: API_ENDPOINTS.ANILIST_CONNECTION.AnilistDisconnect.methods[0],
//         mutationKey: [API_ENDPOINTS.ANILIST_CONNECTION.AnilistDisconnect.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useAnilistConnectionStatus() {
//     return useServerQuery<object with connection status and user info>({
//         endpoint: API_ENDPOINTS.ANILIST_CONNECTION.AnilistConnectionStatus.endpoint,
//         method: API_ENDPOINTS.ANILIST_CONNECTION.AnilistConnectionStatus.methods[0],
//         queryKey: [API_ENDPOINTS.ANILIST_CONNECTION.AnilistConnectionStatus.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// anime
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetAnimeEpisodeCollection(id: number) {
//     return useServerQuery<Anime_EpisodeCollection>({
//         endpoint: API_ENDPOINTS.ANIME.GetAnimeEpisodeCollection.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.ANIME.GetAnimeEpisodeCollection.methods[0],
//         queryKey: [API_ENDPOINTS.ANIME.GetAnimeEpisodeCollection.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// anime_collection
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetLibraryCollection() {
//     return useServerQuery<Anime_LibraryCollection>({
//         endpoint: API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.endpoint,
//         method: API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.methods[0],
//         queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.key],
//         enabled: true,
//     })
// }

// export function useGetLibraryCollection() {
//     return useServerMutation<Anime_LibraryCollection>({
//         endpoint: API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.endpoint,
//         method: API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.methods[1],
//         mutationKey: [API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetAnimeCollectionSchedule() {
//     return useServerQuery<Array<Anime_ScheduleItem>>({
//         endpoint: API_ENDPOINTS.ANIME_COLLECTION.GetAnimeCollectionSchedule.endpoint,
//         method: API_ENDPOINTS.ANIME_COLLECTION.GetAnimeCollectionSchedule.methods[0],
//         queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetAnimeCollectionSchedule.key],
//         enabled: true,
//     })
// }

// export function useAddUnknownMedia() {
//     return useServerMutation<AL_AnimeCollection, AddUnknownMedia_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_COLLECTION.AddUnknownMedia.endpoint,
//         method: API_ENDPOINTS.ANIME_COLLECTION.AddUnknownMedia.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_COLLECTION.AddUnknownMedia.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// anime_entries
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetAnimeEntry(id: number) {
//     return useServerQuery<Anime_Entry>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.methods[0],
//         queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntry.key],
//         enabled: true,
//     })
// }

// export function useAnimeEntryBulkAction() {
//     return useServerMutation<Array<Anime_LocalFile>, AnimeEntryBulkAction_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.AnimeEntryBulkAction.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.AnimeEntryBulkAction.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.AnimeEntryBulkAction.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useFetchAnimeEntrySuggestions() {
//     return useServerMutation<Array<AL_BaseAnime>, FetchAnimeEntrySuggestions_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.FetchAnimeEntrySuggestions.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.FetchAnimeEntrySuggestions.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.FetchAnimeEntrySuggestions.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useAnimeEntryManualMatch() {
//     return useServerMutation<Array<Anime_LocalFile>, AnimeEntryManualMatch_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.AnimeEntryManualMatch.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.AnimeEntryManualMatch.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.AnimeEntryManualMatch.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMissingEpisodes() {
//     return useServerQuery<Anime_MissingEpisodes>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.GetMissingEpisodes.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.GetMissingEpisodes.methods[0],
//         queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetMissingEpisodes.key],
//         enabled: true,
//     })
// }

// export function useGetAnimeEntrySilenceStatus(id: number) {
//     return useServerQuery<Models_SilencedMediaEntry>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntrySilenceStatus.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntrySilenceStatus.methods[0],
//         queryKey: [API_ENDPOINTS.ANIME_ENTRIES.GetAnimeEntrySilenceStatus.key],
//         enabled: true,
//     })
// }

// export function useToggleAnimeEntrySilenceStatus() {
//     return useServerMutation<boolean, ToggleAnimeEntrySilenceStatus_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.ToggleAnimeEntrySilenceStatus.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.ToggleAnimeEntrySilenceStatus.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.ToggleAnimeEntrySilenceStatus.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateAnimeEntryProgress() {
//     return useServerMutation<boolean, UpdateAnimeEntryProgress_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.UpdateAnimeEntryProgress.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.UpdateAnimeEntryProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.UpdateAnimeEntryProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateAnimeEntryRepeat() {
//     return useServerMutation<boolean, UpdateAnimeEntryRepeat_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.UpdateAnimeEntryRepeat.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.UpdateAnimeEntryRepeat.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.UpdateAnimeEntryRepeat.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useValidateAnimeEntryLocalFiles() {
//     return useServerMutation<boolean, ValidateAnimeEntryLocalFiles_Variables>({
//         endpoint: API_ENDPOINTS.ANIME_ENTRIES.ValidateAnimeEntryLocalFiles.endpoint,
//         method: API_ENDPOINTS.ANIME_ENTRIES.ValidateAnimeEntryLocalFiles.methods[0],
//         mutationKey: [API_ENDPOINTS.ANIME_ENTRIES.ValidateAnimeEntryLocalFiles.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// auto_downloader
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useRunAutoDownloader() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.RunAutoDownloader.endpoint,
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.RunAutoDownloader.methods[0],
//         mutationKey: [API_ENDPOINTS.AUTO_DOWNLOADER.RunAutoDownloader.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetAutoDownloaderRule(id: number) {
//     return useServerQuery<Anime_AutoDownloaderRule>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRule.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRule.methods[0],
//         queryKey: [API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRule.key],
//         enabled: true,
//     })
// }

// export function useGetAutoDownloaderRulesByAnime(id: number) {
//     return useServerQuery<Array<Anime_AutoDownloaderRule>>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRulesByAnime.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRulesByAnime.methods[0],
//         queryKey: [API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRulesByAnime.key],
//         enabled: true,
//     })
// }

// export function useGetAutoDownloaderRules() {
//     return useServerQuery<Array<Anime_AutoDownloaderRule>>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRules.endpoint,
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRules.methods[0],
//         queryKey: [API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderRules.key],
//         enabled: true,
//     })
// }

// export function useCreateAutoDownloaderRule() {
//     return useServerMutation<Anime_AutoDownloaderRule, CreateAutoDownloaderRule_Variables>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.CreateAutoDownloaderRule.endpoint,
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.CreateAutoDownloaderRule.methods[0],
//         mutationKey: [API_ENDPOINTS.AUTO_DOWNLOADER.CreateAutoDownloaderRule.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateAutoDownloaderRule() {
//     return useServerMutation<Anime_AutoDownloaderRule, UpdateAutoDownloaderRule_Variables>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.UpdateAutoDownloaderRule.endpoint,
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.UpdateAutoDownloaderRule.methods[0],
//         mutationKey: [API_ENDPOINTS.AUTO_DOWNLOADER.UpdateAutoDownloaderRule.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useDeleteAutoDownloaderRule(id: number) {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.DeleteAutoDownloaderRule.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.DeleteAutoDownloaderRule.methods[0],
//         mutationKey: [API_ENDPOINTS.AUTO_DOWNLOADER.DeleteAutoDownloaderRule.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetAutoDownloaderItems() {
//     return useServerQuery<Array<Models_AutoDownloaderItem>>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderItems.endpoint,
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderItems.methods[0],
//         queryKey: [API_ENDPOINTS.AUTO_DOWNLOADER.GetAutoDownloaderItems.key],
//         enabled: true,
//     })
// }

// export function useDeleteAutoDownloaderItem(id: number) {
//     return useServerMutation<boolean, DeleteAutoDownloaderItem_Variables>({
//         endpoint: API_ENDPOINTS.AUTO_DOWNLOADER.DeleteAutoDownloaderItem.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.AUTO_DOWNLOADER.DeleteAutoDownloaderItem.methods[0],
//         mutationKey: [API_ENDPOINTS.AUTO_DOWNLOADER.DeleteAutoDownloaderItem.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// continuity
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useUpdateContinuityWatchHistoryItem() {
//     return useServerMutation<boolean, UpdateContinuityWatchHistoryItem_Variables>({
//         endpoint: API_ENDPOINTS.CONTINUITY.UpdateContinuityWatchHistoryItem.endpoint,
//         method: API_ENDPOINTS.CONTINUITY.UpdateContinuityWatchHistoryItem.methods[0],
//         mutationKey: [API_ENDPOINTS.CONTINUITY.UpdateContinuityWatchHistoryItem.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetContinuityWatchHistoryItem(id: number) {
//     return useServerQuery<Continuity_WatchHistoryItemResponse>({
//         endpoint: API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistoryItem.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistoryItem.methods[0],
//         queryKey: [API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistoryItem.key],
//         enabled: true,
//     })
// }

// export function useGetContinuityWatchHistory() {
//     return useServerQuery<Continuity_WatchHistory>({
//         endpoint: API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistory.endpoint,
//         method: API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistory.methods[0],
//         queryKey: [API_ENDPOINTS.CONTINUITY.GetContinuityWatchHistory.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// directory_selector
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useDirectorySelector() {
//     return useServerMutation<DirectorySelectorResponse, DirectorySelector_Variables>({
//         endpoint: API_ENDPOINTS.DIRECTORY_SELECTOR.DirectorySelector.endpoint,
//         method: API_ENDPOINTS.DIRECTORY_SELECTOR.DirectorySelector.methods[0],
//         mutationKey: [API_ENDPOINTS.DIRECTORY_SELECTOR.DirectorySelector.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// directstream
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useDirectstreamPlayLocalFile() {
//     return useServerMutation<Mediastream_MediaContainer, DirectstreamPlayLocalFile_Variables>({
//         endpoint: API_ENDPOINTS.DIRECTSTREAM.DirectstreamPlayLocalFile.endpoint,
//         method: API_ENDPOINTS.DIRECTSTREAM.DirectstreamPlayLocalFile.methods[0],
//         mutationKey: [API_ENDPOINTS.DIRECTSTREAM.DirectstreamPlayLocalFile.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// discord
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useSetDiscordMangaActivity() {
//     return useServerMutation<boolean, SetDiscordMangaActivity_Variables>({
//         endpoint: API_ENDPOINTS.DISCORD.SetDiscordMangaActivity.endpoint,
//         method: API_ENDPOINTS.DISCORD.SetDiscordMangaActivity.methods[0],
//         mutationKey: [API_ENDPOINTS.DISCORD.SetDiscordMangaActivity.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useSetDiscordLegacyAnimeActivity() {
//     return useServerMutation<boolean, SetDiscordLegacyAnimeActivity_Variables>({
//         endpoint: API_ENDPOINTS.DISCORD.SetDiscordLegacyAnimeActivity.endpoint,
//         method: API_ENDPOINTS.DISCORD.SetDiscordLegacyAnimeActivity.methods[0],
//         mutationKey: [API_ENDPOINTS.DISCORD.SetDiscordLegacyAnimeActivity.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useSetDiscordAnimeActivityWithProgress() {
//     return useServerMutation<boolean, SetDiscordAnimeActivityWithProgress_Variables>({
//         endpoint: API_ENDPOINTS.DISCORD.SetDiscordAnimeActivityWithProgress.endpoint,
//         method: API_ENDPOINTS.DISCORD.SetDiscordAnimeActivityWithProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.DISCORD.SetDiscordAnimeActivityWithProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateDiscordAnimeActivityWithProgress() {
//     return useServerMutation<boolean, UpdateDiscordAnimeActivityWithProgress_Variables>({
//         endpoint: API_ENDPOINTS.DISCORD.UpdateDiscordAnimeActivityWithProgress.endpoint,
//         method: API_ENDPOINTS.DISCORD.UpdateDiscordAnimeActivityWithProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.DISCORD.UpdateDiscordAnimeActivityWithProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useCancelDiscordActivity() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.DISCORD.CancelDiscordActivity.endpoint,
//         method: API_ENDPOINTS.DISCORD.CancelDiscordActivity.methods[0],
//         mutationKey: [API_ENDPOINTS.DISCORD.CancelDiscordActivity.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// docs
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetDocs() {
//     return useServerQuery<Array<ApiDocsGroup>>({
//         endpoint: API_ENDPOINTS.DOCS.GetDocs.endpoint,
//         method: API_ENDPOINTS.DOCS.GetDocs.methods[0],
//         queryKey: [API_ENDPOINTS.DOCS.GetDocs.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// download
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useDownloadTorrentFile() {
//     return useServerMutation<boolean, DownloadTorrentFile_Variables>({
//         endpoint: API_ENDPOINTS.DOWNLOAD.DownloadTorrentFile.endpoint,
//         method: API_ENDPOINTS.DOWNLOAD.DownloadTorrentFile.methods[0],
//         mutationKey: [API_ENDPOINTS.DOWNLOAD.DownloadTorrentFile.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useDownloadRelease() {
//     return useServerMutation<DownloadReleaseResponse, DownloadRelease_Variables>({
//         endpoint: API_ENDPOINTS.DOWNLOAD.DownloadRelease.endpoint,
//         method: API_ENDPOINTS.DOWNLOAD.DownloadRelease.methods[0],
//         mutationKey: [API_ENDPOINTS.DOWNLOAD.DownloadRelease.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// extensions
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useFetchExternalExtensionData() {
//     return useServerMutation<Extension_Extension, FetchExternalExtensionData_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.FetchExternalExtensionData.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.FetchExternalExtensionData.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.FetchExternalExtensionData.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useInstallExternalExtension() {
//     return useServerMutation<ExtensionRepo_ExtensionInstallResponse, InstallExternalExtension_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.InstallExternalExtension.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.InstallExternalExtension.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.InstallExternalExtension.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUninstallExternalExtension() {
//     return useServerMutation<boolean, UninstallExternalExtension_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.UninstallExternalExtension.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.UninstallExternalExtension.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.UninstallExternalExtension.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateExtensionCode() {
//     return useServerMutation<boolean, UpdateExtensionCode_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.UpdateExtensionCode.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.UpdateExtensionCode.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.UpdateExtensionCode.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useReloadExternalExtensions() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ReloadExternalExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ReloadExternalExtensions.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.ReloadExternalExtensions.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useReloadExternalExtension() {
//     return useServerMutation<boolean, ReloadExternalExtension_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ReloadExternalExtension.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ReloadExternalExtension.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.ReloadExternalExtension.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useListExtensionData() {
//     return useServerQuery<Array<Extension_Extension>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ListExtensionData.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ListExtensionData.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.ListExtensionData.key],
//         enabled: true,
//     })
// }

// export function useGetExtensionPayload() {
//     return useServerQuery<string>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GetExtensionPayload.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GetExtensionPayload.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.GetExtensionPayload.key],
//         enabled: true,
//     })
// }

// export function useListDevelopmentModeExtensions() {
//     return useServerQuery<Array<Extension_Extension>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ListDevelopmentModeExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ListDevelopmentModeExtensions.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.ListDevelopmentModeExtensions.key],
//         enabled: true,
//     })
// }

// export function useGetAllExtensions() {
//     return useServerMutation<ExtensionRepo_AllExtensions, GetAllExtensions_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GetAllExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GetAllExtensions.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.GetAllExtensions.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetExtensionUpdateData() {
//     return useServerQuery<Array<ExtensionRepo_UpdateData>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GetExtensionUpdateData.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GetExtensionUpdateData.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.GetExtensionUpdateData.key],
//         enabled: true,
//     })
// }

// export function useListMangaProviderExtensions() {
//     return useServerQuery<Array<ExtensionRepo_MangaProviderExtensionItem>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ListMangaProviderExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ListMangaProviderExtensions.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.ListMangaProviderExtensions.key],
//         enabled: true,
//     })
// }

// export function useListOnlinestreamProviderExtensions() {
//     return useServerQuery<Array<ExtensionRepo_OnlinestreamProviderExtensionItem>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ListOnlinestreamProviderExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ListOnlinestreamProviderExtensions.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.ListOnlinestreamProviderExtensions.key],
//         enabled: true,
//     })
// }

// export function useListAnimeTorrentProviderExtensions() {
//     return useServerQuery<Array<ExtensionRepo_AnimeTorrentProviderExtensionItem>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.ListAnimeTorrentProviderExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.ListAnimeTorrentProviderExtensions.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.ListAnimeTorrentProviderExtensions.key],
//         enabled: true,
//     })
// }

// export function useGetPluginSettings() {
//     return useServerQuery<ExtensionRepo_StoredPluginSettingsData>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GetPluginSettings.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GetPluginSettings.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.GetPluginSettings.key],
//         enabled: true,
//     })
// }

// export function useSetPluginSettingsPinnedTrays() {
//     return useServerMutation<boolean, SetPluginSettingsPinnedTrays_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.SetPluginSettingsPinnedTrays.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.SetPluginSettingsPinnedTrays.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.SetPluginSettingsPinnedTrays.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGrantPluginPermissions() {
//     return useServerMutation<boolean, GrantPluginPermissions_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GrantPluginPermissions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GrantPluginPermissions.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.GrantPluginPermissions.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRunExtensionPlaygroundCode() {
//     return useServerMutation<RunPlaygroundCodeResponse, RunExtensionPlaygroundCode_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.RunExtensionPlaygroundCode.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.RunExtensionPlaygroundCode.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.RunExtensionPlaygroundCode.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetExtensionUserConfig() {
//     return useServerQuery<ExtensionRepo_ExtensionUserConfig>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GetExtensionUserConfig.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GetExtensionUserConfig.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.GetExtensionUserConfig.key],
//         enabled: true,
//     })
// }

// export function useSaveExtensionUserConfig() {
//     return useServerMutation<boolean, SaveExtensionUserConfig_Variables>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.SaveExtensionUserConfig.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.SaveExtensionUserConfig.methods[0],
//         mutationKey: [API_ENDPOINTS.EXTENSIONS.SaveExtensionUserConfig.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMarketplaceExtensions() {
//     return useServerQuery<Array<Extension_Extension>>({
//         endpoint: API_ENDPOINTS.EXTENSIONS.GetMarketplaceExtensions.endpoint,
//         method: API_ENDPOINTS.EXTENSIONS.GetMarketplaceExtensions.methods[0],
//         queryKey: [API_ENDPOINTS.EXTENSIONS.GetMarketplaceExtensions.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// filecache
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetFileCacheTotalSize() {
//     return useServerQuery<string>({
//         endpoint: API_ENDPOINTS.FILECACHE.GetFileCacheTotalSize.endpoint,
//         method: API_ENDPOINTS.FILECACHE.GetFileCacheTotalSize.methods[0],
//         queryKey: [API_ENDPOINTS.FILECACHE.GetFileCacheTotalSize.key],
//         enabled: true,
//     })
// }

// export function useRemoveFileCacheBucket() {
//     return useServerMutation<boolean, RemoveFileCacheBucket_Variables>({
//         endpoint: API_ENDPOINTS.FILECACHE.RemoveFileCacheBucket.endpoint,
//         method: API_ENDPOINTS.FILECACHE.RemoveFileCacheBucket.methods[0],
//         mutationKey: [API_ENDPOINTS.FILECACHE.RemoveFileCacheBucket.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetFileCacheMediastreamVideoFilesTotalSize() {
//     return useServerQuery<string>({
//         endpoint: API_ENDPOINTS.FILECACHE.GetFileCacheMediastreamVideoFilesTotalSize.endpoint,
//         method: API_ENDPOINTS.FILECACHE.GetFileCacheMediastreamVideoFilesTotalSize.methods[0],
//         queryKey: [API_ENDPOINTS.FILECACHE.GetFileCacheMediastreamVideoFilesTotalSize.key],
//         enabled: true,
//     })
// }

// export function useClearFileCacheMediastreamVideoFiles() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.FILECACHE.ClearFileCacheMediastreamVideoFiles.endpoint,
//         method: API_ENDPOINTS.FILECACHE.ClearFileCacheMediastreamVideoFiles.methods[0],
//         mutationKey: [API_ENDPOINTS.FILECACHE.ClearFileCacheMediastreamVideoFiles.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// global_mapping
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetUnmappedFiles() {
//     return useServerQuery<Array<Models_UnmappedFile>>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.methods[0],
//         queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUnmappedFiles.key],
//         enabled: true,
//     })
// }

// export function useGetIgnoredFiles() {
//     return useServerQuery<Array<Models_UnmappedFile>>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.methods[0],
//         queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetIgnoredFiles.key],
//         enabled: true,
//     })
// }

// export function useGetGlobalMappings() {
//     return useServerQuery<Array<Models_GlobalAnimeFileMapping>>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.methods[0],
//         queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetGlobalMappings.key],
//         enabled: true,
//     })
// }

// export function useMapFileToAniList() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.MapFileToAniList.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.MapFileToAniList.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.MapFileToAniList.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useIgnoreFile() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.IgnoreFile.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.IgnoreFile.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.IgnoreFile.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUnignoreFile() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.UnignoreFile.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.UnignoreFile.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.UnignoreFile.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRemoveMapping() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.RemoveMapping.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.RemoveMapping.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.RemoveMapping.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetProgressSyncStats() {
//     return useServerQuery<Record<string, number>>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.methods[0],
//         queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetProgressSyncStats.key],
//         enabled: true,
//     })
// }

// export function useRetryFailedSyncItems() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.RetryFailedSyncItems.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.RetryFailedSyncItems.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.RetryFailedSyncItems.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetFilesForAnime() {
//     return useServerQuery<Array<string>>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetFilesForAnime.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.GetFilesForAnime.methods[0],
//         queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetFilesForAnime.key],
//         enabled: true,
//     })
// }

// export function useGetUserSubscriptions() {
//     return useServerQuery<GetUserSubscriptionsResponse>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.methods[0],
//         queryKey: [API_ENDPOINTS.GLOBAL_MAPPING.GetUserSubscriptions.key],
//         enabled: true,
//     })
// }

// export function useSubscribeToAnime() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.SubscribeToAnime.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.SubscribeToAnime.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.SubscribeToAnime.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUnsubscribeFromAnime() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.GLOBAL_MAPPING.UnsubscribeFromAnime.endpoint,
//         method: API_ENDPOINTS.GLOBAL_MAPPING.UnsubscribeFromAnime.methods[0],
//         mutationKey: [API_ENDPOINTS.GLOBAL_MAPPING.UnsubscribeFromAnime.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// localfiles
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetLocalFiles() {
//     return useServerQuery<Array<Anime_LocalFile>>({
//         endpoint: API_ENDPOINTS.LOCALFILES.GetLocalFiles.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.GetLocalFiles.methods[0],
//         queryKey: [API_ENDPOINTS.LOCALFILES.GetLocalFiles.key],
//         enabled: true,
//     })
// }

// export function useImportLocalFiles() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.LOCALFILES.ImportLocalFiles.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.ImportLocalFiles.methods[0],
//         mutationKey: [API_ENDPOINTS.LOCALFILES.ImportLocalFiles.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useLocalFileBulkAction() {
//     return useServerMutation<Array<Anime_LocalFile>, LocalFileBulkAction_Variables>({
//         endpoint: API_ENDPOINTS.LOCALFILES.LocalFileBulkAction.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.LocalFileBulkAction.methods[0],
//         mutationKey: [API_ENDPOINTS.LOCALFILES.LocalFileBulkAction.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateLocalFileData() {
//     return useServerMutation<Array<Anime_LocalFile>, UpdateLocalFileData_Variables>({
//         endpoint: API_ENDPOINTS.LOCALFILES.UpdateLocalFileData.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.UpdateLocalFileData.methods[0],
//         mutationKey: [API_ENDPOINTS.LOCALFILES.UpdateLocalFileData.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateLocalFiles() {
//     return useServerMutation<boolean, UpdateLocalFiles_Variables>({
//         endpoint: API_ENDPOINTS.LOCALFILES.UpdateLocalFiles.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.UpdateLocalFiles.methods[0],
//         mutationKey: [API_ENDPOINTS.LOCALFILES.UpdateLocalFiles.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useDeleteLocalFiles() {
//     return useServerMutation<boolean, DeleteLocalFiles_Variables>({
//         endpoint: API_ENDPOINTS.LOCALFILES.DeleteLocalFiles.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.DeleteLocalFiles.methods[0],
//         mutationKey: [API_ENDPOINTS.LOCALFILES.DeleteLocalFiles.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRemoveEmptyDirectories() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.LOCALFILES.RemoveEmptyDirectories.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.RemoveEmptyDirectories.methods[0],
//         mutationKey: [API_ENDPOINTS.LOCALFILES.RemoveEmptyDirectories.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMediaAvailability() {
//     return useServerQuery<Array<Anime_LocalFile>>({
//         endpoint: API_ENDPOINTS.LOCALFILES.GetMediaAvailability.endpoint,
//         method: API_ENDPOINTS.LOCALFILES.GetMediaAvailability.methods[0],
//         queryKey: [API_ENDPOINTS.LOCALFILES.GetMediaAvailability.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// mal
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useMALAuth() {
//     return useServerMutation<MalAuthResponse, MALAuth_Variables>({
//         endpoint: API_ENDPOINTS.MAL.MALAuth.endpoint,
//         method: API_ENDPOINTS.MAL.MALAuth.methods[0],
//         mutationKey: [API_ENDPOINTS.MAL.MALAuth.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useEditMALListEntryProgress() {
//     return useServerMutation<boolean, EditMALListEntryProgress_Variables>({
//         endpoint: API_ENDPOINTS.MAL.EditMALListEntryProgress.endpoint,
//         method: API_ENDPOINTS.MAL.EditMALListEntryProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.MAL.EditMALListEntryProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useMALLogout() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.MAL.MALLogout.endpoint,
//         method: API_ENDPOINTS.MAL.MALLogout.methods[0],
//         mutationKey: [API_ENDPOINTS.MAL.MALLogout.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// manga
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetAnilistMangaCollection() {
//     return useServerQuery<AL_MangaCollection, GetAnilistMangaCollection_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.GetAnilistMangaCollection.endpoint,
//         method: API_ENDPOINTS.MANGA.GetAnilistMangaCollection.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetAnilistMangaCollection.key],
//         enabled: true,
//     })
// }

// export function useGetRawAnilistMangaCollection() {
//     return useServerQuery<AL_MangaCollection>({
//         endpoint: API_ENDPOINTS.MANGA.GetRawAnilistMangaCollection.endpoint,
//         method: API_ENDPOINTS.MANGA.GetRawAnilistMangaCollection.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetRawAnilistMangaCollection.key],
//         enabled: true,
//     })
// }

// export function useGetRawAnilistMangaCollection() {
//     return useServerMutation<AL_MangaCollection>({
//         endpoint: API_ENDPOINTS.MANGA.GetRawAnilistMangaCollection.endpoint,
//         method: API_ENDPOINTS.MANGA.GetRawAnilistMangaCollection.methods[1],
//         mutationKey: [API_ENDPOINTS.MANGA.GetRawAnilistMangaCollection.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaCollection() {
//     return useServerQuery<Manga_Collection>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaCollection.endpoint,
//         method: API_ENDPOINTS.MANGA.GetMangaCollection.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetMangaCollection.key],
//         enabled: true,
//     })
// }

// export function useGetMangaEntry(id: number) {
//     return useServerQuery<Manga_Entry>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaEntry.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.MANGA.GetMangaEntry.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetMangaEntry.key],
//         enabled: true,
//     })
// }

// export function useGetMangaEntryDetails(id: number) {
//     return useServerQuery<AL_MangaDetailsById_Media>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaEntryDetails.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.MANGA.GetMangaEntryDetails.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetMangaEntryDetails.key],
//         enabled: true,
//     })
// }

// export function useGetMangaLatestChapterNumbersMap() {
//     return useServerQuery<Record<number, Array<Manga_MangaLatestChapterNumberItem>>>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaLatestChapterNumbersMap.endpoint,
//         method: API_ENDPOINTS.MANGA.GetMangaLatestChapterNumbersMap.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetMangaLatestChapterNumbersMap.key],
//         enabled: true,
//     })
// }

// export function useRefetchMangaChapterContainers() {
//     return useServerMutation<boolean, RefetchMangaChapterContainers_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.RefetchMangaChapterContainers.endpoint,
//         method: API_ENDPOINTS.MANGA.RefetchMangaChapterContainers.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.RefetchMangaChapterContainers.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useEmptyMangaEntryCache() {
//     return useServerMutation<boolean, EmptyMangaEntryCache_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.EmptyMangaEntryCache.endpoint,
//         method: API_ENDPOINTS.MANGA.EmptyMangaEntryCache.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.EmptyMangaEntryCache.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaEntryChapters() {
//     return useServerMutation<Manga_ChapterContainer, GetMangaEntryChapters_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaEntryChapters.endpoint,
//         method: API_ENDPOINTS.MANGA.GetMangaEntryChapters.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.GetMangaEntryChapters.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaEntryPages() {
//     return useServerMutation<Manga_PageContainer, GetMangaEntryPages_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaEntryPages.endpoint,
//         method: API_ENDPOINTS.MANGA.GetMangaEntryPages.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.GetMangaEntryPages.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaEntryDownloadedChapters(id: number) {
//     return useServerQuery<Array<Manga_ChapterContainer>>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaEntryDownloadedChapters.endpoint.replace("{id}", String(id)),
//         method: API_ENDPOINTS.MANGA.GetMangaEntryDownloadedChapters.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetMangaEntryDownloadedChapters.key],
//         enabled: true,
//     })
// }

// export function useAnilistListManga() {
//     return useServerMutation<AL_ListManga, AnilistListManga_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.AnilistListManga.endpoint,
//         method: API_ENDPOINTS.MANGA.AnilistListManga.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.AnilistListManga.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateMangaProgress() {
//     return useServerMutation<boolean, UpdateMangaProgress_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.UpdateMangaProgress.endpoint,
//         method: API_ENDPOINTS.MANGA.UpdateMangaProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.UpdateMangaProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useMangaManualSearch() {
//     return useServerMutation<Array<HibikeManga_SearchResult>, MangaManualSearch_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.MangaManualSearch.endpoint,
//         method: API_ENDPOINTS.MANGA.MangaManualSearch.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.MangaManualSearch.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useMangaManualMapping() {
//     return useServerMutation<boolean, MangaManualMapping_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.MangaManualMapping.endpoint,
//         method: API_ENDPOINTS.MANGA.MangaManualMapping.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.MangaManualMapping.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaMapping() {
//     return useServerMutation<Manga_MappingResponse, GetMangaMapping_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.GetMangaMapping.endpoint,
//         method: API_ENDPOINTS.MANGA.GetMangaMapping.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.GetMangaMapping.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRemoveMangaMapping() {
//     return useServerMutation<boolean, RemoveMangaMapping_Variables>({
//         endpoint: API_ENDPOINTS.MANGA.RemoveMangaMapping.endpoint,
//         method: API_ENDPOINTS.MANGA.RemoveMangaMapping.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA.RemoveMangaMapping.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetLocalMangaPage() {
//     return useServerQuery<Manga_PageContainer>({
//         endpoint: API_ENDPOINTS.MANGA.GetLocalMangaPage.endpoint,
//         method: API_ENDPOINTS.MANGA.GetLocalMangaPage.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA.GetLocalMangaPage.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// manga_download
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useDownloadMangaChapters() {
//     return useServerMutation<boolean, DownloadMangaChapters_Variables>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.DownloadMangaChapters.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.DownloadMangaChapters.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.DownloadMangaChapters.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaDownloadData() {
//     return useServerMutation<Manga_MediaDownloadData, GetMangaDownloadData_Variables>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadData.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadData.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadData.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaDownloadQueue() {
//     return useServerQuery<Array<Models_ChapterDownloadQueueItem>>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadQueue.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadQueue.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadQueue.key],
//         enabled: true,
//     })
// }

// export function useStartMangaDownloadQueue() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.StartMangaDownloadQueue.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.StartMangaDownloadQueue.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.StartMangaDownloadQueue.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useStopMangaDownloadQueue() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.StopMangaDownloadQueue.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.StopMangaDownloadQueue.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.StopMangaDownloadQueue.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useClearAllChapterDownloadQueue() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.ClearAllChapterDownloadQueue.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.ClearAllChapterDownloadQueue.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.ClearAllChapterDownloadQueue.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useResetErroredChapterDownloadQueue() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.ResetErroredChapterDownloadQueue.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.ResetErroredChapterDownloadQueue.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.ResetErroredChapterDownloadQueue.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useDeleteMangaDownloadedChapters() {
//     return useServerMutation<boolean, DeleteMangaDownloadedChapters_Variables>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.DeleteMangaDownloadedChapters.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.DeleteMangaDownloadedChapters.methods[0],
//         mutationKey: [API_ENDPOINTS.MANGA_DOWNLOAD.DeleteMangaDownloadedChapters.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMangaDownloadsList() {
//     return useServerQuery<Array<Manga_DownloadListItem>>({
//         endpoint: API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadsList.endpoint,
//         method: API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadsList.methods[0],
//         queryKey: [API_ENDPOINTS.MANGA_DOWNLOAD.GetMangaDownloadsList.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// manual_dump
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useTestDump() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.MANUAL_DUMP.TestDump.endpoint,
//         method: API_ENDPOINTS.MANUAL_DUMP.TestDump.methods[0],
//         mutationKey: [API_ENDPOINTS.MANUAL_DUMP.TestDump.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// mediastream
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetClientMediaSettings() {
//     return useServerQuery<Models_ClientMediaSettings>({
//         endpoint: API_ENDPOINTS.MEDIASTREAM.GetClientMediaSettings.endpoint,
//         method: API_ENDPOINTS.MEDIASTREAM.GetClientMediaSettings.methods[0],
//         queryKey: [API_ENDPOINTS.MEDIASTREAM.GetClientMediaSettings.key],
//         enabled: true,
//     })
// }

// export function useSaveClientMediaSettings() {
//     return useServerMutation<Models_ClientMediaSettings, SaveClientMediaSettings_Variables>({
//         endpoint: API_ENDPOINTS.MEDIASTREAM.SaveClientMediaSettings.endpoint,
//         method: API_ENDPOINTS.MEDIASTREAM.SaveClientMediaSettings.methods[0],
//         mutationKey: [API_ENDPOINTS.MEDIASTREAM.SaveClientMediaSettings.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRequestMediastreamMediaContainer() {
//     return useServerMutation<Mediastream_MediaContainer, RequestMediastreamMediaContainer_Variables>({
//         endpoint: API_ENDPOINTS.MEDIASTREAM.RequestMediastreamMediaContainer.endpoint,
//         method: API_ENDPOINTS.MEDIASTREAM.RequestMediastreamMediaContainer.methods[0],
//         mutationKey: [API_ENDPOINTS.MEDIASTREAM.RequestMediastreamMediaContainer.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function usePreloadMediastreamMediaContainer() {
//     return useServerMutation<boolean, PreloadMediastreamMediaContainer_Variables>({
//         endpoint: API_ENDPOINTS.MEDIASTREAM.PreloadMediastreamMediaContainer.endpoint,
//         method: API_ENDPOINTS.MEDIASTREAM.PreloadMediastreamMediaContainer.methods[0],
//         mutationKey: [API_ENDPOINTS.MEDIASTREAM.PreloadMediastreamMediaContainer.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// metadata
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function usePopulateFillerData() {
//     return useServerMutation<true, PopulateFillerData_Variables>({
//         endpoint: API_ENDPOINTS.METADATA.PopulateFillerData.endpoint,
//         method: API_ENDPOINTS.METADATA.PopulateFillerData.methods[0],
//         mutationKey: [API_ENDPOINTS.METADATA.PopulateFillerData.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRemoveFillerData() {
//     return useServerMutation<boolean, RemoveFillerData_Variables>({
//         endpoint: API_ENDPOINTS.METADATA.RemoveFillerData.endpoint,
//         method: API_ENDPOINTS.METADATA.RemoveFillerData.methods[0],
//         mutationKey: [API_ENDPOINTS.METADATA.RemoveFillerData.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// playback_manager
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function usePlaybackSyncCurrentProgress() {
//     return useServerMutation<number>({
//         endpoint: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackSyncCurrentProgress.endpoint,
//         method: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackSyncCurrentProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackSyncCurrentProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function usePlaybackGetNextEpisode() {
//     return useServerQuery<Anime_LocalFile>({
//         endpoint: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackGetNextEpisode.endpoint,
//         method: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackGetNextEpisode.methods[0],
//         queryKey: [API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackGetNextEpisode.key],
//         enabled: true,
//     })
// }

// export function usePlaybackStartManualTracking() {
//     return useServerMutation<boolean, PlaybackStartManualTracking_Variables>({
//         endpoint: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackStartManualTracking.endpoint,
//         method: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackStartManualTracking.methods[0],
//         mutationKey: [API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackStartManualTracking.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function usePlaybackCancelManualTracking() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackCancelManualTracking.endpoint,
//         method: API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackCancelManualTracking.methods[0],
//         mutationKey: [API_ENDPOINTS.PLAYBACK_MANAGER.PlaybackCancelManualTracking.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// releases
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useInstallLatestUpdate() {
//     return useServerMutation<Status, InstallLatestUpdate_Variables>({
//         endpoint: API_ENDPOINTS.RELEASES.InstallLatestUpdate.endpoint,
//         method: API_ENDPOINTS.RELEASES.InstallLatestUpdate.methods[0],
//         mutationKey: [API_ENDPOINTS.RELEASES.InstallLatestUpdate.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetLatestUpdate() {
//     return useServerQuery<Updater_Update>({
//         endpoint: API_ENDPOINTS.RELEASES.GetLatestUpdate.endpoint,
//         method: API_ENDPOINTS.RELEASES.GetLatestUpdate.methods[0],
//         queryKey: [API_ENDPOINTS.RELEASES.GetLatestUpdate.key],
//         enabled: true,
//     })
// }

// export function useGetChangelog() {
//     return useServerQuery<string>({
//         endpoint: API_ENDPOINTS.RELEASES.GetChangelog.endpoint,
//         method: API_ENDPOINTS.RELEASES.GetChangelog.methods[0],
//         queryKey: [API_ENDPOINTS.RELEASES.GetChangelog.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// report
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useSaveIssueReport() {
//     return useServerMutation<boolean, SaveIssueReport_Variables>({
//         endpoint: API_ENDPOINTS.REPORT.SaveIssueReport.endpoint,
//         method: API_ENDPOINTS.REPORT.SaveIssueReport.methods[0],
//         mutationKey: [API_ENDPOINTS.REPORT.SaveIssueReport.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useDownloadIssueReport() {
//     return useServerQuery<Report_IssueReport>({
//         endpoint: API_ENDPOINTS.REPORT.DownloadIssueReport.endpoint,
//         method: API_ENDPOINTS.REPORT.DownloadIssueReport.methods[0],
//         queryKey: [API_ENDPOINTS.REPORT.DownloadIssueReport.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// scan
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useScanLocalFiles() {
//     return useServerMutation<Array<Anime_LocalFile>, ScanLocalFiles_Variables>({
//         endpoint: API_ENDPOINTS.SCAN.ScanLocalFiles.endpoint,
//         method: API_ENDPOINTS.SCAN.ScanLocalFiles.methods[0],
//         mutationKey: [API_ENDPOINTS.SCAN.ScanLocalFiles.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// scan_summary
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetScanSummaries() {
//     return useServerQuery<Array<Summary_ScanSummaryItem>>({
//         endpoint: API_ENDPOINTS.SCAN_SUMMARY.GetScanSummaries.endpoint,
//         method: API_ENDPOINTS.SCAN_SUMMARY.GetScanSummaries.methods[0],
//         queryKey: [API_ENDPOINTS.SCAN_SUMMARY.GetScanSummaries.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// settings
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetSettings() {
//     return useServerQuery<Models_Settings>({
//         endpoint: API_ENDPOINTS.SETTINGS.GetSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.GetSettings.methods[0],
//         queryKey: [API_ENDPOINTS.SETTINGS.GetSettings.key],
//         enabled: true,
//     })
// }

// export function useGetGlobalSettings() {
//     return useServerQuery<Models_GlobalSettings>({
//         endpoint: API_ENDPOINTS.SETTINGS.GetGlobalSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.GetGlobalSettings.methods[0],
//         queryKey: [API_ENDPOINTS.SETTINGS.GetGlobalSettings.key],
//         enabled: true,
//     })
// }

// export function useUpdateGlobalSettings() {
//     return useServerMutation<Models_GlobalSettings>({
//         endpoint: API_ENDPOINTS.SETTINGS.UpdateGlobalSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.UpdateGlobalSettings.methods[0],
//         mutationKey: [API_ENDPOINTS.SETTINGS.UpdateGlobalSettings.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetUserSettings() {
//     return useServerQuery<Models_Settings>({
//         endpoint: API_ENDPOINTS.SETTINGS.GetUserSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.GetUserSettings.methods[0],
//         queryKey: [API_ENDPOINTS.SETTINGS.GetUserSettings.key],
//         enabled: true,
//     })
// }

// export function useUpdateUserSettings() {
//     return useServerMutation<Models_Settings>({
//         endpoint: API_ENDPOINTS.SETTINGS.UpdateUserSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.UpdateUserSettings.methods[0],
//         mutationKey: [API_ENDPOINTS.SETTINGS.UpdateUserSettings.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGettingStarted() {
//     return useServerMutation<Status, GettingStarted_Variables>({
//         endpoint: API_ENDPOINTS.SETTINGS.GettingStarted.endpoint,
//         method: API_ENDPOINTS.SETTINGS.GettingStarted.methods[0],
//         mutationKey: [API_ENDPOINTS.SETTINGS.GettingStarted.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useSaveSettings() {
//     return useServerMutation<Status, SaveSettings_Variables>({
//         endpoint: API_ENDPOINTS.SETTINGS.SaveSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.SaveSettings.methods[0],
//         mutationKey: [API_ENDPOINTS.SETTINGS.SaveSettings.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useTriggerLibraryScan() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SETTINGS.TriggerLibraryScan.endpoint,
//         method: API_ENDPOINTS.SETTINGS.TriggerLibraryScan.methods[0],
//         mutationKey: [API_ENDPOINTS.SETTINGS.TriggerLibraryScan.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useSaveAutoDownloaderSettings() {
//     return useServerMutation<boolean, SaveAutoDownloaderSettings_Variables>({
//         endpoint: API_ENDPOINTS.SETTINGS.SaveAutoDownloaderSettings.endpoint,
//         method: API_ENDPOINTS.SETTINGS.SaveAutoDownloaderSettings.methods[0],
//         mutationKey: [API_ENDPOINTS.SETTINGS.SaveAutoDownloaderSettings.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// sse_polling
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useSSEClientEvents() {
//     return useServerQuery<boolean>({
//         endpoint: API_ENDPOINTS.SSE_POLLING.SSEClientEvents.endpoint,
//         method: API_ENDPOINTS.SSE_POLLING.SSEClientEvents.methods[0],
//         queryKey: [API_ENDPOINTS.SSE_POLLING.SSEClientEvents.key],
//         enabled: true,
//     })
// }

// export function useSSENativePlayerEvents() {
//     return useServerQuery<boolean>({
//         endpoint: API_ENDPOINTS.SSE_POLLING.SSENativePlayerEvents.endpoint,
//         method: API_ENDPOINTS.SSE_POLLING.SSENativePlayerEvents.methods[0],
//         queryKey: [API_ENDPOINTS.SSE_POLLING.SSENativePlayerEvents.key],
//         enabled: true,
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// status
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetStatus() {
//     return useServerQuery<Status>({
//         endpoint: API_ENDPOINTS.STATUS.GetStatus.endpoint,
//         method: API_ENDPOINTS.STATUS.GetStatus.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetStatus.key],
//         enabled: true,
//     })
// }

// export function useGetLogFilenames() {
//     return useServerQuery<Array<string>>({
//         endpoint: API_ENDPOINTS.STATUS.GetLogFilenames.endpoint,
//         method: API_ENDPOINTS.STATUS.GetLogFilenames.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetLogFilenames.key],
//         enabled: true,
//     })
// }

// export function useDeleteLogs() {
//     return useServerMutation<boolean, DeleteLogs_Variables>({
//         endpoint: API_ENDPOINTS.STATUS.DeleteLogs.endpoint,
//         method: API_ENDPOINTS.STATUS.DeleteLogs.methods[0],
//         mutationKey: [API_ENDPOINTS.STATUS.DeleteLogs.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetLatestLogContent() {
//     return useServerQuery<string>({
//         endpoint: API_ENDPOINTS.STATUS.GetLatestLogContent.endpoint,
//         method: API_ENDPOINTS.STATUS.GetLatestLogContent.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetLatestLogContent.key],
//         enabled: true,
//     })
// }

// export function useGetAnnouncements() {
//     return useServerMutation<Array<Updater_Announcement>, GetAnnouncements_Variables>({
//         endpoint: API_ENDPOINTS.STATUS.GetAnnouncements.endpoint,
//         method: API_ENDPOINTS.STATUS.GetAnnouncements.methods[0],
//         mutationKey: [API_ENDPOINTS.STATUS.GetAnnouncements.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetMemoryStats() {
//     return useServerQuery<MemoryStatsResponse>({
//         endpoint: API_ENDPOINTS.STATUS.GetMemoryStats.endpoint,
//         method: API_ENDPOINTS.STATUS.GetMemoryStats.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetMemoryStats.key],
//         enabled: true,
//     })
// }

// export function useGetMemoryProfile() {
//     return useServerQuery<null>({
//         endpoint: API_ENDPOINTS.STATUS.GetMemoryProfile.endpoint,
//         method: API_ENDPOINTS.STATUS.GetMemoryProfile.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetMemoryProfile.key],
//         enabled: true,
//     })
// }

// export function useGetGoRoutineProfile() {
//     return useServerQuery<null>({
//         endpoint: API_ENDPOINTS.STATUS.GetGoRoutineProfile.endpoint,
//         method: API_ENDPOINTS.STATUS.GetGoRoutineProfile.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetGoRoutineProfile.key],
//         enabled: true,
//     })
// }

// export function useGetCPUProfile() {
//     return useServerQuery<null>({
//         endpoint: API_ENDPOINTS.STATUS.GetCPUProfile.endpoint,
//         method: API_ENDPOINTS.STATUS.GetCPUProfile.methods[0],
//         queryKey: [API_ENDPOINTS.STATUS.GetCPUProfile.key],
//         enabled: true,
//     })
// }

// export function useForceGC() {
//     return useServerMutation<MemoryStatsResponse>({
//         endpoint: API_ENDPOINTS.STATUS.ForceGC.endpoint,
//         method: API_ENDPOINTS.STATUS.ForceGC.methods[0],
//         mutationKey: [API_ENDPOINTS.STATUS.ForceGC.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// sync_library
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useSyncUserLibrary() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_LIBRARY.SyncUserLibrary.endpoint,
//         method: API_ENDPOINTS.SYNC_LIBRARY.SyncUserLibrary.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_LIBRARY.SyncUserLibrary.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useLibraryChanged() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_LIBRARY.LibraryChanged.endpoint,
//         method: API_ENDPOINTS.SYNC_LIBRARY.LibraryChanged.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_LIBRARY.LibraryChanged.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetSyncStats() {
//     return useServerQuery<Record<string, interface{}>>({
//         endpoint: API_ENDPOINTS.SYNC_LIBRARY.GetSyncStats.endpoint,
//         method: API_ENDPOINTS.SYNC_LIBRARY.GetSyncStats.methods[0],
//         queryKey: [API_ENDPOINTS.SYNC_LIBRARY.GetSyncStats.key],
//         enabled: true,
//     })
// }

// export function useRegisterWebSocketSession() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_LIBRARY.RegisterWebSocketSession.endpoint,
//         method: API_ENDPOINTS.SYNC_LIBRARY.RegisterWebSocketSession.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_LIBRARY.RegisterWebSocketSession.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUnregisterWebSocketSession() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_LIBRARY.UnregisterWebSocketSession.endpoint,
//         method: API_ENDPOINTS.SYNC_LIBRARY.UnregisterWebSocketSession.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_LIBRARY.UnregisterWebSocketSession.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// sync_progress
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetUserProgress() {
//     return useServerQuery<Array<Models_UserEpisodeProgress>>({
//         endpoint: API_ENDPOINTS.SYNC_PROGRESS.GetUserProgress.endpoint,
//         method: API_ENDPOINTS.SYNC_PROGRESS.GetUserProgress.methods[0],
//         queryKey: [API_ENDPOINTS.SYNC_PROGRESS.GetUserProgress.key],
//         enabled: true,
//     })
// }

// export function useGetResumePoint() {
//     return useServerQuery<ResumePoint>({
//         endpoint: API_ENDPOINTS.SYNC_PROGRESS.GetResumePoint.endpoint,
//         method: API_ENDPOINTS.SYNC_PROGRESS.GetResumePoint.methods[0],
//         queryKey: [API_ENDPOINTS.SYNC_PROGRESS.GetResumePoint.key],
//         enabled: true,
//     })
// }

// export function useStartWatching() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_PROGRESS.StartWatching.endpoint,
//         method: API_ENDPOINTS.SYNC_PROGRESS.StartWatching.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_PROGRESS.StartWatching.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateProgress() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_PROGRESS.UpdateProgress.endpoint,
//         method: API_ENDPOINTS.SYNC_PROGRESS.UpdateProgress.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_PROGRESS.UpdateProgress.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function usePauseWatching() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_PROGRESS.PauseWatching.endpoint,
//         method: API_ENDPOINTS.SYNC_PROGRESS.PauseWatching.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_PROGRESS.PauseWatching.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useStopWatching() {
//     return useServerMutation<boolean>({
//         endpoint: API_ENDPOINTS.SYNC_PROGRESS.StopWatching.endpoint,
//         method: API_ENDPOINTS.SYNC_PROGRESS.StopWatching.methods[0],
//         mutationKey: [API_ENDPOINTS.SYNC_PROGRESS.StopWatching.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// theme
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetTheme() {
//     return useServerQuery<Models_Theme>({
//         endpoint: API_ENDPOINTS.THEME.GetTheme.endpoint,
//         method: API_ENDPOINTS.THEME.GetTheme.methods[0],
//         queryKey: [API_ENDPOINTS.THEME.GetTheme.key],
//         enabled: true,
//     })
// }

// export function useUpdateTheme() {
//     return useServerMutation<Models_Theme, UpdateTheme_Variables>({
//         endpoint: API_ENDPOINTS.THEME.UpdateTheme.endpoint,
//         method: API_ENDPOINTS.THEME.UpdateTheme.methods[0],
//         mutationKey: [API_ENDPOINTS.THEME.UpdateTheme.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// torrent_client
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useGetActiveTorrentList() {
//     return useServerQuery<Array<TorrentClient_Torrent>>({
//         endpoint: API_ENDPOINTS.TORRENT_CLIENT.GetActiveTorrentList.endpoint,
//         method: API_ENDPOINTS.TORRENT_CLIENT.GetActiveTorrentList.methods[0],
//         queryKey: [API_ENDPOINTS.TORRENT_CLIENT.GetActiveTorrentList.key],
//         enabled: true,
//     })
// }

// export function useTorrentClientAction() {
//     return useServerMutation<boolean, TorrentClientAction_Variables>({
//         endpoint: API_ENDPOINTS.TORRENT_CLIENT.TorrentClientAction.endpoint,
//         method: API_ENDPOINTS.TORRENT_CLIENT.TorrentClientAction.methods[0],
//         mutationKey: [API_ENDPOINTS.TORRENT_CLIENT.TorrentClientAction.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useTorrentClientDownload() {
//     return useServerMutation<boolean, TorrentClientDownload_Variables>({
//         endpoint: API_ENDPOINTS.TORRENT_CLIENT.TorrentClientDownload.endpoint,
//         method: API_ENDPOINTS.TORRENT_CLIENT.TorrentClientDownload.methods[0],
//         mutationKey: [API_ENDPOINTS.TORRENT_CLIENT.TorrentClientDownload.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useTorrentClientAddMagnetFromRule() {
//     return useServerMutation<boolean, TorrentClientAddMagnetFromRule_Variables>({
//         endpoint: API_ENDPOINTS.TORRENT_CLIENT.TorrentClientAddMagnetFromRule.endpoint,
//         method: API_ENDPOINTS.TORRENT_CLIENT.TorrentClientAddMagnetFromRule.methods[0],
//         mutationKey: [API_ENDPOINTS.TORRENT_CLIENT.TorrentClientAddMagnetFromRule.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// torrent_search
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useSearchTorrent() {
//     return useServerMutation<Torrent_SearchData, SearchTorrent_Variables>({
//         endpoint: API_ENDPOINTS.TORRENT_SEARCH.SearchTorrent.endpoint,
//         method: API_ENDPOINTS.TORRENT_SEARCH.SearchTorrent.methods[0],
//         mutationKey: [API_ENDPOINTS.TORRENT_SEARCH.SearchTorrent.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// users
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// export function useUserLogin() {
//     return useServerMutation<LoginResponse>({
//         endpoint: API_ENDPOINTS.USERS.UserLogin.endpoint,
//         method: API_ENDPOINTS.USERS.UserLogin.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.UserLogin.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUserLogout() {
//     return useServerMutation<Record<string, string>>({
//         endpoint: API_ENDPOINTS.USERS.UserLogout.endpoint,
//         method: API_ENDPOINTS.USERS.UserLogout.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.UserLogout.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetUserProfile() {
//     return useServerQuery<Models_User>({
//         endpoint: API_ENDPOINTS.USERS.GetUserProfile.endpoint,
//         method: API_ENDPOINTS.USERS.GetUserProfile.methods[0],
//         queryKey: [API_ENDPOINTS.USERS.GetUserProfile.key],
//         enabled: true,
//     })
// }

// export function useGetUserViewer() {
//     return useServerQuery<AL_GetViewer_Viewer>({
//         endpoint: API_ENDPOINTS.USERS.GetUserViewer.endpoint,
//         method: API_ENDPOINTS.USERS.GetUserViewer.methods[0],
//         queryKey: [API_ENDPOINTS.USERS.GetUserViewer.key],
//         enabled: true,
//     })
// }

// export function useUpdateUserProfile() {
//     return useServerMutation<Models_User>({
//         endpoint: API_ENDPOINTS.USERS.UpdateUserProfile.endpoint,
//         method: API_ENDPOINTS.USERS.UpdateUserProfile.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.UpdateUserProfile.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useChangePassword() {
//     return useServerMutation<Record<string, string>>({
//         endpoint: API_ENDPOINTS.USERS.ChangePassword.endpoint,
//         method: API_ENDPOINTS.USERS.ChangePassword.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.ChangePassword.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetAllUsers() {
//     return useServerQuery<Array<Models_User>>({
//         endpoint: API_ENDPOINTS.USERS.GetAllUsers.endpoint,
//         method: API_ENDPOINTS.USERS.GetAllUsers.methods[0],
//         queryKey: [API_ENDPOINTS.USERS.GetAllUsers.key],
//         enabled: true,
//     })
// }

// export function useCreateUser() {
//     return useServerMutation<Models_User>({
//         endpoint: API_ENDPOINTS.USERS.CreateUser.endpoint,
//         method: API_ENDPOINTS.USERS.CreateUser.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.CreateUser.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useUpdateUser() {
//     return useServerMutation<Models_User>({
//         endpoint: API_ENDPOINTS.USERS.UpdateUser.endpoint,
//         method: API_ENDPOINTS.USERS.UpdateUser.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.UpdateUser.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useDeleteUser() {
//     return useServerMutation<Record<string, string>>({
//         endpoint: API_ENDPOINTS.USERS.DeleteUser.endpoint,
//         method: API_ENDPOINTS.USERS.DeleteUser.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.DeleteUser.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useResetUserPassword() {
//     return useServerMutation<Record<string, string>>({
//         endpoint: API_ENDPOINTS.USERS.ResetUserPassword.endpoint,
//         method: API_ENDPOINTS.USERS.ResetUserPassword.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.ResetUserPassword.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useGetWhitelist() {
//     return useServerQuery<Array<string>>({
//         endpoint: API_ENDPOINTS.USERS.GetWhitelist.endpoint,
//         method: API_ENDPOINTS.USERS.GetWhitelist.methods[0],
//         queryKey: [API_ENDPOINTS.USERS.GetWhitelist.key],
//         enabled: true,
//     })
// }

// export function useAddToWhitelist() {
//     return useServerMutation<Record<string, string>>({
//         endpoint: API_ENDPOINTS.USERS.AddToWhitelist.endpoint,
//         method: API_ENDPOINTS.USERS.AddToWhitelist.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.AddToWhitelist.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useRemoveFromWhitelist() {
//     return useServerMutation<Record<string, string>>({
//         endpoint: API_ENDPOINTS.USERS.RemoveFromWhitelist.endpoint,
//         method: API_ENDPOINTS.USERS.RemoveFromWhitelist.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.RemoveFromWhitelist.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

// export function useSetupRequired() {
//     return useServerQuery<Record<string, boolean>>({
//         endpoint: API_ENDPOINTS.USERS.SetupRequired.endpoint,
//         method: API_ENDPOINTS.USERS.SetupRequired.methods[0],
//         queryKey: [API_ENDPOINTS.USERS.SetupRequired.key],
//         enabled: true,
//     })
// }

// export function useGetUserPreference() {
//     return useServerQuery<UserPreferenceResponse>({
//         endpoint: API_ENDPOINTS.USERS.GetUserPreference.endpoint,
//         method: API_ENDPOINTS.USERS.GetUserPreference.methods[0],
//         queryKey: [API_ENDPOINTS.USERS.GetUserPreference.key],
//         enabled: true,
//     })
// }

// export function useSetUserPreference() {
//     return useServerMutation<UserPreferenceResponse>({
//         endpoint: API_ENDPOINTS.USERS.SetUserPreference.endpoint,
//         method: API_ENDPOINTS.USERS.SetUserPreference.methods[0],
//         mutationKey: [API_ENDPOINTS.USERS.SetUserPreference.key],
//         onSuccess: async () => {
// 
//         },
//     })
// }

