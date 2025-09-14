"use client"

import React, { useState } from "react"
import { PageWrapper } from "@/components/shared/page-wrapper"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { FiUsers as Users, FiMap as Map, FiBarChart as BarChart3, FiSettings as Settings } from "react-icons/fi"
import { UnmappedFilesTable } from "./_containers/unmapped-files-table"
import { IgnoredFilesTable } from "./_containers/ignored-files-table"
import { GlobalMappingsTable } from "./_containers/global-mappings-table"
import { ProgressSyncStats } from "./_containers/progress-sync-stats"

export default function AdminPage() {
    const [activeTab, setActiveTab] = useState("file-mapping")


    return (
        <PageWrapper className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold">Admin Panel</h1>
                <p className="text-muted-foreground">
                    Manage server settings, users, and file mappings
                </p>
            </div>

            <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
                <TabsList className="grid grid-cols-2 w-[400px]">
                    <TabsTrigger value="file-mapping" className="flex items-center gap-2">
                        <Map className="h-4 w-4" />
                        File Management
                    </TabsTrigger>
                    <TabsTrigger value="users" className="flex items-center gap-2">
                        <Users className="h-4 w-4" />
                        User Management
                    </TabsTrigger>
                </TabsList>

                <TabsContent value="file-mapping" className="space-y-6">
                    <div className="grid grid-cols-1 gap-6">
                        {/* Progress Sync Stats */}
                        <ProgressSyncStats />

                        {/* Unmapped Files */}
                        <UnmappedFilesTable />

                        {/* Tabs for other file management */}
                        <Tabs defaultValue="ignored" className="space-y-4">
                            <TabsList>
                                <TabsTrigger value="ignored">Ignored Files</TabsTrigger>
                                <TabsTrigger value="mapped">Global Mappings</TabsTrigger>
                            </TabsList>

                            <TabsContent value="ignored">
                                <IgnoredFilesTable />
                            </TabsContent>

                            <TabsContent value="mapped">
                                <GlobalMappingsTable />
                            </TabsContent>
                        </Tabs>
                    </div>
                </TabsContent>

                <TabsContent value="users" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Users className="h-5 w-5" />
                                User Management
                            </CardTitle>
                        </CardHeader>
                        <CardContent>
                            <p className="text-muted-foreground mb-4">
                                User management has been moved to a dedicated page for better organization.
                            </p>
                            <a
                                href="/admin/users"
                                className="inline-flex items-center px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
                            >
                                Go to User Management
                            </a>
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>
        </PageWrapper>
    )
}