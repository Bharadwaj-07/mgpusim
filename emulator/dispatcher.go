package emulator

import (
	"fmt"
	"reflect"

	"gitlab.com/yaotsu/core/conn"
	"gitlab.com/yaotsu/core/event"
)

// A Dispatcher is a Yaotsu component that is responsible for distributing
// the Work-groups to compute units.
//
//   ToCommandProcessor <=> Receives kernel launching requests and send
//                          kernel compeletion signal
//   ToComputeUnits <=> Send MapWorkGroupReq to compute units and
//                      receives from the compute units about the completion
//                      of the workgroups.
type Dispatcher struct {
	*conn.BasicComponent

	ComputeUnits        []conn.Component
	ComputeUnitsRunning []bool
	InFlightGrids       []*Grid

	mapWGReqFactory MapWGReqFactory
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(name string,
	mapWorkGroupReqFactory MapWGReqFactory) *Dispatcher {
	d := new(Dispatcher)
	d.BasicComponent = conn.NewBasicComponent(name)
	d.mapWGReqFactory = mapWorkGroupReqFactory

	d.ComputeUnits = make([]conn.Component, 0)
	d.ComputeUnitsRunning = make([]bool, 0)
	d.InFlightGrids = make([]*Grid, 0)

	d.AddPort("ToCommandProcessor")
	d.AddPort("ToComputeUnits")

	return d
}

// RegisterCU allows the dispatcher to dispatch workgroups to the
// ComputeUnit
func (d *Dispatcher) RegisterCU(cu conn.Component) {
	d.ComputeUnits = append(d.ComputeUnits, cu)
	d.ComputeUnitsRunning = append(d.ComputeUnitsRunning, false)
}

func (d *Dispatcher) dispatch(cu conn.Component, wg *WorkGroup, time event.VTimeInSec) {
	req := d.mapWGReqFactory.Create()
	req.SetSource(d)
	req.SetDestination(cu)
	req.SetSendTime(time)
	req.WG = wg

	d.GetConnection("ToComputeUnits").Send(req)
}

// Receive processes the incomming requests
func (d *Dispatcher) Receive(req conn.Request) *conn.Error {
	switch req := req.(type) {
	case *LaunchKernelReq:
		return d.processLaunchKernelReq(req)
	default:
		return conn.NewError(
			fmt.Sprintf("cannot process request %s", reflect.TypeOf(req)), false, 0)
	}
}

func (d *Dispatcher) processLaunchKernelReq(req *LaunchKernelReq) *conn.Error {
	grid := NewGrid()
	grid.Packet = req.Packet
	grid.CodeObject = req.HsaCo
	grid.SpawnWorkGroups()

	d.InFlightGrids = append(d.InFlightGrids, grid)

	// Dispatch workgroups for the first round
	for i, cu := range d.ComputeUnits {
		// Skip running CUs
		if d.ComputeUnitsRunning[i] {
			continue
		}

		// No more workgroups to schedule
		if len(grid.WorkGroups) == 0 {
			break
		}

		d.dispatch(cu, grid.WorkGroups[0], req.RecvTime())
		grid.WorkGroupsRunning = append(grid.WorkGroupsRunning, grid.WorkGroups[0])
		grid.WorkGroups = grid.WorkGroups[1:]

	}

	return nil
}

// Handle processes the events that is scheduled for the CommandProcessor
func (d *Dispatcher) Handle(e event.Event) error {
	return nil
}
