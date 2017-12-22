package emu

import "log"

func (u *ALU) runSOPC(state InstEmuState) {
	inst := state.Inst()
	switch inst.Opcode {
	case 0:
		u.runSCMPEQU32(state)
	case 1:
		u.runSCMPLGU32(state)
	case 2:
		u.runSCMPGTI32(state)
	case 6:
		u.runSCMPEQU32(state)
	case 7:
		u.runSCMPLGU32(state)
	default:
		log.Panicf("Opcode %d for SOPC format is not implemented", inst.Opcode)
	}
}

func (u *ALU) runSCMPGTI32(state InstEmuState) {
	sp := state.Scratchpad().AsSOPC()
	src0 := asInt32(uint32(sp.SRC0))
	src1 := asInt32(uint32(sp.SRC1))
	if src0 > src1 {
		sp.SCC = 1
	} else {
		sp.SCC = 0
	}
}

func (u *ALU) runSCMPEQU32(state InstEmuState) {
	sp := state.Scratchpad().AsSOPC()
	if sp.SRC0 == sp.SRC1 {
		sp.SCC = 1
	} else {
		sp.SCC = 0
	}
}

func (u *ALU) runSCMPLGU32(state InstEmuState) {
	sp := state.Scratchpad().AsSOPC()
	if sp.SRC0 != sp.SRC1 {
		sp.SCC = 1
	} else {
		sp.SCC = 0
	}
}
