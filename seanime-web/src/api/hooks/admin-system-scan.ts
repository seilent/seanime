import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { toast } from "sonner"

export interface SystemScanOptions {
    libraryPaths?: string[]
    skipExistingFiles?: boolean
    forceRescan?: boolean
}

export interface SystemScanResult {
    filesProcessed: number
    newMappings: number
    updatedMappings: number
    errors: string[]
    duration: string
    unmappedFilePaths: string[]
}

export interface SystemScanStatus {
    isScanning: boolean
    autoScanEnabled: boolean
    isDebouncing: boolean
    lastScan: string | null
    totalFiles: number
    mappedFiles: number
    unmappedFiles: number
}

/**
 * Hook to start a system-wide scan
 * Requires admin privileges
 */
export function useStartSystemScan() {
    const queryClient = useQueryClient()

    return useMutation<SystemScanResult, Error, SystemScanOptions>({
        mutationFn: async (options: SystemScanOptions) => {
            const response = await fetch(API_ENDPOINTS.ADMIN_SYSTEM_SCAN.StartSystemScan.endpoint, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                credentials: "include",
                body: JSON.stringify({
                    libraryPaths: options.libraryPaths || [],
                    skipExistingFiles: options.skipExistingFiles ?? true,
                    forceRescan: options.forceRescan ?? false,
                }),
            })

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({})) as { error?: string }
                throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`)
            }

            const data = await response.json() as { data: any }
            return data.data
        },
        onSuccess: (data) => {
            // Invalidate status query to refresh the UI
            queryClient.invalidateQueries({ queryKey: ["admin-system-scan-status"] })

            toast.success(
                `System scan completed successfully! Processed ${data.filesProcessed} files, ${data.newMappings} new mappings, ${data.updatedMappings} updated mappings.`,
                { duration: 8000 }
            )
        },
        onError: (error) => {
            console.error("System scan failed:", error)
            toast.error(`System scan failed: ${error.message}`, { duration: 10000 })
        },
    })
}

/**
 * Hook to get system scan status
 * Requires admin privileges
 */
export function useGetSystemScanStatus() {
    return useQuery<SystemScanStatus>({
        queryKey: ["admin-system-scan-status"],
        queryFn: async () => {
            const response = await fetch(API_ENDPOINTS.ADMIN_SYSTEM_SCAN.GetSystemScanStatus.endpoint, {
                credentials: "include",
            })

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({})) as { error?: string }
                throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`)
            }

            const data = await response.json() as { data: any }
            return data.data
        },
        refetchInterval: 5000, // Refetch every 5 seconds to keep status updated
        refetchIntervalInBackground: false,
    })
}