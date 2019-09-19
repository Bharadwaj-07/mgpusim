package l1v

import (
	"gitlab.com/akita/akita"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/mem/cache"
	"gitlab.com/akita/util"
	"gitlab.com/akita/util/akitaext"
)

// A Cache is a customized L1 cache the for R9nano GPUs.
type Cache struct {
	*akitaext.TickingComponent

	TopPort     akita.Port
	BottomPort  akita.Port
	ControlPort akita.Port

	numReqPerCycle   int
	log2BlockSize    uint64
	storage          *mem.Storage
	directory        cache.Directory
	mshr             cache.MSHR
	bankLatency      int
	wayAssociativity int
	lowModuleFinder  cache.LowModuleFinder

	dirBuf   util.Buffer
	bankBufs []util.Buffer

	coalesceStage    *coalescer
	directoryStage   *directory
	bankStages       []*bankStage
	parseBottomStage *bottomParser
	respondStage     *respondStage
	controlStage     *controlStage

	transactions             []*transaction
	postCoalesceTransactions []*transaction

	isPaused bool
}

// Tick update the state of the cache
func (c *Cache) Tick(now akita.VTimeInSec) bool {
	madeProgress := false

	for i := 0; i < c.numReqPerCycle; i++ {
		if !c.isPaused {

			madeProgress = c.respondStage.Tick(now) || madeProgress
			madeProgress = c.parseBottomStage.Tick(now) || madeProgress
			for _, bs := range c.bankStages {
				madeProgress = bs.Tick(now) || madeProgress
			}
			madeProgress = c.directoryStage.Tick(now) || madeProgress
			madeProgress = c.coalesceStage.Tick(now) || madeProgress
		}
		madeProgress = c.controlStage.Tick(now) || madeProgress

	}

	return madeProgress
}
