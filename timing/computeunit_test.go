package timing

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3"
	"gitlab.com/yaotsu/gcn3/insts"
	"gitlab.com/yaotsu/gcn3/kernels"
	"gitlab.com/yaotsu/mem"
)

type mockWGMapper struct {
	OK         bool
	UnmappedWg *WorkGroup
}

func (m *mockWGMapper) MapWG(req *gcn3.MapWGReq) bool {
	return m.OK
}

func (m *mockWGMapper) UnmapWG(wg *WorkGroup) {
	m.UnmappedWg = wg
}

type mockWfDispatcher struct {
	dispatchedWf *gcn3.DispatchWfReq
}

func (m *mockWfDispatcher) DispatchWf(wf *Wavefront, req *gcn3.DispatchWfReq) {
	m.dispatchedWf = req
}

type mockDecoder struct {
	Inst *insts.Inst
}

func (d *mockDecoder) Decode(buf []byte) (*insts.Inst, error) {
	return d.Inst, nil
}

func exampleGrid() *kernels.Grid {
	grid := kernels.NewGrid()

	grid.CodeObject = insts.NewHsaCo()
	grid.CodeObject.HsaCoHeader = new(insts.HsaCoHeader)

	packet := new(kernels.HsaKernelDispatchPacket)
	grid.Packet = packet

	wg := kernels.NewWorkGroup()
	wg.Grid = grid
	grid.WorkGroups = append(grid.WorkGroups, wg)

	wf := kernels.NewWavefront()
	wf.WG = wg
	wg.Wavefronts = append(wg.Wavefronts, wf)

	return grid
}

