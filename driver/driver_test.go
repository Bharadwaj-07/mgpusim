package driver

import (
	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/mem/vm"
	"gitlab.com/akita/util/ca"
)

var _ = ginkgo.Describe("Driver", func() {

	var (
		mockCtrl *gomock.Controller
		//gpu      *gcn3.GPU
		mmu *MockMMU

		driver         *Driver
		engine         *MockEngine
		toGPUs         *MockPort
		toMMU          *MockPort
		remotePMCPorts []*MockPort
		context        *Context
		cmdQueue       *CommandQueue
	)

	ginkgo.BeforeEach(func() {
		mockCtrl = gomock.NewController(ginkgo.GinkgoT())
		engine = NewMockEngine(mockCtrl)
		toGPUs = NewMockPort(mockCtrl)
		mmu = NewMockMMU(mockCtrl)
		toMMU = NewMockPort(mockCtrl)

		driver = NewDriver(engine, mmu)
		driver.ToGPUs = toGPUs
		driver.ToMMU = toMMU

		for i := 0; i < 2; i++ {
			gpu := gcn3.NewGPU("GPU", engine)
			remotePMCPorts = append(remotePMCPorts, NewMockPort(mockCtrl))
			driver.remotePMCPorts = append(driver.remotePMCPorts, akita.NewLimitNumMsgPort(driver, 1))
			driver.remotePMCPorts[i] = remotePMCPorts[i]
			driver.RegisterGPU(gpu, 4*mem.GB)

		}

		context = driver.Init()
		context.pid = 1
		cmdQueue = driver.CreateCommandQueue(context)
	})

	ginkgo.AfterEach(func() {
		mockCtrl.Finish()
	})

	ginkgo.Context("process MemCopyH2D command", func() {
		ginkgo.It("should send request", func() {

			srcData := make([]byte, 0x2200)
			cmd := &MemCopyH2DCommand{
				Dst: GPUPtr(0x200000100),
				Src: srcData,
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = false

			mmu.EXPECT().
				Translate(ca.PID(1), uint64(0x200000100)).
				Return(uint64(0x100000100), &vm.Page{
					PID:      1,
					VAddr:    0x200000000,
					PAddr:    0x100000000,
					PageSize: 0x800,
					Valid:    true,
				})
			mmu.EXPECT().
				Translate(ca.PID(1), uint64(0x200000800)).
				Return(uint64(0x100000800), &vm.Page{
					PID:      1,
					VAddr:    0x200000800,
					PAddr:    0x100000800,
					PageSize: 0x800,
					Valid:    true,
				})
			mmu.EXPECT().
				Translate(ca.PID(1), uint64(0x200001000)).
				Return(uint64(0x100001000), &vm.Page{
					PID:      1,
					VAddr:    0x200001000,
					PAddr:    0x100001000,
					PageSize: 0x1000,
					Valid:    true,
				})
			mmu.EXPECT().
				Translate(ca.PID(1), uint64(0x200002000)).
				Return(uint64(0x100002000), &vm.Page{
					PID:      1,
					VAddr:    0x200002000,
					PAddr:    0x100002000,
					PageSize: 0x1000,
					Valid:    true,
				})

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(nil)
			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(driver.requestsToSend).To(HaveLen(4))
			Expect(cmdQueue.IsRunning).To(BeTrue())
			Expect(cmd.Reqs).To(HaveLen(4))
		})
	})

	ginkgo.Context("process MemCopyH2D return", func() {
		ginkgo.It("should remove one request", func() {
			req := gcn3.NewMemCopyH2DReq(9, toGPUs, nil,
				make([]byte, 4), 0x104)
			req2 := gcn3.NewMemCopyH2DReq(9, toGPUs, nil,
				make([]byte, 4), 0x100)
			cmd := &MemCopyH2DCommand{
				Dst:  GPUPtr(0x100),
				Src:  uint32(1),
				Reqs: []akita.Msg{req, req2},
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = true

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(req)
			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().
				Schedule(gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeTrue())
			Expect(cmdQueue.commands).To(HaveLen(1))
			Expect(cmd.Reqs).NotTo(ContainElement(req))
			Expect(cmd.Reqs).To(ContainElement(req2))
		})

		ginkgo.It("should remove command from queue if no more pending request", func() {
			req := gcn3.NewMemCopyH2DReq(9,
				toGPUs, nil,
				make([]byte, 4), 0x100)
			cmd := &MemCopyH2DCommand{
				Dst:  GPUPtr(0x100),
				Src:  uint32(1),
				Reqs: []akita.Msg{req},
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = true

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(req)

			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(
				gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeFalse())
			Expect(cmdQueue.NumCommand()).To(Equal(0))
		})

	})

	ginkgo.Context("process MemCopyD2HCommand", func() {
		ginkgo.It("should send request", func() {
			data := uint32(1)
			cmd := &MemCopyD2HCommand{
				Dst: &data,
				Src: GPUPtr(0x200000100),
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = false

			mmu.EXPECT().Translate(ca.PID(1), uint64(0x200000100)).
				Return(uint64(0x100000100), &vm.Page{
					PID:      1,
					VAddr:    0x200000000,
					PAddr:    0x100000000,
					PageSize: 0x1000,
					Valid:    true,
				})

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(nil)
			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(
				gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeTrue())
			Expect(cmd.Reqs).To(HaveLen(1))
			Expect(driver.requestsToSend).To(HaveLen(1))
		})
	})

	ginkgo.Context("process MemCopyD2H return", func() {
		ginkgo.It("should remove request", func() {
			data := uint64(0)
			req := gcn3.NewMemCopyD2HReq(
				9, nil, toGPUs, 0x100, []byte{1, 0, 0, 0})
			req2 := gcn3.NewMemCopyD2HReq(
				9, nil, toGPUs, 0x104, []byte{1, 0, 0, 0})
			cmd := &MemCopyD2HCommand{
				Dst:  &data,
				Src:  GPUPtr(0x100),
				Reqs: []akita.Msg{req, req2},
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = true

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(req)
			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(
				gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeTrue())
			Expect(cmdQueue.commands).To(HaveLen(1))
			Expect(cmd.Reqs).To(ContainElement(req2))
			Expect(cmd.Reqs).NotTo(ContainElement(req))
		})

		ginkgo.It("should continue queue", func() {
			data := uint32(0)
			req := gcn3.NewMemCopyD2HReq(9, nil, toGPUs,
				0x100,
				[]byte{1, 0, 0, 0})
			cmd := &MemCopyD2HCommand{
				Dst:     &data,
				RawData: []byte{1, 0, 0, 0},
				Src:     GPUPtr(0x100),
				Reqs:    []akita.Msg{req},
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = true

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(req)
			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeFalse())
			Expect(cmdQueue.commands).To(HaveLen(0))
			Expect(data).To(Equal(uint32(1)))
		})

	})

	ginkgo.Context("process LaunchKernelCommand", func() {
		ginkgo.It("should send request to GPU", func() {
			cmd := &LaunchKernelCommand{
				CodeObject: nil,
				GridSize:   [3]uint32{256, 1, 1},
				WGSize:     [3]uint16{64, 1, 1},
				KernelArgs: nil,
			}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = false

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(nil)

			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(
				gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeTrue())
			Expect(cmd.Reqs).To(HaveLen(1))
			req := cmd.Reqs[0].(*gcn3.LaunchKernelReq)
			Expect(req.PID).To(Equal(ca.PID(1)))
			Expect(driver.requestsToSend).To(HaveLen(1))
		})
	})

	ginkgo.It("should process LaunchKernel return", func() {
		req := gcn3.NewLaunchKernelReq(9, toGPUs, nil)
		cmd := &LaunchKernelCommand{
			Reqs: []akita.Msg{req},
		}
		cmdQueue.Enqueue(cmd)
		cmdQueue.IsRunning = true

		toGPUs.EXPECT().
			Retrieve(akita.VTimeInSec(11)).
			Return(req)

		toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

		engine.EXPECT().Schedule(gomock.AssignableToTypeOf(akita.TickEvent{}))

		driver.Handle(*akita.NewTickEvent(11, nil))

		Expect(cmdQueue.IsRunning).To(BeFalse())
		Expect(cmdQueue.commands).To(HaveLen(0))
	})

	ginkgo.Context("process FlushCommand", func() {
		ginkgo.It("should send request to GPU", func() {
			cmd := &FlushCommand{}
			cmdQueue.Enqueue(cmd)
			cmdQueue.IsRunning = false

			toGPUs.EXPECT().
				Retrieve(akita.VTimeInSec(11)).
				Return(nil)
			toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

			engine.EXPECT().Schedule(
				gomock.AssignableToTypeOf(akita.TickEvent{}))

			driver.Handle(*akita.NewTickEvent(11, nil))

			Expect(cmdQueue.IsRunning).To(BeTrue())
			Expect(cmd.Reqs).To(HaveLen(1))
			Expect(driver.requestsToSend).To(HaveLen(1))
		})
	})

	ginkgo.It("should process Flush return", func() {
		req := gcn3.NewFlushCommand(9, toGPUs, nil)
		cmd := &FlushCommand{
			Reqs: []akita.Msg{req},
		}
		cmdQueue.Enqueue(cmd)

		cmdQueue.IsRunning = true

		toGPUs.EXPECT().
			Retrieve(akita.VTimeInSec(11)).
			Return(req)

		toMMU.EXPECT().Retrieve(akita.VTimeInSec(11)).Return(nil)

		engine.EXPECT().Schedule(gomock.AssignableToTypeOf(akita.TickEvent{}))

		driver.Handle(*akita.NewTickEvent(11, nil))

		Expect(cmdQueue.IsRunning).To(BeFalse())
		Expect(cmdQueue.commands).To(HaveLen(0))
	})

	ginkgo.It("should handle page migration req from MMU ", func() {
		req := vm.NewPageMigrationReqToDriver(10, nil, driver.ToMMU)
		toMMU.EXPECT().Retrieve(akita.VTimeInSec(10)).Return(req)
		driver.isCurrentlyHandlingMigrationReq = false

		for i := 0; i < 2; i++ {
			rdmaDrainReq := gcn3.NewRDMADrainCmdFromDriver(10, driver.ToGPUs, driver.GPUs[i].ToDriver)
			toGPUs.EXPECT().Send(gomock.AssignableToTypeOf(rdmaDrainReq))

		}

		driver.parseFromMMU(10)

		Expect(driver.currentPageMigrationReq).To(Equal(req))
		Expect(driver.isCurrentlyHandlingMigrationReq).To(BeTrue())
		Expect(driver.numRDMADrainACK).To(Equal(uint64(2)))
	})
	ginkgo.It("should handle RDMA Drain RSP ", func() {
		req := gcn3.NewRDMADrainRspToDriver(10, nil, driver.ToGPUs)
		driver.numRDMADrainACK = 1

		pageMigrationReq := vm.NewPageMigrationReqToDriver(10, nil, driver.ToMMU)
		pageMigrationReq.PageSize = 4 * mem.KB
		pageMigrationReq.CurPageHostGPU = 1
		pageMigrationReq.CurAccessingGPUs = append(pageMigrationReq.CurAccessingGPUs, 1)
		GpuReqToVaddrMap := make(map[uint64][]uint64)
		GpuReqToVaddrMap[2] = append(GpuReqToVaddrMap[2], 0x100)
		migrationInfo := new(vm.PageMigrationInfo)
		migrationInfo.GpuReqToVaddrMap = GpuReqToVaddrMap
		pageMigrationReq.MigrationInfo = migrationInfo

		driver.currentPageMigrationReq = pageMigrationReq

		toGPUs.EXPECT().Retrieve(akita.VTimeInSec(10)).Return(req)

		driver.processReturnReq(10)

		Expect(driver.numShootDownACK).To(Equal(uint64(1)))
		Expect(driver.NeedTick).To(BeTrue())
		Expect(len(driver.requestsToSend)).To(Equal(1))

	})

	ginkgo.It("should handle shootdown complete rsp", func() {
		req := gcn3.NewShootdownCompleteRsp(10, nil, driver.ToGPUs)

		pageMigrationReq := vm.NewPageMigrationReqToDriver(10, nil, driver.ToMMU)
		pageMigrationReq.PageSize = 4 * mem.KB
		pageMigrationReq.CurPageHostGPU = 1
		pageMigrationReq.CurAccessingGPUs = append(pageMigrationReq.CurAccessingGPUs, 1)
		GpuReqToVaddrMap := make(map[uint64][]uint64)
		GpuReqToVaddrMap[2] = append(GpuReqToVaddrMap[2], 0x100)
		migrationInfo := new(vm.PageMigrationInfo)
		migrationInfo.GpuReqToVaddrMap = GpuReqToVaddrMap
		pageMigrationReq.MigrationInfo = migrationInfo

		driver.currentPageMigrationReq = pageMigrationReq

		driver.numShootDownACK = 1

		page := &vm.Page{
			PID:      0,
			VAddr:    0x100,
			PAddr:    4294967296,
			PageSize: 4096,
			Valid:    true,
			GPUID:    1,
			Unified:  true,
		}

		mmu.EXPECT().CreatePage(page)
		mmu.CreatePage(page)

		mmu.EXPECT().CreatePage(page)
		driver.memAllocator.AllocatePageWithGivenVAddr(0, 1, 0x100, true)

		mmu.EXPECT().
			GetPageWithGivenVaddr(uint64(0x100), ca.PID(0)).
			Return(&vm.Page{
				PID:      0,
				VAddr:    0x100,
				PAddr:    4294967296,
				PageSize: 0x1000,
				Valid:    true,
				GPUID:    1,
				Unified:  true,
			})

		page2 := &vm.Page{
			PID:      0,
			VAddr:    0x100,
			PAddr:    8589934592,
			PageSize: 4096,
			Valid:    true,
			GPUID:    2,
			Unified:  true,
		}

		mmu.EXPECT().CreatePage(page2)

		mmu.EXPECT().RemovePage(ca.PID(0), uint64(0x100))
		mmu.EXPECT().MarkPageAsMigrating(uint64(0x100), ca.PID(0))

		toGPUs.EXPECT().Retrieve(akita.VTimeInSec(10)).Return(req)

		//migrationReqToCP := gcn3.NewPageMigrationReqToCP(10, driver.ToGPUs, driver.GPUs[1].ToDriver)
		//migrationReqToCP.DestinationPMCPort = driver.remotePMCPorts[0]

		driver.processReturnReq(10)
		Expect(driver.numPagesMigratingACK).To(Equal(uint64(1)))
		Expect(driver.migrationReqToSendToCP[0].Dst).To(Equal(driver.GPUs[1].ToDriver))
		Expect(driver.migrationReqToSendToCP[0].DestinationPMCPort).To(Equal(driver.remotePMCPorts[0]))
		Expect(driver.migrationReqToSendToCP[0].ToReadFromPhysicalAddress).To(Equal(uint64(4294967296)))
		Expect(driver.migrationReqToSendToCP[0].ToWriteToPhysicalAddress).To(Equal(uint64(8589934592)))
		Expect(driver.migrationReqToSendToCP[0].PageSize).To(Equal(uint64(4 * mem.KB)))

	})

	ginkgo.It("should send migration req to CP", func() {
		migrationReqToCP := gcn3.NewPageMigrationReqToCP(10, driver.ToGPUs, driver.GPUs[1].ToDriver)
		driver.migrationReqToSendToCP = append(driver.migrationReqToSendToCP, migrationReqToCP)

		toGPUs.EXPECT().Send(migrationReqToCP)

		driver.sendMigrationReqToCP(10)

		Expect(driver.isCurrentlyMigratingOnePage).To(BeTrue())
		Expect(driver.NeedTick).To(BeTrue())

	})

	ginkgo.It("should process page migration rsp from CP", func() {
		req := gcn3.NewPageMigrationRspToDriver(10, nil, driver.ToGPUs)

		toGPUs.EXPECT().Retrieve(akita.VTimeInSec(10)).Return(req)
		driver.numPagesMigratingACK = 2
		driver.processReturnReq(10)

		Expect(driver.numPagesMigratingACK).To(Equal(uint64(1)))
		Expect(driver.isCurrentlyMigratingOnePage).To(BeFalse())

	})

	ginkgo.It("should process page migration rsp from CP and send restart reqs to GPU, RDMA and reply to MMU", func() {
		req := gcn3.NewPageMigrationRspToDriver(10, nil, driver.ToGPUs)
		toGPUs.EXPECT().Retrieve(akita.VTimeInSec(10)).Return(req)

		driver.numPagesMigratingACK = 1

		pageMigrationReq := vm.NewPageMigrationReqToDriver(10, nil, driver.ToMMU)
		pageMigrationReq.PageSize = 4 * mem.KB
		pageMigrationReq.CurPageHostGPU = 1
		pageMigrationReq.CurAccessingGPUs = append(pageMigrationReq.CurAccessingGPUs, 1)
		pageMigrationReq.RespondToTop = true
		GpuReqToVaddrMap := make(map[uint64][]uint64)
		GpuReqToVaddrMap[2] = append(GpuReqToVaddrMap[2], 0x100)
		migrationInfo := new(vm.PageMigrationInfo)
		migrationInfo.GpuReqToVaddrMap = GpuReqToVaddrMap
		pageMigrationReq.MigrationInfo = migrationInfo

		driver.currentPageMigrationReq = pageMigrationReq

		requestsToSend := make([]akita.Msg, 0)

		for i := 0; i < 2; i++ {
			req := gcn3.NewRDMARestartCmdFromDriver(10, driver.ToGPUs, driver.GPUs[i].ToDriver)
			requestsToSend = append(requestsToSend, req)
		}

		for i := 0; i < len(pageMigrationReq.CurAccessingGPUs); i++ {
			restartGPUID := pageMigrationReq.CurAccessingGPUs[i] - 1
			restartReq := gcn3.NewGPURestartReq(10, driver.ToGPUs, driver.GPUs[restartGPUID].ToDriver)
			requestsToSend = append(requestsToSend, restartReq)
		}

		reqToMMU := vm.NewPageMigrationRspFromDriver(10, driver.ToMMU, pageMigrationReq.Src)
		reqToMMU.Vaddr = append(reqToMMU.Vaddr, 0x100)
		reqToMMU.RspToTop = true

		driver.processReturnReq(10)

		Expect(driver.toSendToMMU).To(Equal(reqToMMU))

		for i := 0; i < len(requestsToSend); i++ {
			Expect(driver.requestsToSend[i]).To(Equal(requestsToSend[i]))
		}

	})

	ginkgo.It("should send to MMU", func() {
		reqToMMU := vm.NewPageMigrationRspFromDriver(10, driver.ToMMU, nil)
		driver.toSendToMMU = reqToMMU

		toMMU.EXPECT().Send(reqToMMU)

		driver.sendToMMU(10)

		Expect(driver.NeedTick).To(BeTrue())
		Expect(driver.toSendToMMU).To(BeNil())

	})

})
