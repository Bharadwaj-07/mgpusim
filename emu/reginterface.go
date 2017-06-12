package emu

import (
	"log"

	"gitlab.com/yaotsu/gcn3/insts"
)

// RegInterface wraps the interface of communication between any units
// to the register files.
//
// The emulator implements a very simple version of this interface. It
// read and write the register directly from the register of the wavefront.
//
type RegInterface interface {
	// ReadReg reads from the register. Parameter wf specifies which wavefront
	// is reading, reg specifies the register to read. The finaly result is
	// to be written to the writeTo buffer.
	ReadReg(wf interface{}, laneID int, reg *insts.Reg, writeTo []byte)

	// WriteReg writes the value in the writeFrom buffer to the register
	WriteReg(wf interface{}, laneID int, reg *insts.Reg, writeFrom []byte)
}

// RegInterfaceImpl is a RegInterface implementation for the emulator. It reads/
// write the memory directly to/ from the buffer. No request is generated in
// this process.
type RegInterfaceImpl struct {
}

// ReadReg directly put the register value to the writeTo buffer
func (i *RegInterfaceImpl) ReadReg(
	wf interface{},
	laneID int,
	reg *insts.Reg,
	writeTo []byte,
) {
	emuWf := wf.(*Wavefront)

	var regFile []byte
	var offset int
	if reg.IsSReg() {
		regFile = emuWf.SRegFile
		offset = reg.RegIndex() * 4
	} else if reg.IsVReg() {
		regFile = emuWf.VRegFile
		offset = (laneID*256 + reg.RegIndex()) * 4
	} else if reg.RegType == insts.Scc {
		writeTo[0] = emuWf.SCC
	} else {
		log.Panic("Only SGPRs and VGPRs are supported ")
	}

	copy(writeTo, regFile[offset:])
}

// WriteReg directly fetch the value from the writeFrom buffer and put the value
// into the register.
func (i *RegInterfaceImpl) WriteReg(
	wf interface{},
	laneID int,
	reg *insts.Reg,
	writeFrom []byte,
) {
	emuWf := wf.(*Wavefront)

	var regFile []byte
	var offset int
	if reg.IsSReg() {
		regFile = emuWf.SRegFile
		offset = reg.RegIndex() * 4
	} else if reg.IsVReg() {
		regFile = emuWf.VRegFile
		offset = (laneID*256 + reg.RegIndex()) * 4
	} else if reg.RegType == insts.Scc {
		emuWf.SCC = writeFrom[0]
	} else {
		log.Panic("Only SGPRs and VGPRs are supported ")
	}

	copy(regFile[offset:], writeFrom)
}
