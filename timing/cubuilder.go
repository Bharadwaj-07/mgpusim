package timing

import (
	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/core/util"
	"gitlab.com/yaotsu/gcn3/emu"
)

// A Builder can construct a fully functional ComputeUnit to the outside world.
// It simplify the compute unit building process.
type Builder struct {
	Engine    core.Engine
	Freq      util.Freq
	CUName    string
	SIMDCount int
	VGPRCount []int
	SGPRCount int

	Decoder            emu.Decoder
	ScratchpadPreparer ScratchpadPreparer
	ALU                emu.ALU

	InstMem   core.Component
	ScalarMem core.Component
	VectorMem core.Component

	ConnToInstMem   core.Connection
	ConnToScalarMem core.Connection
	ConnToVectorMem core.Connection
}

// NewBuilder returns a default builder object
func NewBuilder() *Builder {
	b := new(Builder)
	b.Freq = 800 * util.MHz
	b.SIMDCount = 4
	b.SGPRCount = 3200
	b.VGPRCount = []int{16384, 16384, 16384, 16384}

	return b
}

// Build returns a newly constructed compute unit according to the
// configuration
func (b *Builder) Build() *ComputeUnit {
	cu := NewComputeUnit(b.CUName, b.Engine)
	cu.Freq = b.Freq
	cu.Decoder = b.Decoder
	cu.WGMapper = NewWGMapper(cu, 4)
	cu.WfDispatcher = NewWfDispatcher(cu)

	b.ALU = emu.NewALUImpl(nil)
	b.ScratchpadPreparer = NewScratchpadPreparerImpl(cu)

	for i := 0; i < 4; i++ {
		cu.WfPools = append(cu.WfPools, NewWavefrontPool(10))
	}

	b.equipScheduler(cu)
	b.equipScalarUnits(cu)
	b.equipSIMDUnits(cu)
	b.equipLDSUnit(cu)
	b.equipVectorMemoryUnit(cu)
	b.equipRegisterFiles(cu)

	b.connectToMem(cu)

	return cu
}

func (b *Builder) equipScheduler(cu *ComputeUnit) {
	fetchArbitor := new(FetchArbiter)
	issueArbitor := new(IssueArbiter)
	scheduler := NewScheduler(cu, fetchArbitor, issueArbitor)
	cu.Scheduler = scheduler
}

func (b *Builder) equipScalarUnits(cu *ComputeUnit) {
	cu.BranchUnit = NewBranchUnit(cu, b.ScratchpadPreparer, b.ALU)

	scalarDecoder := NewDecodeUnit(cu)
	cu.ScalarDecoder = scalarDecoder
	scalarUnit := NewScalarUnit(cu, b.ScratchpadPreparer, b.ALU)
	cu.ScalarUnit = scalarUnit
	for i := 0; i < b.SIMDCount; i++ {
		scalarDecoder.AddExecutionUnit(scalarUnit)
	}
}

func (b *Builder) equipSIMDUnits(cu *ComputeUnit) {
	vectorDecoder := NewDecodeUnit(cu)
	cu.VectorDecoder = vectorDecoder
	for i := 0; i < b.SIMDCount; i++ {
		simdUnit := NewSIMDUnit(cu, b.ScratchpadPreparer, b.ALU)
		vectorDecoder.AddExecutionUnit(simdUnit)
		cu.SIMDUnit = append(cu.SIMDUnit, simdUnit)
	}
}

func (b *Builder) equipLDSUnit(cu *ComputeUnit) {
	ldsDecoder := NewDecodeUnit(cu)
	cu.LDSDecoder = ldsDecoder

	ldsUnit := NewLDSUnit(cu, b.ScratchpadPreparer, b.ALU)
	cu.LDSUnit = ldsUnit

	for i := 0; i < b.SIMDCount; i++ {
		ldsDecoder.AddExecutionUnit(ldsUnit)
	}
}

func (b *Builder) equipVectorMemoryUnit(cu *ComputeUnit) {
	vectorMemDecoder := NewDecodeUnit(cu)
	cu.VectorMemDecoder = vectorMemDecoder

	vectorMemoryUnit := NewVectorMemoryUnit(cu, b.ScratchpadPreparer)
	cu.VectorMemUnit = vectorMemoryUnit

	for i := 0; i < b.SIMDCount; i++ {
		vectorMemDecoder.AddExecutionUnit(vectorMemoryUnit)
	}
}

func (b *Builder) equipRegisterFiles(cu *ComputeUnit) {
	sRegFile := NewSimpleRegisterFile(uint64(b.SGPRCount*4), 0)
	cu.SRegFile = sRegFile

	for i := 0; i < b.SIMDCount; i++ {
		vRegFile := NewSimpleRegisterFile(uint64(b.VGPRCount[i]*4), 1024)
		cu.VRegFile = append(cu.VRegFile, vRegFile)
	}
}

func (b *Builder) connectToMem(cu *ComputeUnit) {
	cu.InstMem = b.InstMem
	cu.ScalarMem = b.ScalarMem
	cu.VectorMem = b.VectorMem
	core.PlugIn(cu, "ToInstMem", b.ConnToInstMem)
	core.PlugIn(cu, "ToScalarMem", b.ConnToScalarMem)
	core.PlugIn(cu, "ToVectorMem", b.ConnToVectorMem)
}
