package emu

import (
	"fmt"
	"log"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3/insts"
)

// ISADebugger is a hook that hooks to a emulator computeunit for each intruction
type ISADebugger struct {
	akita.LogHookBase

	prevWf *Wavefront
}

// NewISADebugger returns a new ISADebugger that keeps instruction log in logger
func NewISADebugger(logger *log.Logger) *ISADebugger {
	h := new(ISADebugger)
	h.Logger = logger
	return h
}

// Func defines the behavior of the tracer when the tracer is invoked.
func (h *ISADebugger) Func(ctx akita.HookCtx) {
	wf, ok := ctx.Item.(*Wavefront)
	if !ok {
		return
	}

	// For debugging
	if wf.FirstWiFlatID != 0 {
		return
	}

	h.logWholeWf(wf)
	// if h.prevWf == nil || h.prevWf.FirstWiFlatID != wf.FirstWiFlatID {
	// 	h.logWholeWf(wf)
	// } else {
	// 	h.logDiffWf(wf)
	// }

	h.stubWf(wf)
}

func (h *ISADebugger) logWholeWf(wf *Wavefront) {
	output := fmt.Sprintf("\n\twg - (%d, %d, %d), wf - %d\n",
		wf.WG.IDX, wf.WG.IDY, wf.WG.IDZ, wf.FirstWiFlatID)
	output += fmt.Sprintf("\tInst: %s\n", wf.Inst().String(nil))
	output += fmt.Sprintf("\tPC: 0x%016x\n", wf.PC)
	output += fmt.Sprintf("\tEXEC: 0x%016x\n", wf.Exec)
	output += fmt.Sprintf("\tSCC: 0x%02x\n", wf.SCC)
	output += fmt.Sprintf("\tVCC: 0x%016x\n", wf.VCC)

	output += "\tSGPRs:\n"
	for i := 0; i < int(wf.CodeObject.WFSgprCount); i++ {
		regValue := insts.BytesToUint32(wf.ReadReg(insts.SReg(i), 1, 0))
		output += fmt.Sprintf("\t\ts%d: 0x%08x\n", i, regValue)
	}

	output += "\tVGPRs: \n"
	for i := 0; i < int(wf.CodeObject.WIVgprCount); i++ {
		output += fmt.Sprintf("\t\tv%d: ", i)
		for laneID := 0; laneID < 64; laneID++ {
			regValue := insts.BytesToUint32(wf.ReadReg(insts.VReg(i), 1, laneID))
			output += fmt.Sprintf("0x%08x ", regValue)
		}
		output += fmt.Sprintf("\n")
	}

	h.Logger.Print(output)
}

//nolint:gocyclo
func (h *ISADebugger) logDiffWf(wf *Wavefront) {
	output := fmt.Sprintf("\n\twg - (%d, %d, %d), wf - %d\n",
		wf.WG.IDX, wf.WG.IDY, wf.WG.IDZ, wf.FirstWiFlatID)
	output += fmt.Sprintf("\tInst: %s\n", wf.Inst().String(nil))
	if wf.Exec != h.prevWf.Exec {
		output += fmt.Sprintf("\tEXEC: 0x%016x\n", wf.Exec)
	}

	if wf.SCC != h.prevWf.SCC {
		output += fmt.Sprintf("\tSCC: 0x%02x\n", wf.SCC)
	}

	if wf.VCC != h.prevWf.VCC {
		output += fmt.Sprintf("\tVCC: 0x%016x\n", wf.VCC)
	}

	output += "\tSGPRs:\n"
	for i := 0; i < int(wf.CodeObject.WFSgprCount); i++ {
		regValue := insts.BytesToUint32(wf.ReadReg(insts.SReg(i), 1, 0))
		prevRegValue := insts.BytesToUint32(h.prevWf.ReadReg(insts.SReg(i), 1, 0))
		if regValue != prevRegValue {
			output += fmt.Sprintf("\t\ts%d: 0x%08x\n", i, regValue)
		}
	}

	output += "\tVGPRs: \n"
	for i := 0; i < int(wf.CodeObject.WIVgprCount); i++ {
		updated := false
		for laneID := 0; laneID < 64; laneID++ {
			regValue := insts.BytesToUint32(wf.ReadReg(insts.VReg(i), 1, laneID))
			prevRegValue := insts.BytesToUint32(h.prevWf.ReadReg(insts.VReg(i), 1, laneID))
			if regValue != prevRegValue {
				updated = true
				break
			}
		}

		if updated {
			output += fmt.Sprintf("\t\tv%d: ", i)
			for laneID := 0; laneID < 64; laneID++ {
				regValue := insts.BytesToUint32(wf.ReadReg(insts.VReg(i), 1, laneID))
				output += fmt.Sprintf("0x%08x ", regValue)
			}
			output += fmt.Sprintf("\n")
		}
	}

	h.Logger.Print(output)
}

func (h *ISADebugger) stubWf(wf *Wavefront) {
	h.prevWf = NewWavefront(wf.Wavefront)

	h.prevWf.SRegFile = make([]byte, len(wf.SRegFile))
	copy(h.prevWf.SRegFile, wf.SRegFile)

	h.prevWf.VRegFile = make([]byte, len(wf.VRegFile))
	copy(h.prevWf.VRegFile, wf.VRegFile)

	h.prevWf.SCC = wf.SCC
	h.prevWf.VCC = wf.VCC
	h.prevWf.Exec = wf.Exec
}
