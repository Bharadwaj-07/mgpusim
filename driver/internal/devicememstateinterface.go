package internal

type deviceMemoryState interface {
	setInitialAddress(addr uint64)
	getInitialAddress() uint64
	setStorageSize(size uint64)
	getStorageSize() uint64
	addSinglePAddr(addr uint64)
	popNextAvailablePAddrs() uint64
	noAvailablePAddrs() bool
	allocateMultiplePages(numPages int) []uint64
}

func newDeviceRegularMemoryState() deviceMemoryState {
	return &deviceMemoryStateImpl{}
}

//original implementation of deviceMemoryState holding free addresses in array
type deviceMemoryStateImpl struct {
	initialAddress  uint64
	storageSize     uint64
	availablePAddrs []uint64
}

func (dms *deviceMemoryStateImpl) setInitialAddress(addr uint64) {
	dms.initialAddress = addr
}

func (dms *deviceMemoryStateImpl) getInitialAddress() uint64 {
		return dms.initialAddress
}

func (dms *deviceMemoryStateImpl) setStorageSize(size uint64) {
	dms.storageSize = size
}

func (dms *deviceMemoryStateImpl) getStorageSize() uint64 {
	return dms.storageSize
}

func (dms *deviceMemoryStateImpl) addSinglePAddr(addr uint64) {
	dms.availablePAddrs = append(dms.availablePAddrs, addr)
}

func (dms *deviceMemoryStateImpl) popNextAvailablePAddrs() uint64  {
	nextPAddr := dms.availablePAddrs[0]
	dms.availablePAddrs = dms.availablePAddrs[1:]
	return  nextPAddr
}

func (dms *deviceMemoryStateImpl) noAvailablePAddrs() bool {
	return len(dms.availablePAddrs) == 0
}

func (dms *deviceMemoryStateImpl) allocateMultiplePages(
	numPages int,
) (pAddrs []uint64) {
	for i := 0; i < numPages; i++ {
		pAddr := dms.popNextAvailablePAddrs()
		pAddrs = append(pAddrs, pAddr)
	}
	return pAddrs
}