var _ = Describe("ComputeUnit", func() {
	var (
		cu           *ComputeUnit
		engine       *core.MockEngine
		wgMapper     *mockWGMapper
		wfDispatcher *mockWfDispatcher
		decoder      *mockDecoder

		connection *core.MockConnection
		instMem    *core.MockComponent

		grid *kernels.Grid
	)

	BeforeEach(func() {
		engine = core.NewMockEngine()
		wgMapper = new(mockWGMapper)
		wfDispatcher = new(mockWfDispatcher)
		decoder = new(mockDecoder)

		cu = NewComputeUnit("cu", engine)
		cu.WGMapper = wgMapper
		cu.WfDispatcher = wfDispatcher
		cu.Decoder = decoder
		cu.Freq = 1
		cu.SRegFile = NewSimpleRegisterFile(1024, 0)
		cu.VRegFile = append(cu.VRegFile, NewSimpleRegisterFile(4096, 64))

		for i := 0; i < 4; i++ {
			cu.WfPools = append(cu.WfPools, NewWavefrontPool(10))
		}

		connection = core.NewMockConnection()
		core.PlugIn(cu, "ToACE", connection)

		instMem = core.NewMockComponent("InstMem")
		cu.InstMem = instMem

		grid = exampleGrid()
	})

	Context("when processing MapWGReq", func() {
		It("should reply OK if mapping is successful", func() {
			wgMapper.OK = true

			wg := grid.WorkGroups[0]
			req := gcn3.NewMapWGReq(nil, cu, 10, wg)
			req.SetRecvTime(10)

			expectedResponse := gcn3.NewMapWGReq(cu, nil, 10, wg)
			expectedResponse.Ok = true
			expectedResponse.SetRecvTime(10)
			connection.ExpectSend(expectedResponse, nil)

			cu.Handle(req)

			Expect(connection.AllExpectedSent()).To(BeTrue())
		})

		It("should reply not OK if there are pending wavefronts", func() {
			wf := grid.WorkGroups[0].Wavefronts[0]
			cu.WfToDispatch[wf] = new(WfDispatchInfo)

			wg := grid.WorkGroups[0]
			req := gcn3.NewMapWGReq(nil, cu, 10, wg)
			req.SetRecvTime(10)

			expectedResponse := gcn3.NewMapWGReq(cu, nil, 10, wg)
			expectedResponse.Ok = false
			expectedResponse.SetRecvTime(10)
			connection.ExpectSend(expectedResponse, nil)

			cu.Handle(req)

			Expect(connection.AllExpectedSent()).To(BeTrue())
		})

		It("should reply not OK if mapping is failed", func() {
			wgMapper.OK = false

			wg := grid.WorkGroups[0]
			req := gcn3.NewMapWGReq(nil, cu, 10, wg)
			req.SetRecvTime(10)

			expectedResponse := gcn3.NewMapWGReq(cu, nil, 10, wg)
			expectedResponse.Ok = false
			expectedResponse.SetRecvTime(10)
			connection.ExpectSend(expectedResponse, nil)

			cu.Handle(req)

			Expect(connection.AllExpectedSent()).To(BeTrue())
		})
	})

	Context("when processing DispatchWfReq", func() {
		It("should dispatch wf", func() {
			wg := grid.WorkGroups[0]
			cu.wrapWG(wg, nil)

			wf := wg.Wavefronts[0]
			req := gcn3.NewDispatchWfReq(nil, cu, 10, wf)
			req.SetRecvTime(11)

			cu.Handle(req)

			Expect(wfDispatcher.dispatchedWf).To(BeIdenticalTo(req))
		})

		It("should handle WfDispatchCompletionEvent", func() {
			cu.running = false
			wf := grid.WorkGroups[0].Wavefronts[0]
			managedWf := new(Wavefront)
			managedWf.Wavefront = wf
			managedWf.State = WfDispatching

			info := new(WfDispatchInfo)
			info.Wavefront = wf
			info.SIMDID = 0
			cu.WfToDispatch[wf] = info

			req := gcn3.NewDispatchWfReq(nil, cu, 10, wf)
			evt := NewWfDispatchCompletionEvent(11, cu, managedWf)
			evt.DispatchWfReq = req

			expectedResponse := gcn3.NewDispatchWfReq(cu, nil, 11, wf)
			expectedResponse.SetSendTime(11)
			connection.ExpectSend(expectedResponse, nil)

			cu.Handle(evt)

			Expect(len(engine.ScheduledEvent)).To(Equal(1))
			Expect(connection.AllExpectedSent()).To(BeTrue())
			Expect(len(cu.WfPools[0].wfs)).To(Equal(1))
			Expect(len(cu.WfToDispatch)).To(Equal(0))
			Expect(managedWf.State).To(Equal(WfReady))
			Expect(cu.running).To(BeTrue())
		})
	})

	Context("when handling mem.AccessReq", func() {
		It("should handle fetch return", func() {
			wf := new(Wavefront)
			inst := NewInst(nil)
			wf.inst = inst
			wf.PC = 0x1000

			req := mem.NewAccessReq()
			req.SetSrc(instMem)
			req.SetDst(cu)
			req.SetRecvTime(10)
			req.Type = mem.Read
			req.ByteSize = 4
			info := new(MemAccessInfo)
			info.Action = MemAccessInstFetch
			info.Wf = wf
			req.Info = info

			rawInst := insts.NewInst()
			decoder.Inst = rawInst
			decoder.Inst.ByteSize = 4

			cu.Handle(req)

			Expect(wf.State).To(Equal(WfFetched))
			Expect(wf.LastFetchTime).To(BeNumerically("~", 10))
			Expect(wf.PC).To(Equal(uint64(0x1004)))
			Expect(wf.inst).To(BeIdenticalTo(inst))
			Expect(wf.inst.Inst).To(BeIdenticalTo(rawInst))
		})

		It("should handle scalar data load return", func() {
			rawWf := grid.WorkGroups[0].Wavefronts[0]
			inst := NewInst(insts.NewInst())
			wf := NewWavefront(rawWf)
			wf.inst = inst
			wf.SRegOffset = 0
			wf.OutstandingScalarMemAccess = 1

			info := new(MemAccessInfo)
			info.Action = MemAccessScalarDataLoad
			info.Wf = wf
			info.Dst = insts.SReg(0)

			req := mem.NewAccessReq()
			req.Info = info
			req.Buf = insts.Uint32ToBytes(32)
			req.SetSendTime(10)

			cu.Handle(req)

			access := new(RegisterAccess)
			access.Reg = insts.SReg(0)
			access.WaveOffset = 0
			access.RegCount = 1
			cu.SRegFile.Read(access)
			Expect(insts.BytesToUint32(access.Data)).To(Equal(uint32(32)))
			Expect(wf.OutstandingScalarMemAccess).To(Equal(0))
		})

		It("should handle vector data load return, and the return is not the last one for an instruction", func() {
			rawWf := grid.WorkGroups[0].Wavefronts[0]
			inst := NewInst(insts.NewInst())
			wf := NewWavefront(rawWf)
			wf.SIMDID = 0
			wf.inst = inst
			wf.VRegOffset = 0
			wf.OutstandingVectorMemAccess = 1

			info := new(MemAccessInfo)
			info.Action = MemAccessVectorDataLoad
			info.Wf = wf
			info.TotalReqs = 4
			info.ReturnedReqs = 1
			info.Inst = inst
			info.Dst = insts.VReg(0)
			for i := 0; i < 64; i++ {
				info.PreCoalescedAddrs[i] = uint64(4096 + i*4)
			}
			req := mem.NewAccessReq()
			req.Info = info
			req.Address = 4096
			req.ByteSize = 64
			req.Buf = make([]byte, 64)
			for i := 0; i < 16; i++ {
				copy(req.Buf[i*4:i*4+4], insts.Uint32ToBytes(uint32(i)))
			}

			cu.Handle(req)

			Expect(info.ReturnedReqs).To(Equal(2))
			for i := 0; i < 16; i++ {
				access := new(RegisterAccess)
				access.RegCount = 1
				access.WaveOffset = 0
				access.LaneID = i
				access.Reg = insts.VReg(0)
				cu.VRegFile[0].Read(access)
				Expect(insts.BytesToUint32(access.Data)).To(Equal(uint32(i)))
			}

		})

		It("should handle vector data load return, and the return is the last one for an instruction", func() {
			rawWf := grid.WorkGroups[0].Wavefronts[0]
			inst := NewInst(insts.NewInst())
			wf := NewWavefront(rawWf)
			wf.SIMDID = 0
			wf.inst = inst
			wf.VRegOffset = 0
			wf.OutstandingVectorMemAccess = 1

			info := new(MemAccessInfo)
			info.Action = MemAccessVectorDataLoad
			info.Wf = wf
			info.TotalReqs = 4
			info.ReturnedReqs = 3
			info.Inst = inst
			info.Dst = insts.VReg(0)
			for i := 0; i < 64; i++ {
				info.PreCoalescedAddrs[i] = uint64(4096 + i*4)
			}
			req := mem.NewAccessReq()
			req.Info = info
			req.Address = 4096 + 64*3
			req.ByteSize = 64
			req.Buf = make([]byte, 64)
			for i := 0; i < 16; i++ {
				copy(req.Buf[i*4:i*4+4], insts.Uint32ToBytes(uint32(i+48)))
			}

			cu.Handle(req)

			Expect(info.ReturnedReqs).To(Equal(4))
			Expect(wf.OutstandingVectorMemAccess).To(Equal(0))
			for i := 48; i < 64; i++ {
				access := new(RegisterAccess)
				access.RegCount = 1
				access.WaveOffset = 0
				access.LaneID = i
				access.Reg = insts.VReg(0)
				cu.VRegFile[0].Read(access)
				Expect(insts.BytesToUint32(access.Data)).To(Equal(uint32(i)))
			}
		})

		It("should handle vector data store return and the return is not the last one from an instruction", func() {

			rawWf := grid.WorkGroups[0].Wavefronts[0]
			inst := NewInst(insts.NewInst())
			wf := NewWavefront(rawWf)
			wf.SIMDID = 0
			wf.inst = inst
			wf.VRegOffset = 0
			wf.OutstandingVectorMemAccess = 1

			info := new(MemAccessInfo)
			info.Action = MemAccessVectorDataStore
			info.Wf = wf
			info.TotalReqs = 4
			info.ReturnedReqs = 1
			info.Inst = inst
			info.Dst = insts.VReg(0)
			req := mem.NewAccessReq()
			req.Info = info
			req.Address = 4096 + 64*3
			req.ByteSize = 64

			cu.Handle(req)

			Expect(info.ReturnedReqs).To(Equal(2))
		})

		It("should handle vector data store return and the return is the last one from an instruction", func() {

			rawWf := grid.WorkGroups[0].Wavefronts[0]
			inst := NewInst(insts.NewInst())
			wf := NewWavefront(rawWf)
			wf.SIMDID = 0
			wf.inst = inst
			wf.VRegOffset = 0
			wf.OutstandingVectorMemAccess = 1

			info := new(MemAccessInfo)
			info.Action = MemAccessVectorDataStore
			info.Wf = wf
			info.TotalReqs = 4
			info.ReturnedReqs = 3
			info.Inst = inst
			info.Dst = insts.VReg(0)
			req := mem.NewAccessReq()
			req.Info = info
			req.Address = 4096 + 64*3
			req.ByteSize = 64

			cu.Handle(req)

			Expect(info.ReturnedReqs).To(Equal(4))
			Expect(wf.OutstandingVectorMemAccess).To(Equal(0))
		})
	})

})
