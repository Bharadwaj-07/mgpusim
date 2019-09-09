package gcn3

import (
	"log"
	"reflect"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3/kernels"
	"gitlab.com/akita/util/tracing"
	"gopkg.in/cheggaaa/pb.v1"
)

// A MapWGEvent is an event used by the dispatcher to map a work-group
type MapWGEvent struct {
	*akita.EventBase
}

// NewMapWGEvent creates a new MapWGEvent
func NewMapWGEvent(t akita.VTimeInSec, handler akita.Handler) *MapWGEvent {
	e := new(MapWGEvent)
	e.EventBase = akita.NewEventBase(t, handler)
	return e
}

// DispatcherState defines the current state of the dispatcher
type DispatcherState int

// A list of all possible dispatcher states
const (
	DispatcherIdle DispatcherState = iota
	DispatcherToMapWG
	DispatcherWaitMapWGACK
)

// A Dispatcher is a component that can dispatch work-groups and wavefronts
// to ComputeUnits.
//
//     <=> ToCUs The connection that is connecting the dispatcher and the
//         compute units
//
//     <=> ToCP The connection that is connecting the dispatcher
//         with the command processor
//
// The protocol that is defined by the dispatcher is as follows:
//
// When the dispatcher receives a LaunchKernelReq request from the command
// processor, the kernel launching process is started. One dispatcher can only
// process one kernel at a time. So if the dispatcher is busy when the
// LaunchKernel is received, an NACK will be replied to the command processor.
//
// During the kernel dispatching process, the dispatcher will first check if
// the next compute unit can map a workgroup or not by sending a MapWGReq.
// The selection of the compute unit is in a round-robin fashion. If the
// compute unit can map a work-group, the dispatcher will dispatch wavefronts
// onto the compute unit by sending DispatchWfReq. The dispatcher will wait
// for the compute unit to return completion message for the DispatchWfReq
// before dispatching the next wavefront.
//
// Dispatcher receives
//
//     KernelDispatchReq ---- Request the dispatcher to dispatch the a kernel
//                            to the compute units (Initialize)
//
//     MapWGReq ---- The request return from the compute unit tells if the
//                   compute unit is able to run the work-group (Receive(?))
//
//     WGFinishMesg ---- The CU send this message to the dispatcher to notify
//                       the completion of a workgroup (Finalization(?))
//
type Dispatcher struct {
	*akita.ComponentBase

	CUs    []akita.Port
	cuBusy map[akita.Port]bool

	engine      akita.Engine
	gridBuilder kernels.GridBuilder
	Freq        akita.Freq

	// The request that is being processed, one dispatcher can only dispatch one kernel at a time.
	dispatchingReq  *LaunchKernelReq
	totalWGs        int
	currentWG       *kernels.WorkGroup
	dispatchedWGs   map[string]*MapWGReq
	completedWGs    []*kernels.WorkGroup
	dispatchingWfs  []*kernels.Wavefront
	dispatchingCUID int
	state           DispatcherState
	progressBar     *pb.ProgressBar

	ToCUs              akita.Port
	ToCommandProcessor akita.Port
}

func (d *Dispatcher) NotifyRecv(now akita.VTimeInSec, port akita.Port) {
	req := port.Retrieve(now)
	akita.ProcessMsgAsEvent(req, d.engine, d.Freq)
}

func (d *Dispatcher) NotifyPortFree(now akita.VTimeInSec, port akita.Port) {
	//panic("implement me")
}

// Handle perform actions when an event is triggered
func (d *Dispatcher) Handle(evt akita.Event) error {
	ctx := akita.HookCtx{
		Domain: d,
		Now:    evt.Time(),
		Pos:    akita.HookPosBeforeEvent,
		Item:   evt,
	}
	d.InvokeHook(&ctx)

	d.Lock()
	switch evt := evt.(type) {
	case *LaunchKernelReq:
		d.handleLaunchKernelReq(evt)
	case *MapWGEvent:
		d.handleMapWGEvent(evt)
	case *MapWGReq:
		d.handleMapWGReq(evt)
	case *WGFinishMesg:
		d.handleWGFinishMesg(evt)
	default:
		log.Panicf("Unable to process evevt of type %s", reflect.TypeOf(evt))
	}
	d.Unlock()

	ctx.Pos = akita.HookPosAfterEvent
	d.InvokeHook(&ctx)

	return nil
}

func (d *Dispatcher) handleLaunchKernelReq(
	req *LaunchKernelReq,
) error {

	if d.dispatchingReq != nil {
		log.Panic("dispatcher not done dispatching the previous kernel")
	}

	d.initKernelDispatching(req.Time(), req)
	d.scheduleMapWG(d.Freq.NextTick(req.Time()))

	tracing.TraceReqReceive(req, req.Time(), d)

	return nil
}

func (d *Dispatcher) replyLaunchKernelReq(
	ok bool,
	req *LaunchKernelReq,
	now akita.VTimeInSec,
) *akita.SendError {
	req.OK = ok
	req.Src, req.Dst = req.Dst, req.Src
	req.SendTime = req.RecvTime
	return d.ToCommandProcessor.Send(req)
}

