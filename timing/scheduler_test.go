package timing

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/insts"
	"gitlab.com/yaotsu/mem"
)

type mockWfArbitor struct {
	wfsToReturn [][]*Wavefront
}

func newMockWfArbitor() *mockWfArbitor {
	a := new(mockWfArbitor)
	a.wfsToReturn = make([][]*Wavefront, 0)
	return a
}

func (m *mockWfArbitor) Arbitrate([]*WavefrontPool) []*Wavefront {
	if len(m.wfsToReturn) == 0 {
		return nil
	}
	wfs := m.wfsToReturn[0]
	m.wfsToReturn = m.wfsToReturn[1:]
	return wfs
}

type mockCUComponent struct {
	canAccept    bool
	acceptedWave []*Wavefront
}

func (c *mockCUComponent) CanAcceptWave() bool {
	return c.canAccept
}

func (c *mockCUComponent) AcceptWave(wave *Wavefront) {
	c.acceptedWave = append(c.acceptedWave, wave)
}

func (c *mockCUComponent) Run(now core.VTimeInSec) {
}

var _ = Describe("Scheduler", func() {
	var (
		toInstMemConn    *core.MockConnection
		engine           *core.MockEngine
		cu               *ComputeUnit
		branchUnit       *mockCUComponent
		ldsDecoder       *mockCUComponent
		vectorMemDecoder *mockCUComponent
		vectorDecoder    *mockCUComponent
		scalarDecoder    *mockCUComponent
		scheduler        *Scheduler
		fetchArbitor     *mockWfArbitor
		issueArbitor     *mockWfArbitor
		instMem          *core.MockComponent
	)

	BeforeEach(func() {
		toInstMemConn = core.NewMockConnection()
		engine = core.NewMockEngine()
		cu = NewComputeUnit("cu", engine)
		cu.Freq = 1

		vectorDecoder = new(mockCUComponent)
		cu.VectorDecoder = vectorDecoder
		scalarDecoder = new(mockCUComponent)
		cu.ScalarDecoder = scalarDecoder
		branchUnit = new(mockCUComponent)
		cu.BranchUnit = branchUnit
		vectorMemDecoder = new(mockCUComponent)
		cu.VectorMemDecoder = vectorMemDecoder
		ldsDecoder = new(mockCUComponent)
		cu.LDSDecoder = ldsDecoder

		cu.InstMem = instMem
		core.PlugIn(cu, "ToInstMem", toInstMemConn)

		fetchArbitor = newMockWfArbitor()
		issueArbitor = newMockWfArbitor()
		scheduler = NewScheduler(cu, fetchArbitor, issueArbitor)
	})

	It("should fetch", func() {
		wf := new(Wavefront)
		wf.PC = 8064
		fetchArbitor.wfsToReturn = append(fetchArbitor.wfsToReturn,
			[]*Wavefront{wf})

		reqToExpect := mem.NewAccessReq()
		reqToExpect.SetSrc(cu)
		reqToExpect.SetDst(instMem)
		reqToExpect.Address = 8064
		reqToExpect.ByteSize = 8
		reqToExpect.Type = mem.Read
		reqToExpect.SetSendTime(10)
		reqToExpect.Info = wf
		toInstMemConn.ExpectSend(reqToExpect, nil)

		scheduler.DoFetch(10)

		Expect(toInstMemConn.AllExpectedSent()).To(BeTrue())
		Expect(wf.State).To(Equal(WfFetching))
	})

	It("should issue", func() {
		wfs := make([]*Wavefront, 0)
		issueDirs := []insts.ExeUnit{
			insts.ExeUnitBranch,
			insts.ExeUnitLDS,
			insts.ExeUnitVMem,
			insts.ExeUnitVALU,
			insts.ExeUnitScalar,
		}
		branchUnit.canAccept = true
		ldsDecoder.canAccept = true
		vectorDecoder.canAccept = true
		vectorMemDecoder.canAccept = true
		scalarDecoder.canAccept = false

		for i := 0; i < 5; i++ {
			wf := new(Wavefront)
			wf.State = WfFetched
			wf.Inst = NewInst(insts.NewInst())
			wf.Inst.ExeUnit = issueDirs[i]
			wfs = append(wfs, wf)
		}

		issueArbitor.wfsToReturn = append(issueArbitor.wfsToReturn, wfs)

		scheduler.DoIssue(10)

		Expect(len(branchUnit.acceptedWave)).To(Equal(1))
		Expect(len(ldsDecoder.acceptedWave)).To(Equal(1))
		Expect(len(vectorDecoder.acceptedWave)).To(Equal(1))
		Expect(len(vectorMemDecoder.acceptedWave)).To(Equal(1))
		Expect(len(scalarDecoder.acceptedWave)).To(Equal(0))

		Expect(wfs[0].State).To(Equal(WfRunning))
		Expect(wfs[1].State).To(Equal(WfRunning))
		Expect(wfs[2].State).To(Equal(WfRunning))
		Expect(wfs[3].State).To(Equal(WfRunning))
		Expect(wfs[4].State).To(Equal(WfFetched))
	})

	It("should issue internal instruction", func() {
		wfs := make([]*Wavefront, 0)
		wf := new(Wavefront)
		wf.Inst = NewInst(insts.NewInst())
		wf.Inst.ExeUnit = insts.ExeUnitSpecial
		wf.State = WfFetched
		wfs = append(wfs, wf)

		issueArbitor.wfsToReturn = append(issueArbitor.wfsToReturn, wfs)
		scheduler.internalExecuting = nil

		scheduler.DoIssue(10)

		Expect(scheduler.internalExecuting).To(BeIdenticalTo(wf))
		Expect(wf.State).To(Equal(WfRunning))
	})

	It("should evaluate internal executing insts", func() {
		wf := new(Wavefront)
		wf.Inst = NewInst(insts.NewInst())
		wf.Inst.Format = insts.FormatTable[insts.Sopp]
		wf.Inst.Opcode = 1 // S_ENDPGM

		scheduler.internalExecuting = wf
		scheduler.EvaluateInternalInst(10)

		Expect(len(engine.ScheduledEvent)).To(Equal(1))
	})

})
