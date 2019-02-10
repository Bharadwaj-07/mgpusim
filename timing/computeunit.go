package timing

import (
	"log"
	"reflect"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3"
	"gitlab.com/akita/gcn3/emu"
	"gitlab.com/akita/gcn3/insts"
	"gitlab.com/akita/gcn3/kernels"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/mem/cache"
)

// Define possible hook positions
const (
	HookPosWGStart = "WG Start"
	HookPosWGEnd   = "WG End"
	HookPosWfStart = "Wf Start"
	HookPosWfEnd   = "Wf End"
)

type CUHookInfo struct {
	Now akita.VTimeInSec
	Pos interface{}
}

// A ComputeUnit in the timing package provides a detailed and accurate
// simulation of a GCN3 ComputeUnit
type ComputeUnit struct {
	*akita.TickingComponent

	WGMapper     WGMapper
	WfDispatcher WfDispatcher
	Decoder      emu.Decoder

	WfPools              []*WavefrontPool
	WfToDispatch         map[*kernels.Wavefront]*WfDispatchInfo
	wgToManagedWgMapping map[*kernels.WorkGroup]*WorkGroup

	inFlightInstFetch       []*InstFetchReqInfo
	inFlightScalarMemAccess []*ScalarMemAccessInfo
	inFlightVectorMemAccess []*VectorMemAccessInfo

	running bool

	Scheduler        Scheduler
	BranchUnit       CUComponent
	VectorMemDecoder CUComponent
	VectorMemUnit    CUComponent
	ScalarDecoder    CUComponent
	VectorDecoder    CUComponent
	LDSDecoder       CUComponent
	ScalarUnit       CUComponent
	SIMDUnit         []CUComponent
	LDSUnit          CUComponent
	SRegFile         RegisterFile
	VRegFile         []RegisterFile

	InstMem          akita.Port
	ScalarMem        akita.Port
	VectorMemModules cache.LowModuleFinder

	ToACE       akita.Port
	ToInstMem   akita.Port
	ToScalarMem akita.Port
	ToVectorMem akita.Port

	ToCP akita.Port
	CP   akita.Port

	inCPRequestProcessingStage akita.Req
	cpRequestHandlingComplete  bool

	isDraining bool

	toSendToCP *gcn3.CUPipelineDrainRsp
}

// Handle processes that events that are scheduled on the ComputeUnit
func (cu *ComputeUnit) Handle(evt akita.Event) error {
	cu.InvokeHook(evt, cu, akita.BeforeEventHookPos, nil)
	cu.Lock()

	switch evt := evt.(type) {
	case akita.TickEvent:
		cu.handleTickEvent(evt)
	case *WfDispatchEvent:
		cu.handleWfDispatchEvent(evt)
	case *WfCompletionEvent:
		cu.handleWfCompletionEvent(evt)
	default:
		cu.Unlock()
		log.Panicf("Unable to process evevt of type %s",
			reflect.TypeOf(evt))
	}

	cu.Unlock()
	cu.InvokeHook(evt, cu, akita.AfterEventHookPos, nil)
	return nil
}

func (cu *ComputeUnit) handleTickEvent(evt akita.TickEvent) {
	now := evt.Time()
	cu.NeedTick = false

	cu.runPipeline(now)
	cu.processInput(now)
	if cu.isDraining {
		cu.drainPipeline(now)
	}
	cu.sendToCP(now)

	if cu.NeedTick {
		cu.TickLater(now)
	}
}

func (cu *ComputeUnit) runPipeline(now akita.VTimeInSec) {
	madeProgress := false

	madeProgress = cu.BranchUnit.Run(now) || madeProgress

	madeProgress = cu.ScalarUnit.Run(now) || madeProgress
	madeProgress = cu.ScalarDecoder.Run(now) || madeProgress

	for _, simdUnit := range cu.SIMDUnit {
		madeProgress = simdUnit.Run(now) || madeProgress
	}
	madeProgress = cu.VectorDecoder.Run(now) || madeProgress

	madeProgress = cu.LDSUnit.Run(now) || madeProgress
	madeProgress = cu.LDSDecoder.Run(now) || madeProgress

	madeProgress = cu.VectorMemUnit.Run(now) || madeProgress
	madeProgress = cu.VectorMemDecoder.Run(now) || madeProgress

	madeProgress = cu.Scheduler.Run(now) || madeProgress

	if madeProgress {
		cu.NeedTick = true
	}
}

