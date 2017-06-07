package kernels

import (
	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3/insts"
)

// A LaunchKernelReq is a request that asks a GPU to launch a kernel
type LaunchKernelReq struct {
	*core.ReqBase

	Packet *HsaKernelDispatchPacket
	HsaCo  *insts.HsaCo

	OK bool
}

// NewLaunchKernelReq returns a new LaunchKernelReq
func NewLaunchKernelReq() *LaunchKernelReq {
	r := new(LaunchKernelReq)
	r.ReqBase = core.NewReqBase()
	return r
}
