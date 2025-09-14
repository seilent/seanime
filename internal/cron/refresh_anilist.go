package cron

import (
)

func RefreshAnilistDataJob(c *JobCtx) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	if c.App.GlobalSettings == nil || c.App.GlobalSettings.Library == nil {
		return
	}

	// Disabled: Collection refresh is now user-specific in pure multiuser system
	// Users trigger their own collection refreshes via handlers
	_ = c // Avoid unused parameter warning
}

func SyncLocalDataJob(c *JobCtx) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	if c.App.GlobalSettings == nil || c.App.GlobalSettings.Library == nil {
		return
	}

	// Disabled: Local data sync is now user-specific in pure multiuser system

	// Disabled: Local data sync is now user-specific in pure multiuser system
	// Users manage their own local data via handlers
	_ = c // Avoid unused parameter warning
}
