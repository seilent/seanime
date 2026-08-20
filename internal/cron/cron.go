package cron

import (
	"seanime/internal/core"
	"time"
)

type JobCtx struct {
	App *core.App
}

func RunJobs(app *core.App) {

	// Run the jobs only if the server is online
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
				// Disabled: Collection refresh is now user-specific in pure multiuser system
				// Users trigger their own collection refreshes via handlers
			}
		}
	}()

	go func() {
		for {
			select {
			case <-refreshLocalDataTicker.C:
				SyncLocalDataJob(ctx)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-refetchReleaseTicker.C:
				app.Updater.ShouldRefetchReleases()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-refetchAnnouncementsTicker.C:
				app.Updater.FetchAnnouncements()
			}
		}
	}()

	go func() {
		time.Sleep(2 * time.Minute)
		SubsPleaseSyncJob(ctx)
		for {
			select {
			case <-subspleaseSyncTicker.C:
				SubsPleaseSyncJob(ctx)
			}
		}
	}()

	autoCleanTicker := time.NewTicker(15 * time.Minute)

	go func() {
		time.Sleep(1 * time.Minute)
		AutoCleanJob(ctx)
		for {
			select {
			case <-autoCleanTicker.C:
				AutoCleanJob(ctx)
			}
		}
	}()

}
