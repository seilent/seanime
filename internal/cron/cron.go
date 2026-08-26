package cron

import (
	"seanime/internal/core"
	"seanime/internal/util"
	"time"
)

type JobCtx struct {
	App *core.App
}

func safeRun(name string, fn func()) {
	defer util.RecoverInModule(name)
	fn()
}

func RunJobs(app *core.App) {

	ctx := &JobCtx{
		App: app,
	}

	refreshAnilistTicker := time.NewTicker(10 * time.Minute)
	refreshLocalDataTicker := time.NewTicker(30 * time.Minute)
	refetchReleaseTicker := time.NewTicker(1 * time.Hour)
	refetchAnnouncementsTicker := time.NewTicker(10 * time.Minute)
	subspleaseSyncTicker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-refreshAnilistTicker.C:
			}
		}
	}()

	go func() {
		for {
			select {
			case <-refreshLocalDataTicker.C:
				safeRun("cron/sync-local-data", func() { SyncLocalDataJob(ctx) })
			}
		}
	}()

	go func() {
		for {
			select {
			case <-refetchReleaseTicker.C:
				safeRun("cron/refetch-releases", func() { app.Updater.ShouldRefetchReleases() })
			}
		}
	}()

	go func() {
		for {
			select {
			case <-refetchAnnouncementsTicker.C:
				safeRun("cron/refetch-announcements", func() { app.Updater.FetchAnnouncements() })
			}
		}
	}()

	go func() {
		time.Sleep(2 * time.Minute)
		safeRun("cron/sp-sync", func() { SubsPleaseSyncJob(ctx) })
		for {
			select {
			case <-subspleaseSyncTicker.C:
				safeRun("cron/sp-sync", func() { SubsPleaseSyncJob(ctx) })
			}
		}
	}()

	autoCleanTicker := time.NewTicker(15 * time.Minute)

	go func() {
		time.Sleep(1 * time.Minute)
		safeRun("cron/auto-clean", func() { AutoCleanJob(ctx) })
		for {
			select {
			case <-autoCleanTicker.C:
				safeRun("cron/auto-clean", func() { AutoCleanJob(ctx) })
			}
		}
	}()

}
