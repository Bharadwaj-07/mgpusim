package timing

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("Vector Memory Unit", func() {

	// var (
	// 	cu        *ComputeUnit
	// 	sp        *mockScratchpadPreparer
	// 	coalescer *MockCoalescer
	// 	bu        *VectorMemoryUnit
	// 	vectorMem *akita.MockComponent
	// 	conn      *akita.MockConnection
	// )

	// BeforeEach(func() {
	// 	cu = NewComputeUnit("cu", nil)
	// 	sp = new(mockScratchpadPreparer)
	// 	coalescer = new(MockCoalescer)
	// 	bu = NewVectorMemoryUnit(cu, sp, coalescer)
	// 	vectorMem = akita.NewMockComponent("VectorMem")
	// 	conn = akita.NewMockConnection()

	// 	cu.VectorMemModules = new(cache.SingleLowModuleFinder)
	// 	conn.PlugIn(cu.ToVectorMem)
	// })

	// It("should allow accepting wavefront", func() {
	// 	bu.toRead = nil
	// 	Expect(bu.CanAcceptWave()).To(BeTrue())
	// })

	// It("should not allow accepting wavefront is the read stage buffer is occupied", func() {
	// 	bu.toRead = new(Wavefront)
	// 	Expect(bu.CanAcceptWave()).To(BeFalse())
	// })

	// It("should accept wave", func() {
	// 	wave := new(Wavefront)
	// 	bu.AcceptWave(wave, 10)
	// 	Expect(bu.toRead).To(BeIdenticalTo(wave))
	// })

	// It("should read", func() {
	// 	wave := new(Wavefront)
	// 	bu.toRead = wave

	// 	madeProgress := bu.runReadStage(10)

	// 	Expect(madeProgress).To(BeTrue())
	// 	Expect(bu.toExec).To(BeIdenticalTo(wave))
	// 	Expect(bu.toRead).To(BeNil())
	// 	Expect(bu.AddrCoalescingCycleLeft).To(Equal(bu.AddrCoalescingLatency))
	// })

	// It("should reduce cycle left when executing", func() {
	// 	wave := new(Wavefront)
	// 	bu.toExec = wave
	// 	bu.AddrCoalescingCycleLeft = 40

	// 	madeProgress := bu.runExecStage(10)

	// 	Expect(madeProgress).To(BeTrue())
	// 	Expect(bu.toExec).To(BeIdenticalTo(wave))
	// 	Expect(bu.AddrCoalescingCycleLeft).To(Equal(39))
	// })

	// It("should run flat_load_dword", func() {
	// 	wave := NewWavefront(nil)
	// 	inst := NewInst(insts.NewInst())
	// 	inst.Format = insts.FormatTable[insts.FLAT]
	// 	inst.Opcode = 20
	// 	inst.Dst = insts.NewVRegOperand(0, 0, 1)
	// 	wave.inst = inst

	// 	coalescer.ToReturn = make([]CoalescedAccess, 4)
	// 	for i := 0; i < 4; i++ {
	// 		coalescer.ToReturn[i].Addr = uint64(0x40 * i)
	// 		coalescer.ToReturn[i].Size = 64
	// 		for j := 0; j < 16; j++ {
	// 			coalescer.ToReturn[i].LaneIDs =
	// 				append(coalescer.ToReturn[i].LaneIDs, i*16+j)
	// 		}
	// 	}

	// 	bu.toExec = wave

	// 	bu.Run(10)

	// 	Expect(wave.State).To(Equal(WfReady))
	// 	Expect(wave.OutstandingVectorMemAccess).To(Equal(1))
	// 	Expect(wave.OutstandingScalarMemAccess).To(Equal(1))
	// 	Expect(cu.inFlightVectorMemAccess).To(HaveLen(4))
	// 	Expect(cu.inFlightVectorMemAccess[3].Read.IsLastInWave).To(BeTrue())
	// 	Expect(bu.SendBuf).To(HaveLen(4))
	// })

	// It("should run flat_store_dword", func() {
	// 	wave := NewWavefront(nil)
	// 	inst := NewInst(insts.NewInst())
	// 	inst.Format = insts.FormatTable[insts.FLAT]
	// 	inst.Opcode = 28
	// 	inst.Dst = insts.NewVRegOperand(0, 0, 1)
	// 	wave.inst = inst

	// 	sp := wave.Scratchpad().AsFlat()
	// 	for i := 0; i < 64; i++ {
	// 		sp.ADDR[i] = uint64(4096 + i*4)
	// 		sp.DATA[i*4] = uint32(i)
	// 	}
	// 	sp.EXEC = 0xffffffffffffffff

	// 	bu.toExec = wave

	// 	bu.Run(10)

	// 	Expect(wave.State).To(Equal(WfReady))
	// 	Expect(wave.OutstandingVectorMemAccess).To(Equal(1))
	// 	Expect(wave.OutstandingScalarMemAccess).To(Equal(1))
	// 	Expect(cu.inFlightVectorMemAccess).To(HaveLen(64))
	// 	Expect(cu.inFlightVectorMemAccess[63].Write.IsLastInWave).To(BeTrue())
	// 	Expect(bu.SendBuf).To(HaveLen(64))
	// })

	// It("should send memory access requests", func() {
	// 	loadReq := mem.NewReadReq(10, cu.ToVectorMem, vectorMem.ToOutside, 0, 4)
	// 	loadReq.SetSendTime(10)
	// 	bu.SendBuf = append(bu.SendBuf, loadReq)

	// 	conn.ExpectSend(loadReq, nil)

	// 	bu.Run(10)

	// 	Expect(conn.AllExpectedSent()).To(BeTrue())
	// 	Expect(len(bu.SendBuf)).To(Equal(0))
	// })

	// It("should not remove request from read buffer, if send fails", func() {
	// 	loadReq := mem.NewReadReq(10, cu.ToVectorMem, vectorMem.ToOutside, 0, 4)
	// 	bu.SendBuf = append(bu.SendBuf, loadReq)

	// 	err := akita.NewSendError()
	// 	conn.ExpectSend(loadReq, err)

	// 	bu.Run(10)

	// 	Expect(conn.AllExpectedSent()).To(BeTrue())
	// 	Expect(len(bu.SendBuf)).To(Equal(1))
	// })

})
