package timing

import (
	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3/emu"
)

// A SIMDUnit performs branch operations
type SIMDUnit struct {
	cu *ComputeUnit

	scratchpadPreparer ScratchpadPreparer
	alu                emu.ALU

	toExec    *Wavefront
	cycleLeft int

	NumSinglePrecisionUnit int
}

// NewSIMDUnit creates a new branch unit, injecting the dependency of
// the compute unit.
func NewSIMDUnit(
	cu *ComputeUnit,
	scratchpadPreparer ScratchpadPreparer,
	alu emu.ALU,
) *SIMDUnit {
	u := new(SIMDUnit)
	u.cu = cu
	u.scratchpadPreparer = scratchpadPreparer
	u.alu = alu

	u.NumSinglePrecisionUnit = 32

	return u
}

// CanAcceptWave checks if the buffer of the read stage is occupied or not
func (u *SIMDUnit) CanAcceptWave() bool {
	return u.toExec == nil
}

// AcceptWave moves one wavefront into the read buffer of the branch unit
func (u *SIMDUnit) AcceptWave(wave *Wavefront, now akita.VTimeInSec) {
	u.toExec = wave

	// The cycle left if calculated. The pipeline is like the following
	//
	// Lane 00-31 r e w
	// Lane 32-64   r e w
	//
	// The total number of cycles is the execution time plus the first read
	// and the last write.
	u.cycleLeft = 64/u.NumSinglePrecisionUnit + 2

	u.cu.InvokeHook(u.toExec, u.cu, akita.AnyHookPos, &InstHookInfo{now, u.toExec.inst, "Exec"})
}

// Run executes three pipeline stages that are controlled by the SIMDUnit
func (u *SIMDUnit) Run(now akita.VTimeInSec) bool {
	return u.runExecStage(now)
}

func (u *SIMDUnit) runExecStage(now akita.VTimeInSec) bool {
	if u.toExec == nil {
		return false
	}

	u.cycleLeft--
	if u.cycleLeft > 0 {
		return true
	}

	u.scratchpadPreparer.Prepare(u.toExec, u.toExec)
	u.alu.Run(u.toExec)
	u.scratchpadPreparer.Commit(u.toExec, u.toExec)
	u.toExec.State = WfReady
	u.cu.InvokeHook(u.toExec, u.cu, akita.AnyHookPos, &InstHookInfo{now, u.toExec.inst, "Completed"})

	u.toExec = nil
	return true
}
