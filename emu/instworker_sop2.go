package emu

import (
	"log"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/disasm"
)

func (w *InstWorkerImpl) runSop2(
	wf *WfScheduleInfo,
	now core.VTimeInSec,
) error {
	inst := wf.Inst
	switch inst.Opcode {
	case 0:
		return w.runSADDU32(wf, now)
	case 4:
		return w.runSAddCU32(wf, now)
	default:
		log.Panicf("instruction opcode %d for type sop2 not supported\n",
			inst.Opcode)
	}
	return nil
}

func (w *InstWorkerImpl) runSADDU32(
	wf *WfScheduleInfo,
	now core.VTimeInSec,
) error {
	inst := wf.Inst
	pc := w.getRegUint64(disasm.Regs[disasm.Pc], wf.Wf.FirstWiFlatID)
	src1Value := w.getOperandValueUint32(inst.Src1, wf.Wf.FirstWiFlatID)
	src0Value := w.getOperandValueUint32(inst.Src0, wf.Wf.FirstWiFlatID)
	var sccValue uint8
	if src1Value&(1<<31) != 0 && src0Value&(1<<31) != 0 {
		sccValue = 1
	}
	dstValue := src0Value + src1Value
	pc += uint64(inst.ByteSize)
	w.putRegUint32(inst.Dst.Register, wf.Wf.FirstWiFlatID, dstValue)
	w.putRegUint64(disasm.Regs[disasm.Pc], wf.Wf.FirstWiFlatID, pc)
	w.putRegUint8(disasm.Regs[disasm.Scc], wf.Wf.FirstWiFlatID, sccValue)

	w.Scheduler.Completed(wf)
	return nil
}

func (w *InstWorkerImpl) runSAddCU32(
	wf *WfScheduleInfo,
	now core.VTimeInSec,
) error {
	inst := wf.Inst
	pc := w.getRegUint64(disasm.Regs[disasm.Pc], wf.Wf.FirstWiFlatID)
	src1Value := w.getOperandValueUint32(inst.Src1, wf.Wf.FirstWiFlatID)
	src0Value := w.getOperandValueUint32(inst.Src0, wf.Wf.FirstWiFlatID)
	sccValue := w.getRegUint8(disasm.Regs[disasm.Scc], wf.Wf.FirstWiFlatID)
	dstValue := src0Value + src1Value + uint32(sccValue)
	if src1Value&(1<<31) != 0 && src0Value&(1<<31) != 0 {
		sccValue = 1
	}
	pc += uint64(inst.ByteSize)
	w.putRegUint32(inst.Dst.Register, wf.Wf.FirstWiFlatID, dstValue)
	w.putRegUint64(disasm.Regs[disasm.Pc], wf.Wf.FirstWiFlatID, pc)
	w.putRegUint8(disasm.Regs[disasm.Scc], wf.Wf.FirstWiFlatID, sccValue)
	w.Scheduler.Completed(wf)
	return nil
}
