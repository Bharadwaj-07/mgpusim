package gcn3

import (
	"log"

	"gitlab.com/akita/mem"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/mem/cache"
)

// A GPU is the unit that one kernel can run on.
//
// A GPU is a Yaotsu component and it defines the port "ToDriver". Driver is
// a piece of software that conceptually runs in the Cpu. Therefore, all the
// CPU-GPU communication happens on the connection connecting the "ToDriver"
// port.
type GPU struct {
	*akita.ComponentBase

	engine akita.Engine
	Freq   akita.Freq

	Driver           *akita.Port // The DriverComponent
	CommandProcessor *akita.Port // The CommandProcessor
	Dispatchers      []akita.Component
	CUs              []akita.Component
	L1VCaches        []akita.Component
	L1ICaches        []akita.Component
	L1KCaches        []akita.Component
	L2Caches         []akita.Component
	L2CacheFinder    cache.LowModuleFinder
	DRAMStorage      *mem.Storage

	ToDriver           *akita.Port
	ToCommandProcessor *akita.Port
}

func (g *GPU) NotifyPortFree(now akita.VTimeInSec, port *akita.Port) {
}

func (g *GPU) NotifyRecv(now akita.VTimeInSec, port *akita.Port) {
	req := port.Retrieve(now)
	akita.ProcessReqAsEvent(req, g.engine, g.Freq)
}

// Handle defines how a GPU handles akita.
//
// A GPU should not handle any event by itself.
func (g *GPU) Handle(e akita.Event) error {
	now := e.Time()
	req := e.(akita.Req)

	if req.Src() == g.CommandProcessor { // From the CommandProcessor
		req.SetSrc(g.ToDriver)
		req.SetDst(g.Driver)
		req.SetSendTime(now)
		g.ToDriver.Send(req)
		return nil
	} else if req.Src() == g.Driver { // From the Driver
		req.SetSrc(g.ToCommandProcessor)
		req.SetDst(g.CommandProcessor)
		req.SetSendTime(now)
		g.ToCommandProcessor.Send(req)
		return nil
	}

	log.Panic("Unknown source")

	return nil
}

// NewGPU returns a newly created GPU
func NewGPU(name string, engine akita.Engine) *GPU {
	g := new(GPU)
	g.ComponentBase = akita.NewComponentBase(name)

	g.engine = engine
	g.Freq = 1 * akita.GHz

	g.ToDriver = akita.NewPort(g)
	g.ToCommandProcessor = akita.NewPort(g)

	return g
}