func (cu *ComputeUnit) processInput(now akita.VTimeInSec) {
	cu.processInputFromACE(now)
	cu.processInputFromInstMem(now)
	cu.processInputFromScalarMem(now)
	cu.processInputFromVectorMem(now)
	cu.processInputFromCP(now)
}

func (cu *ComputeUnit) processInputFromCP(now akita.VTimeInSec) {

	req := cu.ToCP.Retrieve(now)

	if req == nil {
		return
	}
	cu.NeedTick = true

	//TODO: Should there be a fixed draining latency?
	cu.inCPRequestProcessingStage = req

	switch req := req.(type) {
	case *gcn3.CUPipelineDrainReq:
		cu.handlePipelineDrainReq(now, req)
	case *gcn3.CUPipelineRestart:
		cu.handlePipelineResume(now, req)

	}
}

func (cu *ComputeUnit) handlePipelineDrainReq(
	now akita.VTimeInSec,
	req *gcn3.CUPipelineDrainReq) error {
	//1. Issue drain command to scheduler
	//2. Check all units one by one until all idle
	//3. If all complete issue CU Pipeline Drain Completion respond
	cu.isDraining = true

	cu.Scheduler.StartDraining()

	return nil

}

func (cu *ComputeUnit) handlePipelineResume(
	now akita.VTimeInSec,
	req *gcn3.CUPipelineRestart) error {
	//1. Issue drain command to scheduler
	//2. Check all units one by one until all idle
	//3. If all complete issue CU Pipeline Drain Completion respond
	cu.isDraining = false
	cu.Scheduler.StopDraining()
	return nil

}

func (cu *ComputeUnit) sendToCP(now akita.VTimeInSec,
) {
	if cu.toSendToCP == nil {
		return
	}
	cu.toSendToCP.SetSendTime(now)
	sendErr := cu.ToCP.Send(cu.toSendToCP)
	if sendErr == nil {
		cu.toSendToCP = nil
		cu.NeedTick = true
	}

}

func (cu *ComputeUnit) drainPipeline(now akita.VTimeInSec) {
	drainComplete := true

	drainComplete = drainComplete && cu.BranchUnit.IsIdle()

	drainComplete = drainComplete && cu.ScalarUnit.IsIdle()
	drainComplete = drainComplete && cu.ScalarDecoder.IsIdle()

	for _, simdUnit := range cu.SIMDUnit {
		drainComplete = drainComplete && simdUnit.IsIdle()
	}

	drainComplete = drainComplete && cu.VectorDecoder.IsIdle()

	drainComplete = drainComplete && cu.LDSUnit.IsIdle()
	drainComplete = drainComplete && cu.LDSDecoder.IsIdle()

	drainComplete = drainComplete && cu.VectorMemUnit.IsIdle()
	drainComplete = drainComplete && cu.VectorMemDecoder.IsIdle()

	if drainComplete == true {
		respondToCP := gcn3.NewCUPipelineDrainRsp(now, cu.ToCP, cu.CP)
		cu.toSendToCP = respondToCP

	}

	//cu.NeedTick = true

}

func (cu *ComputeUnit) processInputFromACE(now akita.VTimeInSec) {
	req := cu.ToACE.Retrieve(now)
	if req == nil {
		return
	}

	switch req := req.(type) {
	case *gcn3.MapWGReq:
		cu.handleMapWGReq(now, req)
	}
}

