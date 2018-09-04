package caches

import (
	"log"
	"reflect"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/mem"
	"gitlab.com/yaotsu/mem/cache"
)

type L1VCache struct {
	*core.TickingComponent

	ToCU *core.Port
	ToL2 *core.Port

	L2 *core.Port

	Directory cache.Directory
	Storage   *mem.Storage

	BlockSizeAsPowerOf2 uint64
	Latency             int

	cycleLeft int
	isBusy    bool
	reading   *mem.ReadReq
	writing   *mem.WriteReq

	isStorageBusy bool
	busyBlock     *cache.Block

	toCUBuffer           []core.Req
	toL2Buffer           []core.Req
	pendingDownGoingRead []*mem.ReadReq
}

func (c *L1VCache) Handle(e core.Event) error {
	switch e := e.(type) {
	case *core.TickEvent:
		c.handleTickEvent(e)
	default:
		log.Panicf("cannot handle event of type %s", reflect.TypeOf(e))
	}
	return nil
}

func (c *L1VCache) handleTickEvent(e *core.TickEvent) {
	now := e.Time()
	c.NeedTick = false

	c.sendToCU(now)
	c.sendToL2(now)
	c.doReadWrite(now)
	c.parseFromL2(now)
	c.parseFromCU(now)

	if c.NeedTick {
		c.TickLater(now)
	}
}

func (c *L1VCache) parseFromCU(now core.VTimeInSec) {
	if c.isBusy {
		return
	}

	req := c.ToCU.Retrieve(now)
	if req == nil {
		return
	}

	c.NeedTick = true
	switch req := req.(type) {
	case *mem.ReadReq:
		c.handleReadReq(now, req)
	default:
		log.Panicf("cannot process request of type %s",
			reflect.TypeOf(req))
	}
}

func (c *L1VCache) parseFromL2(now core.VTimeInSec) {
	req := c.ToL2.Peek()

	if req == nil {
		return
	}

	switch req := req.(type) {
	case *mem.DataReadyRsp:
		c.handleDataReadyRsp(now, req)
	default:
		log.Panicf("cannot process request of type %s",
			reflect.TypeOf(req))
	}
}

func (c *L1VCache) handleDataReadyRsp(now core.VTimeInSec, dataReady *mem.DataReadyRsp) {
	readBottom := c.pendingDownGoingRead[0]
	readTop := c.reading
	address := readTop.Address
	_, offset := cache.GetCacheLineID(address, c.BlockSizeAsPowerOf2)

	block := c.Directory.Evict(readBottom.Address)
	block.IsValid = true
	block.IsDirty = false
	c.Storage.Write(block.CacheAddress, dataReady.Data)

	dataReadyToTop := mem.NewDataReadyRsp(
		now, c.ToCU, c.reading.Src(), c.reading.GetID())
	dataReadyToTop.Data = dataReady.Data[offset : offset+readTop.MemByteSize]
	c.toCUBuffer = append(c.toCUBuffer, dataReadyToTop)

	c.ToL2.Retrieve(now)
	c.pendingDownGoingRead = nil
	c.isBusy = false
	c.NeedTick = true
}

func (c *L1VCache) handleReadReq(now core.VTimeInSec, req *mem.ReadReq) {
	c.isBusy = true
	c.reading = req

	block := c.Directory.Lookup(req.Address)
	if block == nil {
		c.handleReadMiss(now, req)
	} else {
		c.handleReadHit(now, req, block)
	}
}

func (c *L1VCache) handleReadMiss(now core.VTimeInSec, req *mem.ReadReq) {
	address := req.Address
	cacheLineID, _ := cache.GetCacheLineID(address, c.BlockSizeAsPowerOf2)
	readBottom := mem.NewReadReq(now, c.ToL2, c.L2, cacheLineID, 1<<c.BlockSizeAsPowerOf2)
	c.pendingDownGoingRead = append(c.pendingDownGoingRead, readBottom)
	c.toL2Buffer = append(c.toL2Buffer, readBottom)
}

func (c *L1VCache) handleReadHit(now core.VTimeInSec, req *mem.ReadReq, block *cache.Block) {
	c.cycleLeft = c.Latency
	c.busyBlock = block
	c.isStorageBusy = true
}

func (c *L1VCache) doReadWrite(now core.VTimeInSec) {
	if !c.isStorageBusy {
		return
	}

	c.cycleLeft--
	c.NeedTick = true

	if c.cycleLeft <= 0 {
		if c.reading != nil {
			c.finishLocalRead(now)
		}
	}
}

func (c *L1VCache) finishLocalRead(now core.VTimeInSec) {
	c.isStorageBusy = false
	c.isBusy = false

	_, offset := cache.GetCacheLineID(c.reading.Address, c.BlockSizeAsPowerOf2)
	data, err := c.Storage.Read(
		c.busyBlock.CacheAddress+offset, c.reading.MemByteSize)
	if err != nil {
		log.Panic(err)
	}

	dataReady := mem.NewDataReadyRsp(now, c.ToCU, c.reading.Src(), c.reading.ID)
	dataReady.Data = data
	c.toCUBuffer = append(c.toCUBuffer, dataReady)
}

func (c *L1VCache) sendToCU(now core.VTimeInSec) {
	if len(c.toCUBuffer) > 0 {
		req := c.toCUBuffer[0]
		req.SetSendTime(now)
		err := c.ToCU.Send(req)
		if err == nil {
			c.toCUBuffer = c.toCUBuffer[1:]
			c.NeedTick = true
		}
	}
}

func (c *L1VCache) sendToL2(now core.VTimeInSec) {
	if len(c.toL2Buffer) > 0 {
		req := c.toL2Buffer[0]
		req.SetSendTime(now)
		err := c.ToL2.Send(req)
		if err == nil {
			c.toL2Buffer = c.toL2Buffer[1:]
			c.NeedTick = true
		}
	}
}

func NewL1VCache(name string, engine core.Engine, freq core.Freq) *L1VCache {
	c := new(L1VCache)
	c.TickingComponent = core.NewTickingComponent(name, engine, freq, c)

	c.ToCU = core.NewPort(c)
	c.ToL2 = core.NewPort(c)
	return c
}
