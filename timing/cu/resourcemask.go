package cu

// AllocStatus represents the allocation status of SGPRs, VGPRs, or LDS units
type AllocStatus byte

// A list of possible status for CU binded storage allocation
const (
	AllocStatusFree      AllocStatus = iota
	AllocStatusToReserve             // A value that is used for reservation caculation
	AllocStatusReserved              // Work-Group mapped, but wavefront not dispatched
	AllocStatusUsed                  // Currently in use
)

// A ResourceMask is data structure to mask the status of some resources
type ResourceMask struct {
	mask []AllocStatus
}

// NewResourceMask returns a newly created ResourceMask with a given size.
func NewResourceMask(size int) *ResourceMask {
	m := new(ResourceMask)
	m.mask = make([]AllocStatus, size)
	return m
}

// NextRegion finds a region that is masked by the resourceMask in
// the state define by statusReq. This function returns the offset of the
// starting point of the region. It also returns a boolean value that
// tells if a region is found
func (m *ResourceMask) NextRegion(
	length int,
	statusReq AllocStatus,
) (int, bool) {
	offset := 0
	currLength := 0
	for offset < len(m.mask) {
		if m.mask[offset] == statusReq {
			currLength++
			if currLength == length {
				return offset - currLength + 1, true
			}
		} else {
			currLength = 0
		}
		offset++
	}
	return 0, false
}

// SetStatus alters the status from the position of offset to offset + length
func (m *ResourceMask) SetStatus(offset, length int, status AllocStatus) {
	for i := 0; i < length; i++ {
		m.mask[offset+i] = status
	}
}
