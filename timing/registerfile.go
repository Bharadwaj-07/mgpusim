package timing

import (
	"log"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/insts"
	"gitlab.com/yaotsu/mem"
)

// A RegisterAccess is an incidence of reading or writing the register
type RegisterAccess struct {
	Time       core.VTimeInSec
	Reg        *insts.Reg
	RegCount 	int
	LaneID     int
	WaveOffset int
	Data       []byte
	OK         bool
}

// A RegisterFile provides the communication interface for a set of registers.
type RegisterFile interface {
	Read(access *RegisterAccess)
	Write(access *RegisterAccess)
}

// A SimpleRegisterFile is a Register file that can always read and write
// registers immediately
type SimpleRegisterFile struct {
	storage *mem.Storage
}

// NewSimpleRegisterFile creates and returns a new SimpleRegisterFile
func NewSimpleRegisterFile(byteSize uint64) *SimpleRegisterFile {
	r := new(SimpleRegisterFile)
	r.storage = mem.NewStorage(byteSize)
	return r
}

func (r *SimpleRegisterFile) Write(access *RegisterAccess) {
	offset := r.getRegOffset(access)

	err := r.storage.Write(uint64(offset), access.Data)
	if err != nil {
		log.Panic(err)
	}

	access.OK = true
}

func (r *SimpleRegisterFile) Read(access *RegisterAccess) {
	offset := r.getRegOffset(access)

	data, err := r.storage.Read(uint64(offset), uint64(4*access.RegCount))
	if err != nil {
		log.Panic(err)
	}

	access.Data = data
	access.OK = true
}

func (r *SimpleRegisterFile) getRegOffset(access *RegisterAccess) int {

	reg := access.Reg
	offset := access.WaveOffset

	if reg.IsSReg() {
		return reg.RegIndex()*4 + offset
	}

	if reg.IsVReg() {
		return reg.RegIndex()*4 + offset
	}

	log.Panic("Register type not supported by register files")

	return 0
}