func (cu *ComputeUnit) handleMapWGReq(
	now akita.VTimeInSec,
	req *gcn3.MapWGReq,
) error {
	//log.Printf("%s map wg at %.12f\n", cu.Name(), req.Time())

	ok := false

	if len(cu.WfToDispatch) == 0 {
		ok = cu.WGMapper.MapWG(req)
	}

	wfs := make([]*Wavefront, 0)
	if ok {
		cu.wrapWG(req.WG, req)
		for _, wf := range req.WG.Wavefronts {
			managedWf := cu.wrapWf(wf)
			managedWf.pid = req.PID
			wfs = append(wfs, managedWf)
		}

		for i, wf := range wfs {
			evt := NewWfDispatchEvent(cu.Freq.NCyclesLater(i, now), cu, wf)
			cu.Engine.Schedule(evt)
		}

		lastEventCycle := 4
		if len(wfs) > 4 {
			lastEventCycle = len(wfs)
		}
		evt := NewWfDispatchEvent(cu.Freq.NCyclesLater(lastEventCycle, now), cu, nil)
		evt.MapWGReq = req
		evt.IsLastInWG = true
		cu.Engine.Schedule(evt)

		cu.InvokeHook(req, cu, HookPosWGStart, &CUHookInfo{
			Now: now,
			Pos: HookPosWGStart,
		})

		return nil
	}

	req.Ok = false
	req.SwapSrcAndDst()
	req.SetSendTime(now)
	err := cu.ToACE.Send(req)
	if err != nil {
		log.Panic(err)
	}

	return nil
}

func (cu *ComputeUnit) handleWfDispatchEvent(
	evt *WfDispatchEvent,
) error {
	now := evt.Time()
	wf := evt.ManagedWf
	if wf != nil {
		info := cu.WfToDispatch[wf.Wavefront]
		cu.WfPools[info.SIMDID].AddWf(wf)
		cu.WfDispatcher.DispatchWf(now, wf)
		delete(cu.WfToDispatch, wf.Wavefront)
		wf.State = WfReady

		cu.InvokeHook(wf, cu, HookPosWfStart, &CUHookInfo{
			Now: evt.Time(),
			Pos: HookPosWfStart,
		})
	}

	// Respond ACK
	if evt.IsLastInWG {
		req := evt.MapWGReq
		req.Ok = true
		req.SwapSrcAndDst()
		req.SetSendTime(evt.Time())
		err := cu.ToACE.Send(req)
		if err != nil {
			log.Panic(err)
		}
	}

	cu.running = true
	cu.TickLater(now)

	return nil
}

func (cu *ComputeUnit) handleWfCompletionEvent(evt *WfCompletionEvent) error {
	wf := evt.Wf
	wg := wf.WG
	wf.State = WfCompleted

	cu.InvokeHook(wf, cu, HookPosWfEnd, &CUHookInfo{
		Now: evt.Time(),
		Pos: HookPosWfEnd,
	})

	if cu.isAllWfInWGCompleted(wg) {
		ok := cu.sendWGCompletionMessage(evt, wg)
		if ok {
			cu.clearWGResource(wg)

			cu.InvokeHook(wg, cu, HookPosWGEnd, &CUHookInfo{
				Now: evt.Time(),
				Pos: HookPosWGEnd,
			})
		}

		if !cu.hasMoreWfsToRun() {
			cu.running = false
		}
	}
	cu.TickLater(evt.Time())

	return nil
}

func (cu *ComputeUnit) clearWGResource(wg *WorkGroup) {
	cu.WGMapper.UnmapWG(wg)
	for _, wf := range wg.Wfs {
		wfPool := cu.WfPools[wf.SIMDID]
		wfPool.RemoveWf(wf)
	}
}

func (cu *ComputeUnit) isAllWfInWGCompleted(wg *WorkGroup) bool {
	for _, wf := range wg.Wfs {
		if wf.State != WfCompleted {
			return false
		}
	}
	return true
}

func (cu *ComputeUnit) sendWGCompletionMessage(
	evt *WfCompletionEvent,
	wg *WorkGroup,
) bool {
	mapReq := wg.MapReq
	dispatcher := mapReq.Dst() // This is dst since the mapReq has been sent back already
	now := evt.Time()
	mesg := gcn3.NewWGFinishMesg(cu.ToACE, dispatcher, now, wg.WorkGroup)

	err := cu.ToACE.Send(mesg)
	if err != nil {
		newEvent := NewWfCompletionEvent(cu.Freq.NextTick(now), cu, evt.Wf)
		cu.Engine.Schedule(newEvent)
		return false
	}
	return true
}

