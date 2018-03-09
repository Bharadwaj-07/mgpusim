package timing

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SIMD Unit", func() {

	var (
		cu  *ComputeUnit
		bu  *SIMDUnit
		sp  *mockScratchpadPreparer
		alu *mockALU
	)

	BeforeEach(func() {
		cu = NewComputeUnit("cu", nil)
		sp = new(mockScratchpadPreparer)
		alu = new(mockALU)
		bu = NewSIMDUnit(cu, sp, alu)
	})

	It("should allow accepting wavefront", func() {
		// wave := new(Wavefront)
		bu.toRead = nil
		Expect(bu.CanAcceptWave()).To(BeTrue())
	})

	It("should not allow accepting wavefront is the read stage buffer is occupied", func() {
		bu.toRead = new(Wavefront)
		Expect(bu.CanAcceptWave()).To(BeFalse())
	})

	It("should accept wave", func() {
		wave := new(Wavefront)
		bu.AcceptWave(wave, 10)
		Expect(bu.toRead).To(BeIdenticalTo(wave))
	})

	It("should run", func() {
		wave1 := new(Wavefront)
		wave2 := new(Wavefront)
		wave3 := new(Wavefront)
		wave3.State = WfRunning

		bu.toRead = wave1
		bu.toExec = wave2
		bu.toWrite = wave3
		bu.execCycleLeft = 1

		bu.Run(10)

		Expect(wave3.State).To(Equal(WfReady))
		Expect(bu.toWrite).To(BeIdenticalTo(wave2))
		Expect(bu.toExec).To(BeIdenticalTo(wave1))
		Expect(bu.execCycleLeft).To(Equal(4))
		Expect(bu.toRead).To(BeNil())

		Expect(sp.wfPrepared).To(BeIdenticalTo(wave1))
		Expect(alu.wfExecuted).To(BeIdenticalTo(wave2))
		Expect(sp.wfCommitted).To(BeIdenticalTo(wave3))
	})

	It("should spend 4 cycles in execution", func() {
		wave1 := new(Wavefront)
		wave2 := new(Wavefront)
		wave3 := new(Wavefront)
		wave3.State = WfRunning

		bu.toRead = wave1
		bu.toExec = wave2
		bu.toWrite = wave3
		bu.execCycleLeft = 4

		bu.Run(10)

		Expect(wave3.State).To(Equal(WfReady))
		Expect(bu.toWrite).To(BeNil())
		Expect(bu.toExec).To(BeIdenticalTo(wave2))
		Expect(bu.execCycleLeft).To(Equal(3))
		Expect(bu.toRead).To(BeIdenticalTo(wave1))
	})
})
