package internal

func newDeviceBuddyMemoryState() deviceMemoryState {
	return &deviceBuddyMemoryState{}
}

//buddy allocation implementation of deviceMemoryState
type deviceBuddyMemoryState struct {
	initialAddress  uint64
	storageSize     uint64
	freeList        []*freeListElement
}

func (bms *deviceBuddyMemoryState) setInitialAddress(addr uint64) {
	bms.initialAddress = addr
}

func (bms *deviceBuddyMemoryState) getInitialAddress() uint64 {
	return bms.initialAddress
}

func (bms *deviceBuddyMemoryState) setStorageSize(size uint64) {
	bms.storageSize = size
	var order int
	for order = 12; (1 << order) <= size; order++ {}
	bms.freeList = make([]*freeListElement, order)

}

func (bms *deviceBuddyMemoryState) getStorageSize() uint64 {
	return bms.storageSize
}

func (bms *deviceBuddyMemoryState) addSinglePAddr(addr uint64) {

}

func (bms *deviceBuddyMemoryState) popNextAvailablePAddrs() uint64  {

	return  0
}

func (bms *deviceBuddyMemoryState) noAvailablePAddrs() bool {
	return false
}