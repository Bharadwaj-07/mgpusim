package gcn3

import (
	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3/insts"
	"gitlab.com/akita/gcn3/kernels"
	"gitlab.com/akita/mem/vm"
)

// FlushCommand requests the GPU to flush all the cache to the main memory
type FlushCommand struct {
	*akita.ReqBase
}

//Shootdown command requests the GPU to perform a TLB shootdown and invalidate the corresponding PTE's
type ShootDownCommand struct {
	*akita.ReqBase

	StartTime akita.VTimeInSec
	EndTime   akita.VTimeInSec

	vAddr []uint64
	pID   vm.PID
}

type ShootDownCompleteCommand struct {
	*akita.ReqBase

	StartTime akita.VTimeInSec
	EndTime   akita.VTimeInSec

	shootDownComplete bool
}

//NewShootdownCommand tells the CP to drain all CU and invalidate PTE's in TLB and Page Tables
func NewShootdownCommand(time akita.VTimeInSec, src, dst akita.Port, vAddr []uint64, pID vm.PID) *ShootDownCommand {
	cmd := new(ShootDownCommand)
	cmd.ReqBase = akita.NewReqBase()
	cmd.SetSendTime(time)
	cmd.SetSrc(src)
	cmd.SetDst(dst)
	cmd.vAddr = vAddr
	cmd.pID = pID
	return cmd
}

func NewShootdownCompleteCommand(time akita.VTimeInSec, src, dst akita.Port) *ShootDownCompleteCommand {
	cmd := new(ShootDownCompleteCommand)
	cmd.ReqBase = akita.NewReqBase()
	cmd.SetSendTime(time)
	cmd.SetSrc(src)
	cmd.SetDst(dst)
	return cmd
}

// NewFlushCommand Creates a new flush command, setting the request send time
// with time and the source and destination.
func NewFlushCommand(time akita.VTimeInSec, src, dst akita.Port) *FlushCommand {
	cmd := new(FlushCommand)
	cmd.ReqBase = akita.NewReqBase()
	cmd.SetSendTime(time)
	cmd.SetSrc(src)
	cmd.SetDst(dst)
	return cmd
}

// A LaunchKernelReq is a request that asks a GPU to launch a kernel
type LaunchKernelReq struct {
	*akita.ReqBase

	PID vm.PID

	Packet        *kernels.HsaKernelDispatchPacket
	PacketAddress uint64
	HsaCo         *insts.HsaCo

	OK bool
}

// NewLaunchKernelReq returns a new LaunchKernelReq
func NewLaunchKernelReq(
	time akita.VTimeInSec,
	src, dst akita.Port) *LaunchKernelReq {
	r := new(LaunchKernelReq)
	r.ReqBase = akita.NewReqBase()
	r.SetSrc(src)
	r.SetDst(dst)
	r.SetSendTime(time)
	return r
}

// A MemCopyH2DReq is a request that asks the DMAEngine to copy memory
// from the host to the device
type MemCopyH2DReq struct {
	*akita.ReqBase
	SrcBuffer  []byte
	DstAddress uint64
}

// NewMemCopyH2DReq created a new MemCopyH2DReq
func NewMemCopyH2DReq(
	time akita.VTimeInSec,
	src, dst akita.Port,
	srcBuffer []byte,
	dstAddress uint64,
) *MemCopyH2DReq {
	reqBase := akita.NewReqBase()
	req := new(MemCopyH2DReq)
	req.ReqBase = reqBase
	req.SetSendTime(time)
	req.SetSrc(src)
	req.SetDst(dst)
	req.SrcBuffer = srcBuffer
	req.DstAddress = dstAddress
	return req
}

// A MemCopyD2HReq is a request that asks the DMAEngine to copy memory
// from the host to the device
type MemCopyD2HReq struct {
	*akita.ReqBase
	SrcAddress uint64
	DstBuffer  []byte
}

// NewMemCopyD2HReq created a new MemCopyH2DReq
func NewMemCopyD2HReq(
	time akita.VTimeInSec,
	src, dst akita.Port,
	srcAddress uint64,
	dstBuffer []byte,
) *MemCopyD2HReq {
	reqBase := akita.NewReqBase()
	req := new(MemCopyD2HReq)
	req.ReqBase = reqBase
	req.SetSendTime(time)
	req.SetSrc(src)
	req.SetDst(dst)
	req.SrcAddress = srcAddress
	req.DstBuffer = dstBuffer
	return req
}
