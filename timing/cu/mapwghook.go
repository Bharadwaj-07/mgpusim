package cu

import (
	"log"
	"reflect"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/timing"
)

// MapWGHook is the hook that hooks to MapWGEvent
type MapWGHook struct {
}

// NewMapWGHook returns a newly created MapWGHook
func NewMapWGHook() *MapWGHook {
	h := new(MapWGHook)
	return h
}

// Type returns type timing.MapWGReq
func (h *MapWGHook) Type() reflect.Type {
	return reflect.TypeOf((*timing.MapWGReq)(nil))
}

// Pos return AfterEvent
func (h *MapWGHook) Pos() core.HookPos {
	return core.AfterEvent
}

// Func defines the behavior when the hook is triggered
func (h *MapWGHook) Func(item interface{}, domain core.Hookable) {
	evt := item.(*timing.MapWGReq)
	log.Printf("MapWG: ok: %t", evt.Ok)
}
