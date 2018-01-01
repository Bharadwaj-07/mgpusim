package timing

import (
	"log"
	"reflect"

	"gitlab.com/yaotsu/core"
)

// VMemUnit is the execution unit that is responsible for executing the
// local data share instuctions
//
// ToVMemL1 <=>
//
// ToScheduler <=>
//
// FromDecoder <=>
type VMemUnit struct {
	*core.ComponentBase
}

// NewVMemUnit creates and retuns a new VMemUnit
func NewVMemUnit(name string) *VMemUnit {
	u := new(VMemUnit)
	u.ComponentBase = core.NewComponentBase(name)
	u.AddPort("ToVMemL1")
	u.AddPort("ToScheduler")
	u.AddPort("FromDecoder")
	return u
}

// Recv defines the how the VMemUnit process incomming requests
func (u *VMemUnit) Recv(req core.Req) *core.Error {
	switch req := req.(type) {
	case *IssueInstReq:
		replyReq := NewInstCompletionReq(u, req.Scheduler, req.RecvTime(),
			req.Wf)
		u.GetConnection("ToScheduler").Send(replyReq)
	default:
		log.Panicf("cannot process request of type %s", reflect.TypeOf(req))
	}
	return nil
}

// Handle defines how the VMemUnit handles events
func (u *VMemUnit) Handle(evt core.Event) error {
	return nil
}