func (cu *ComputeUnit) hasMoreWfsToRun() bool {
	for _, wfpool := range cu.WfPools {
		if len(wfpool.wfs) > 0 {
			return true
		}
	}
	return false
}

func (cu *ComputeUnit) wrapWG(
	raw *kernels.WorkGroup,
	req *gcn3.MapWGReq,
) *WorkGroup {
	wg := NewWorkGroup(raw, req)

	lds := make([]byte, wg.CodeObject().WGGroupSegmentByteSize)
	wg.LDS = lds

	cu.wgToManagedWgMapping[raw] = wg
	return wg
}

func (cu *ComputeUnit) wrapWf(raw *kernels.Wavefront) *Wavefront {
	wf := NewWavefront(raw)
	wg := cu.wgToManagedWgMapping[raw.WG]
	wg.Wfs = append(wg.Wfs, wf)
	wf.WG = wg
	wf.CodeObject = wf.WG.Grid.CodeObject
	wf.Packet = wf.WG.Grid.Packet
	wf.PacketAddress = wf.WG.Grid.PacketAddress
	return wf
}

func (cu *ComputeUnit) processInputFromInstMem(now akita.VTimeInSec) {
	rsp := cu.ToInstMem.Retrieve(now)
	if rsp == nil {
		return
	}
	cu.NeedTick = true

	switch rsp := rsp.(type) {
	case *mem.DataReadyRsp:
		cu.handleFetchReturn(now, rsp)
	default:
		log.Panicf("cannot handle request of type %s from ToInstMem port",
			reflect.TypeOf(rsp))
	}
}

func (cu *ComputeUnit) handleFetchReturn(now akita.VTimeInSec, rsp *mem.DataReadyRsp) {
	if len(cu.inFlightInstFetch) == 0 {
		log.Panic("CU is fetching no instruction")
	}

	info := cu.inFlightInstFetch[0]
	if info.Req.ID != rsp.RespondTo {
		log.Panic("response does not match request")
	}

	wf := info.Wavefront
	addr := info.Address
	cu.inFlightInstFetch = cu.inFlightInstFetch[1:]

	if addr == wf.InstBufferStartPC+uint64(len(wf.InstBuffer)) {
		wf.InstBuffer = append(wf.InstBuffer, rsp.Data...)
	}

	wf.IsFetching = false
	wf.LastFetchTime = now
}

func (cu *ComputeUnit) processInputFromScalarMem(now akita.VTimeInSec) {
	rsp := cu.ToScalarMem.Retrieve(now)
	if rsp == nil {
		return
	}
	cu.NeedTick = true

	switch rsp := rsp.(type) {
	case *mem.DataReadyRsp:
		cu.handleScalarDataLoadReturn(now, rsp)
	default:
		log.Panicf("cannot handle request of type %s from ToInstMem port",
			reflect.TypeOf(rsp))
	}
}

func (cu *ComputeUnit) handleScalarDataLoadReturn(
	now akita.VTimeInSec,
	rsp *mem.DataReadyRsp,
) {
	if len(cu.inFlightScalarMemAccess) == 0 {
		log.Panic("CU is not loading scalar data")
	}

	info := cu.inFlightScalarMemAccess[0]
	if info.Req.ID != rsp.RespondTo {
		log.Panic("response does not match request")
	}

	wf := info.Wavefront
	access := RegisterAccess{}
	access.WaveOffset = wf.SRegOffset
	access.Reg = info.DstSGPR
	access.RegCount = int(len(rsp.Data) / 4)
	access.Data = rsp.Data
	cu.SRegFile.Write(access)

	wf.OutstandingScalarMemAccess--
	cu.inFlightScalarMemAccess = cu.inFlightScalarMemAccess[1:]

	cu.InvokeHook(wf, cu, akita.AnyHookPos, &InstHookInfo{now, info.Inst, "Completed"})
}

