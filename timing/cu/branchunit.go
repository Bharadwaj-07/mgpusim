package cu

import (
	"log"
	"reflect"

	"gitlab.com/yaotsu/core"
)

// BranchUnit is the execution unit that is responsible for executing the
// local data share instuctions
//
// ToScheduler <=> The port that connects the BranchUnit and the Scheduler
type BranchUnit struct {
	*core.ComponentBase

	Freq core.Freq

	engine    core.Engine
	scheduler core.Component
	running   bool

	reading   *Wavefront
	executing *Wavefront
	writing   *Wavefront
}

// NewBranchUnit creates and retuns a new BranchUnit
func NewBranchUnit(name string, engine core.Engine, scheduler core.Component) *BranchUnit {
	u := new(BranchUnit)
	u.ComponentBase = core.NewComponentBase(name)
	u.engine = engine
	u.scheduler = scheduler
	u.AddPort("ToScheduler")
	return u
}

// Recv defines the how the BranchUnit process incomming requests
func (u *BranchUnit) Recv(req core.Req) *core.Error {
	switch req := req.(type) {
	case *IssueInstReq:
		return u.processIssueInstReq(req)
	default:
		log.Panicf("cannot process request of type %s", reflect.TypeOf(req))
	}
	return nil
}

func (u *BranchUnit) processIssueInstReq(req *IssueInstReq) *core.Error {
	if u.reading != nil {
		return core.NewError("unit busy", true, u.Freq.NextTick(req.RecvTime()))
	}

	u.reading = req.Wf
	u.tryStartTick(req.RecvTime())
	return nil
}

// Handle defines how the BranchUnit handles events
func (u *BranchUnit) Handle(evt core.Event) error {
	switch evt := evt.(type) {
	case *core.TickEvent:
		return u.handleTickEvent(evt)
	default:
		log.Panicf("cannot handle event of type %s", reflect.TypeOf(evt))
	}
	return nil
}

func (u *BranchUnit) handleTickEvent(evt *core.TickEvent) error {
	u.doWrite(evt.Time())
	u.doExec(evt.Time())
	u.doRead(evt.Time())

	u.continueTick(evt.Time())

	return nil
}

func (u *BranchUnit) doWrite(now core.VTimeInSec) {
	if u.writing != nil {
		req := NewInstCompletionReq(u, u.scheduler, now, u.writing)
		u.GetConnection("ToScheduler").Send(req)
		u.writing = nil
	}
}

func (u *BranchUnit) doExec(now core.VTimeInSec) {
	if u.executing != nil {
		if u.writing == nil {
			u.writing = u.executing
			u.executing = nil
		}
	}
}

func (u *BranchUnit) doRead(now core.VTimeInSec) {
	if u.reading != nil {
		if u.executing == nil {
			u.executing = u.reading
			u.reading = nil
		}
	}
}

func (u *BranchUnit) tryStartTick(now core.VTimeInSec) {
	if !u.running {
		u.scheduleTick(now)
	}
}

func (u *BranchUnit) continueTick(now core.VTimeInSec) {
	if u.reading == nil &&
		u.executing == nil &&
		u.writing == nil {
		u.running = false
	}

	if u.running {
		u.scheduleTick(now)
	}
}

func (u *BranchUnit) scheduleTick(now core.VTimeInSec) {
	evt := core.NewTickEvent(now, u)
	u.engine.Schedule(evt)
	u.running = true
}