// handleMapWGEvent initiates work-group mapping
func (d *Dispatcher) handleMapWGEvent(evt *MapWGEvent) error {
	now := evt.Time()

	wg := d.currentWG
	if wg == nil {
		wg = d.gridBuilder.NextWG()
		if wg == nil {
			d.state = DispatcherIdle
			return nil
		}
		d.currentWG = wg
	}

	cuID, hasAvailableCU := d.nextAvailableCU()
	if !hasAvailableCU {
		d.state = DispatcherIdle
		return nil
	}

	CU := d.CUs[cuID]
	req := NewMapWGReq(d.ToCUs, CU, now, wg)
	req.PID = d.dispatchingReq.PID
	d.state = DispatcherWaitMapWGACK
	err := d.ToCUs.Send(req)
	if err != nil {
		d.scheduleMapWG(d.Freq.NextTick(now))
		return nil
	}

	d.dispatchedWGs[wg.UID] = req
	d.dispatchingCUID = cuID

	tracing.TraceReqInitiate(
		req, now, d,
		tracing.MsgIDAtReceiver(d.dispatchingReq, d))

	return nil
}

func (d *Dispatcher) initKernelDispatching(
	now akita.VTimeInSec,
	req *LaunchKernelReq,
) {
	d.dispatchingReq = req
	d.gridBuilder.SetKernel(kernels.KernelLaunchInfo{
		CodeObject: req.HsaCo,
		Packet:     req.Packet,
		PacketAddr: req.PacketAddress,
	})
	d.totalWGs = d.gridBuilder.NumWG()
	d.dispatchingCUID = -1
	d.progressBar = pb.New(d.totalWGs)
	d.progressBar.ShowTimeLeft = true
	d.progressBar.Start()

	tracing.TraceReqReceive(
		req, now, d)

}

func (d *Dispatcher) scheduleMapWG(time akita.VTimeInSec) {
	evt := NewMapWGEvent(time, d)
	d.engine.Schedule(evt)
}

// handleMapWGReq deals with the response of the MapWGReq from a compute unit.
func (d *Dispatcher) handleMapWGReq(req *MapWGReq) error {
	now := req.Time()

	if !req.Ok {
		d.state = DispatcherToMapWG
		d.cuBusy[d.CUs[d.dispatchingCUID]] = true
		d.scheduleMapWG(now)

		delete(d.dispatchedWGs, d.currentWG.UID)

		tracing.TraceReqReceive(
			req, now, d)

		return nil
	}

	//wg := d.dispatchingWGs[0]
	d.currentWG = nil
	//d.dispatchingWfs = append(d.dispatchingWfs, wg.Wavefronts...)
	d.state = DispatcherToMapWG
	d.scheduleMapWG(now)

	return nil
}

func (d *Dispatcher) handleWGFinishMesg(mesg *WGFinishMesg) error {
	d.completedWGs = append(d.completedWGs, mesg.WG)
	d.cuBusy[mesg.Src] = false

	mapWGReq := d.dispatchedWGs[mesg.WG.UID]
	delete(d.dispatchedWGs, mesg.WG.UID)

	tracing.TraceReqFinalize(
		mapWGReq, mesg.Time(), d)

	if d.progressBar != nil {
		d.progressBar.Increment()
	}

	if d.totalWGs <= len(d.completedWGs) {
		if d.progressBar != nil {
			d.progressBar.Finish()
		}
		d.replyKernelFinish(mesg.Time())
		return nil
	}

	if d.state == DispatcherIdle {
		d.state = DispatcherToMapWG
		d.scheduleMapWG(d.Freq.NextTick(mesg.Time()))
	}
	return nil
}

func (d *Dispatcher) replyKernelFinish(now akita.VTimeInSec) {
	req := d.dispatchingReq
	req.Src, req.Dst = req.Dst, req.Src
	req.SendTime = now

	d.completedWGs = nil
	d.dispatchingReq = nil

	err := d.ToCommandProcessor.Send(req)
	if err != nil {
		log.Panic(err)
	}

	tracing.TraceReqComplete(
		req, now, d)
}

// RegisterCU adds a CU to the dispatcher so that the dispatcher can
// dispatches wavefronts to the CU
func (d *Dispatcher) RegisterCU(cu akita.Port) {
	d.CUs = append(d.CUs, cu)
	d.cuBusy[cu] = false
}

func (d *Dispatcher) nextAvailableCU() (int, bool) {
	count := len(d.cuBusy)
	cuID := d.dispatchingCUID
	for i := 0; i < count; i++ {
		cuID++
		if cuID >= len(d.cuBusy) {
			cuID = 0
		}

		if !d.cuBusy[d.CUs[cuID]] {
			return cuID, true
		}
	}
	return -1, false
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(
	name string,
	engine akita.Engine,
	gridBuilder kernels.GridBuilder,
) *Dispatcher {
	d := new(Dispatcher)
	d.ComponentBase = akita.NewComponentBase(name)

	d.gridBuilder = gridBuilder
	d.engine = engine

	d.CUs = make([]akita.Port, 0)
	d.cuBusy = make(map[akita.Port]bool, 0)
	d.dispatchedWGs = make(map[string]*MapWGReq)

	d.ToCommandProcessor = akita.NewLimitNumMsgPort(d, 1)
	d.ToCUs = akita.NewLimitNumMsgPort(d, 1)

	d.state = DispatcherIdle

	return d
}