func (cu *ComputeUnit) processInputFromVectorMem(now akita.VTimeInSec) {
	rsp := cu.ToVectorMem.Retrieve(now)
	if rsp == nil {
		return
	}
	cu.NeedTick = true

	switch rsp := rsp.(type) {
	case *mem.DataReadyRsp:
		cu.handleVectorDataLoadReturn(now, rsp)
	case *mem.DoneRsp:
		cu.handleVectorDataStoreRsp(now, rsp)
	default:
		log.Panicf("cannot handle request of type %s from ToInstMem port",
			reflect.TypeOf(rsp))
	}
}

func (cu *ComputeUnit) handleVectorDataLoadReturn(
	now akita.VTimeInSec,
	rsp *mem.DataReadyRsp,
) {
	if len(cu.inFlightVectorMemAccess) == 0 {
		log.Panic("CU is not accessing vector memory")
	}

	info := cu.inFlightVectorMemAccess[0]
	if info.Read.ID != rsp.RespondTo {
		log.Panic("CU cannot receive out of order memory return")
	}
	cu.inFlightVectorMemAccess = cu.inFlightVectorMemAccess[1:]

	wf := info.Wavefront
	inst := info.Inst

	for i, laneID := range info.Lanes {
		offset := info.LaneAddrOffsets[i]
		access := RegisterAccess{}
		access.WaveOffset = wf.VRegOffset
		access.Reg = info.DstVGPR
		access.RegCount = info.RegisterCount
		access.LaneID = laneID
		if inst.FormatType == insts.FLAT && inst.Opcode == 16 { // FLAT_LOAD_UBYTE
			access.Data = insts.Uint32ToBytes(uint32(rsp.Data[offset]))
		} else if inst.FormatType == insts.FLAT && inst.Opcode == 18 {
			access.Data = insts.Uint32ToBytes(uint32(rsp.Data[offset]))
		} else {
			access.Data = rsp.Data[offset : offset+uint64(4*info.RegisterCount)]
		}
		cu.VRegFile[wf.SIMDID].Write(access)
	}

	if info.Read.IsLastInWave {
		wf.OutstandingVectorMemAccess--
		if info.Inst.FormatType == insts.FLAT {
			wf.OutstandingScalarMemAccess--
		}
		cu.InvokeHook(wf, cu, akita.AnyHookPos,
			&InstHookInfo{now, info.Inst, "Completed"})
	}
}

func (cu *ComputeUnit) handleVectorDataStoreRsp(now akita.VTimeInSec, rsp *mem.DoneRsp) {
	info := cu.inFlightVectorMemAccess[0]
	if info.Write.ID != rsp.RespondTo {
		log.Panic("CU cannot receive out of order memory return")
	}
	cu.inFlightVectorMemAccess = cu.inFlightVectorMemAccess[1:]

	wf := info.Wavefront
	if info.Write.IsLastInWave {
		wf.OutstandingVectorMemAccess--
		if info.Inst.FormatType == insts.FLAT {
			wf.OutstandingScalarMemAccess--
		}
		cu.InvokeHook(wf, cu, akita.AnyHookPos,
			&InstHookInfo{now, info.Inst, "Completed"})
	}
}

// NewComputeUnit returns a newly constructed compute unit
func NewComputeUnit(
	name string,
	engine akita.Engine,
) *ComputeUnit {
	cu := new(ComputeUnit)
	cu.TickingComponent = akita.NewTickingComponent(
		name, engine, 1*akita.GHz, cu)

	cu.WfToDispatch = make(map[*kernels.Wavefront]*WfDispatchInfo)
	cu.wgToManagedWgMapping = make(map[*kernels.WorkGroup]*WorkGroup)

	cu.ToACE = akita.NewLimitNumReqPort(cu, 1)
	cu.ToInstMem = akita.NewLimitNumReqPort(cu, 1)
	cu.ToScalarMem = akita.NewLimitNumReqPort(cu, 1)
	cu.ToVectorMem = akita.NewLimitNumReqPort(cu, 1)

	return cu
}
