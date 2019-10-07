package gcn3

import (
	"log"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3/pagemigrationcontroller"
	"gitlab.com/akita/gcn3/rdma"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/mem/cache"
	"gitlab.com/akita/mem/cache/writeback"
	"gitlab.com/akita/mem/vm/addresstranslator"
	"gitlab.com/akita/mem/vm/tlb"
)

// A GPU is the unit that one kernel can run on.
//
// A GPU is a Akita component and it defines the port "ToDriver". Driver is
// a piece of software that conceptually runs in the CPU. Therefore, all the
// CPU-GPU communication happens on the connection connecting the "ToDriver"
// port.
type GPU struct {
	*akita.ComponentBase

	engine akita.Engine
	Freq   akita.Freq

	Driver             akita.Port
	CommandProcessor   akita.Port
	RDMAEngine         *rdma.Engine
	PMC                *pagemigrationcontroller.PageMigrationController
	Dispatchers        []akita.Component
	CUs                []akita.Component
	L1VCaches          []akita.Component
	L1ICaches          []akita.Component
	L1SCaches          []akita.Component
	L2Caches           []*writeback.Cache
	L2CacheFinder      cache.LowModuleFinder
	L2TLBs             []*tlb.TLB
	L1VTLBs            []*tlb.TLB
	L1STLBs            []*tlb.TLB
	L1ITLBs            []*tlb.TLB
	L1VAddrTranslator  []*addresstranslator.AddressTranslator
	L1IAddrTranslator  []*addresstranslator.AddressTranslator
	L1SAddrTranslator  []*addresstranslator.AddressTranslator
	MemoryControllers  []akita.Component
	Storage            *mem.Storage
	InternalConnection akita.Connection

	ToDriver           akita.Port
	ToCommandProcessor akita.Port

	GPUID uint64
}

// NotifyPortFree of a GPU does not do anything.
func (g *GPU) NotifyPortFree(now akita.VTimeInSec, port akita.Port) {
}

// NotifyRecv of a GPU retrieves the request from the port and process requests
// as Events.
func (g *GPU) NotifyRecv(now akita.VTimeInSec, port akita.Port) {
	req := port.Retrieve(now)
	akita.ProcessMsgAsEvent(req, g.engine, g.Freq)
}

// Handle defines how a GPU handles akita.
//
// A GPU should not handle any event by itself.
func (g *GPU) Handle(e akita.Event) error {
	now := e.Time()

	ctx := akita.HookCtx{
		Domain: g,
		Now:    now,
		Pos:    akita.HookPosBeforeEvent,
		Item:   e,
	}
	g.InvokeHook(&ctx)

	g.Lock()

	req := e.(akita.Msg)

	if req.Meta().Src == g.CommandProcessor { // From the CommandProcessor
		req.Meta().Src = g.ToDriver
		req.Meta().Dst = g.Driver
		req.Meta().SendTime = now
		err := g.ToDriver.Send(req)
		if err != nil {
			panic(err)
		}
	} else if req.Meta().Src == g.Driver { // From the Driver
		req.Meta().Src = g.ToCommandProcessor
		req.Meta().Dst = g.CommandProcessor
		req.Meta().SendTime = now
		err := g.ToCommandProcessor.Send(req)
		if err != nil {
			panic(err)
		}
	} else {
		log.Panic("Unknown source")
	}

	g.Unlock()

	ctx.Pos = akita.HookPosAfterEvent
	g.InvokeHook(&ctx)

	return nil
}

// NewGPU returns a newly created GPU
func NewGPU(name string, engine akita.Engine) *GPU {
	g := new(GPU)
	g.ComponentBase = akita.NewComponentBase(name)

	g.engine = engine
	g.Freq = 1 * akita.GHz

	g.ToDriver = akita.NewLimitNumMsgPort(g, 1)
	g.ToCommandProcessor = akita.NewLimitNumMsgPort(g, 1)

	return g
}
