package plugin

import (
	"seanime/internal/continuity"
	"seanime/internal/extension"
	"seanime/internal/goja/goja_bindings"
	goja_util "seanime/internal/util/goja"

	"github.com/dop251/goja"
	"github.com/rs/zerolog"
)

func (a *AppContextImpl) BindContinuityToContextObj(vm *goja.Runtime, obj *goja.Object, logger *zerolog.Logger, ext *extension.Extension, scheduler *goja_util.Scheduler) {

	continuityObj := vm.NewObject()

	_ = continuityObj.Set("updateWatchHistoryItem", func(userID uint, opts continuity.UpdateWatchHistoryItemOptions) goja.Value {
		manager, ok := a.continuityManager.Get()
		if !ok {
			goja_bindings.PanicThrowErrorString(vm, "continuity manager not set")
		}
		err := manager.UpdateWatchHistoryItemForUser(userID, &opts)
		if err != nil {
			goja_bindings.PanicThrowError(vm, err)
		}
		return goja.Undefined()
	})

	_ = continuityObj.Set("getWatchHistoryItem", func(userID uint, mediaId int) goja.Value {
		manager, ok := a.continuityManager.Get()
		if !ok {
			goja_bindings.PanicThrowErrorString(vm, "continuity manager not set")
		}
		resp := manager.GetWatchHistoryItemForUser(userID, mediaId)
		if resp == nil || !resp.Found {
			return goja.Undefined()
		}
		return vm.ToValue(resp.Item)
	})

	_ = continuityObj.Set("getWatchHistory", func(userID uint) goja.Value {
		manager, ok := a.continuityManager.Get()
		if !ok {
			goja_bindings.PanicThrowErrorString(vm, "continuity manager not set")
		}
		return vm.ToValue(manager.GetWatchHistoryForUser(userID))
	})

	_ = continuityObj.Set("deleteWatchHistoryItem", func(userID uint, mediaId int) goja.Value {
		manager, ok := a.continuityManager.Get()
		if !ok {
			goja_bindings.PanicThrowErrorString(vm, "continuity manager not set")
		}
		err := manager.DeleteWatchHistoryItemForUser(userID, mediaId)
		if err != nil {
			goja_bindings.PanicThrowError(vm, err)
		}
		return goja.Undefined()
	})

	_ = obj.Set("continuity", continuityObj)
}
