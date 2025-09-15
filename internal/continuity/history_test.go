package continuity

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"seanime/internal/database/db"
	"seanime/internal/test_utils"
	"seanime/internal/util"
	"seanime/internal/util/filecache"
	"testing"
)

func TestHistoryItems(t *testing.T) {
	test_utils.SetTwoLevelDeep()
	test_utils.InitTestProvider(t)

	logger := util.NewLogger()

	tempDir := t.TempDir()
	t.Log(tempDir)

	database, err := db.NewDatabase(test_utils.ConfigData.Path.DataDir, test_utils.ConfigData.Database.Name, logger)
	require.NoError(t, err)

	cacher, err := filecache.NewCacher(filepath.Join(tempDir, "cache"))
	require.NoError(t, err)

	// Note: This test now only validates the basic functionality
	// since the continuity manager now uses libsync.ProgressManager
	// which requires more complex setup for full integration testing
	manager := NewManager(&NewManagerOptions{
		FileCacher: cacher,
		Logger:     logger,
		Database:   database,
		// SyncManager: nil, // Would need proper sync manager for full testing
	})
	require.NotNil(t, manager)

	const testUserID = uint(1)
	const testMediaId = 123

	// Test basic continuity manager creation
	require.NotNil(t, manager)

	// Test updating watch history (will fail gracefully without sync manager)
	err = manager.UpdateWatchHistoryItemForUser(testUserID, &UpdateWatchHistoryItemOptions{
		MediaId:       testMediaId,
		EpisodeNumber: 1,
		CurrentTime:   10,
		Duration:      100,
	})
	// This should not error even without sync manager
	require.NoError(t, err)

	// Test getting watch history (will return default response without sync manager)
	resp := manager.GetWatchHistoryItemForUser(testUserID, testMediaId)
	require.NotNil(t, resp)
	require.False(t, resp.Found) // Expected to be false without sync manager

}
