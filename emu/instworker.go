package emu

import (
	"log"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3"
	"gitlab.com/yaotsu/gcn3/disasm"
)

// A InstWorker is where instructions got executed
type InstWorker interface {
	Run(wf *WfScheduleInfo, now core.VTimeInSec) error
	Continue(wf *WfScheduleInfo, now core.VTimeInSec) error
}

// InstWorkerImpl is the standard implmentation of a InstWorker
type InstWorkerImpl struct {
	CU        gcn3.ComputeUnit
	Scheduler *Scheduler

	DataMem   core.Component
	ToDataMem core.Connection
}

// Run will emulate the result of a instruction execution
func (w *InstWorkerImpl) Run(
	wf *WfScheduleInfo,
	now core.VTimeInSec,
) error {
	log.Printf("%.10f: Inst %s\n", now, wf.Inst.String())
	inst := wf.Inst
	switch inst.FormatType {
	case disasm.Sop2:
		return w.runSop2(wf, now)
	case disasm.Vop1:
		return w.runVop1(wf, now)
	case disasm.Flat:
		return w.runFlat(wf, now)
	default:
		log.Panicf("instruction type %s not supported\n", inst.FormatName)
	}
	return nil
}

// Continue picks up where instruction is stopped and continue its execution
func (w *InstWorkerImpl) Continue(
	wf *WfScheduleInfo,
	now core.VTimeInSec,
) error {
	inst := wf.Inst
	switch inst.FormatType {
	case disasm.Flat:
		return w.continueFlat(wf, now)
	default:
		log.Panicf("instruction type %s not supported\n", inst.FormatName)
	}
	return nil
}

func (w *InstWorkerImpl) getRegUint64(
	reg *disasm.Reg, wiFlatID int,
) uint64 {
	data := w.CU.ReadReg(reg, wiFlatID, 8)
	return disasm.BytesToUint64(data)
}

func (w *InstWorkerImpl) getRegUint32(
	reg *disasm.Reg, wiFlatID int,
) uint32 {
	data := w.CU.ReadReg(reg, wiFlatID, 4)
	return disasm.BytesToUint32(data)
}

func (w *InstWorkerImpl) getRegUint8(
	reg *disasm.Reg, wiFlatID int,
) uint8 {
	data := w.CU.ReadReg(reg, wiFlatID, 1)
	return disasm.BytesToUint8(data)
}

func (w *InstWorkerImpl) getOperandValueUint32(
	operand *disasm.Operand, wiFlatID int,
) uint32 {
	switch operand.OperandType {
	case disasm.RegOperand:
		return w.getRegUint32(operand.Register, wiFlatID)
	case disasm.IntOperand:
		return uint32(operand.IntValue)
	case disasm.LiteralConstant:
		return uint32(operand.LiteralConstant)
	default:
		log.Panic("invalid operand type")
	}
	return 0
}

func (w *InstWorkerImpl) putRegUint8(
	reg *disasm.Reg, wiFlatID int, value uint8,
) {
	data := disasm.Uint8ToBytes(value)
	w.CU.WriteReg(reg, wiFlatID, data)
}

func (w *InstWorkerImpl) putRegUint32(
	reg *disasm.Reg, wiFlatID int, value uint32,
) {
	data := disasm.Uint32ToBytes(value)
	w.CU.WriteReg(reg, wiFlatID, data)
}

func (w *InstWorkerImpl) putRegUint64(
	reg *disasm.Reg, wiFlatID int, value uint64,
) {
	data := disasm.Uint64ToBytes(value)
	w.CU.WriteReg(reg, wiFlatID, data)
}
