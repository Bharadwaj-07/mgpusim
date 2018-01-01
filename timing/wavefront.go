package timing

import (
	"sync"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/insts"
	"gitlab.com/yaotsu/gcn3/kernels"
)

// WfState marks what state that wavefront it in.
type WfState int

// A list of all possible WfState
const (
	WfDispatching WfState = iota // Dispatching in progress, not ready to run
	WfReady                      // Allow the scheduler to schedule instruction
	WfFetching                   // Fetch request sent, but not returned
	WfFetched                    // Instruction fetched, but not issued
	WfRunning                    // Instruction in fight
	WfCompleted                  // Wavefront completed
)

// A Wavefront in the timing package contains the information of the progress
// of a wavefront
type Wavefront struct {
	*kernels.Wavefront
	sync.RWMutex

	CodeObject *insts.HsaCo
	Packet     *kernels.HsaKernelDispatchPacket

	State          WfState
	Inst           *Inst           // The instruction that is being executed
	ScratchPad     []byte          // A temp data buf that is shared by different stages
	LastFetchTime  core.VTimeInSec // The time that the last instruction was fetched
	CompletedLanes int             // The number of lanes that is completed in the SIMD unit

	PC          uint64
	FetchBuffer []byte
	SIMDID      int
	SRegOffset  int
	VRegOffset  int
	LDSOffset   int
}
