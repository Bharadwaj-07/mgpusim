package emu

// SOP2Layout represents the scratchpad layout for SOP2 instructions
type SOP2Layout struct {
	SRC0 uint64
	SRC1 uint64
	DST  uint64
	SCC  byte
}

// VOP1Layout represents the scratchpad layout for VOP1 instructions
type VOP1Layout struct {
	SRC0 [64]uint64
	DST  [64]uint64
}
