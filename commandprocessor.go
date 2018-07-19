package gcn3

import (
	"log"
	"reflect"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/kernels"
	"gitlab.com/yaotsu/mem"
	"gitlab.com/yaotsu/mem/cache"
)

type Resettable interface {
	Reset()
}

// CommandProcessor is a Yaotsu component that is responsible for receiving
// requests from the driver and dispatch the requests to other parts of the
// GPU.
//
//     ToDriver <=> Receive request and send feedback to the driver
//     ToDispatcher <=> Dispatcher of compute kernels
type CommandProcessor struct {
	*core.ComponentBase

	engine core.Engine
	Freq   core.Freq

	Dispatcher *core.Port
	DMAEngine  *core.Port
	Driver     *core.Port

	ToDriver     *core.Port
	ToDispatcher *core.Port

	ToResetAfterKernel          []Resettable
	L2Caches                    []*cache.WriteBackCache
	GPUStorage                  *mem.Storage
	kernelFixedOverheadInCycles int
}

func (p *CommandProcessor) NotifyRecv(now core.VTimeInSec, port *core.Port) {
	req := port.Retrieve(now)
	core.ProcessReqAsEvent(req, p.engine, p.Freq)
}

func (p *CommandProcessor) NotifyPortFree(now core.VTimeInSec, port *core.Port) {
	//panic("implement me")
}

// Handle processes the events that is scheduled for the CommandProcessor
func (p *CommandProcessor) Handle(e core.Event) error {
	switch req := e.(type) {
	case *kernels.LaunchKernelReq:
		return p.processLaunchKernelReq(req)
	case *ReplyKernelCompletionEvent:
		return p.handleReplyKernelCompletionEvent(req)
	case *MemCopyD2HReq:
		return p.processMemCopyReq(req)
	case *MemCopyH2DReq:
		return p.processMemCopyReq(req)
	default:
		log.Panicf("cannot process request %s", reflect.TypeOf(req))
	}
	return nil
}

func (p *CommandProcessor) processLaunchKernelReq(
	req *kernels.LaunchKernelReq,
) error {
	now := req.Time()
	if req.Src() == p.Driver {
		req.SetDst(p.Dispatcher)
		req.SetSrc(p.ToDispatcher)
		req.SetSendTime(now)
		p.ToDispatcher.Send(req)
	} else if req.Src() == p.Dispatcher {

		p.finalFlush()

		req.SetDst(p.Driver)
		req.SetSrc(p.ToDriver)
		evt := NewReplyKernelCompletionEvent(
			p.Freq.NCyclesLater(p.kernelFixedOverheadInCycles, now),
			p, req,
		)
		p.engine.Schedule(evt)
	} else {
		log.Panic("The request sent to the command processor has unknown src")
	}
	return nil
}

func (p *CommandProcessor) finalFlush() {
	// FIXME: This is magic, remove
	for _, r := range p.ToResetAfterKernel {
		r.Reset()
	}

	for _, l2Cache := range p.L2Caches {
		p.flushL2(l2Cache)
	}
}

func (p *CommandProcessor) flushL2(l2 *cache.WriteBackCache) {
	dir := l2.Directory.(*cache.DirectoryImpl)
	for _, set := range dir.Sets {
		for _, block := range set.Blocks {
			if block.IsLocked {
				log.Panic("block locked.")
			}

			if block.IsDirty && block.IsValid {
				cacheData, _ := l2.Storage.Read(block.CacheAddress, uint64(dir.BlockSize))
				p.GPUStorage.Write(block.Tag, cacheData)
			}
		}

	}
}

func (p *CommandProcessor) handleReplyKernelCompletionEvent(evt *ReplyKernelCompletionEvent) error {
	now := evt.Time()
	evt.Req.SetSendTime(now)
	p.ToDriver.Send(evt.Req)
	return nil
}

func (p *CommandProcessor) processMemCopyReq(req core.Req) error {
	now := req.Time()
	if req.Src() == p.Driver {
		req.SetDst(p.DMAEngine)
		req.SetSrc(p.ToDispatcher)
		req.SetSendTime(now)
		p.ToDispatcher.Send(req)
	} else if req.Src() == p.DMAEngine {
		req.SetDst(p.Driver)
		req.SetSrc(p.ToDriver)
		req.SetSendTime(now)
		p.ToDriver.Send(req)
	} else {
		log.Panic("The request sent to the command processor has unknown src")
	}
	return nil
}

// NewCommandProcessor creates a new CommandProcessor
func NewCommandProcessor(name string, engine core.Engine) *CommandProcessor {
	c := new(CommandProcessor)
	c.ComponentBase = core.NewComponentBase(name)

	c.engine = engine
	c.Freq = 1 * core.GHz

	c.kernelFixedOverheadInCycles = 1600
	c.L2Caches = make([]*cache.WriteBackCache, 0)

	c.ToDriver = core.NewPort(c)
	c.ToDispatcher = core.NewPort(c)

	return c
}

type ReplyKernelCompletionEvent struct {
	*core.EventBase
	Req *kernels.LaunchKernelReq
}

func NewReplyKernelCompletionEvent(
	time core.VTimeInSec,
	handler core.Handler,
	req *kernels.LaunchKernelReq,
) *ReplyKernelCompletionEvent {
	evt := new(ReplyKernelCompletionEvent)
	evt.EventBase = core.NewEventBase(time, handler)
	evt.Req = req
	return evt
}
