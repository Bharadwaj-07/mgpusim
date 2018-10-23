package timing

import (
	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3/emu"
)

// A LDSUnit performs Scalar operations
type LDSUnit struct {
	cu *ComputeUnit

	scratchpadPreparer ScratchpadPreparer
	alu                emu.ALU

	toRead  *Wavefront
	toExec  *Wavefront
	toWrite *Wavefront
}

// NewLDSUnit creates a new Scalar unit, injecting the dependency of
// the compute unit.
func NewLDSUnit(
	cu *ComputeUnit,
	scratchpadPreparer ScratchpadPreparer,
	alu emu.ALU,
) *LDSUnit {
	u := new(LDSUnit)
	u.cu = cu
	u.scratchpadPreparer = scratchpadPreparer
	u.alu = alu
	return u
}

// CanAcceptWave checks if the buffer of the read stage is occupied or not
func (u *LDSUnit) CanAcceptWave() bool {
	return u.toRead == nil
}

// AcceptWave moves one wavefront into the read buffer of the Scalar unit
func (u *LDSUnit) AcceptWave(wave *Wavefront, now akita.VTimeInSec) {
	u.toRead = wave
	u.cu.InvokeHook(u.toRead, u.cu, akita.AnyHookPos, &InstHookInfo{now, u.toRead.inst, "Read"})
}

// Run executes three pipeline stages that are controlled by the LDSUnit
func (u *LDSUnit) Run(now akita.VTimeInSec) {
	u.runWriteStage(now)
	u.runExecStage(now)
	u.runReadStage(now)
}

func (u *LDSUnit) runReadStage(now akita.VTimeInSec) {
	if u.toRead == nil {
		return
	}

	if u.toExec == nil {
		u.scratchpadPreparer.Prepare(u.toRead, u.toRead)
		u.cu.InvokeHook(u.toRead, u.cu, akita.AnyHookPos, &InstHookInfo{now, u.toRead.inst, "Exec"})

		u.toExec = u.toRead
		u.toRead = nil
	}
}

func (u *LDSUnit) runExecStage(now akita.VTimeInSec) {
	if u.toExec == nil {
		return
	}

	if u.toWrite == nil {
		u.alu.SetLDS(u.toExec.WG.LDS)
		u.alu.Run(u.toExec)
		u.cu.InvokeHook(u.toExec, u.cu, akita.AnyHookPos, &InstHookInfo{now, u.toExec.inst, "Write"})

		u.toWrite = u.toExec
		u.toExec = nil
	}
}

func (u *LDSUnit) runWriteStage(now akita.VTimeInSec) {
	if u.toWrite == nil {
		return
	}

	u.scratchpadPreparer.Commit(u.toWrite, u.toWrite)

	u.cu.InvokeHook(u.toWrite, u.cu, akita.AnyHookPos, &InstHookInfo{now, u.toWrite.inst, "Completed"})

	u.toWrite.State = WfReady
	u.toWrite = nil
}
