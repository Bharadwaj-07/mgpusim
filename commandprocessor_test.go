package gcn3

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo" // . "github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/akita"
	"gitlab.com/akita/mem/vm"
)

var _ = Describe("CommandProcessor", func() {

	var (
		mockCtrl         *gomock.Controller
		engine           *MockEngine
		driver           *MockPort
		dispatcher       *MockPort
		commandProcessor *CommandProcessor
		toDriver         *MockPort
		toDispatcher     *MockPort
		cus              []*MockPort
		toCU             *MockPort
		vmModules        []*MockPort
		toVMModules      *MockPort
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		engine = NewMockEngine(mockCtrl)

		driver = NewMockPort(mockCtrl)
		dispatcher = NewMockPort(mockCtrl)
		toDriver = NewMockPort(mockCtrl)
		toDispatcher = NewMockPort(mockCtrl)
		commandProcessor = NewCommandProcessor("commandProcessor", engine)
		commandProcessor.numCUs = 10
		commandProcessor.numVMUnits = 11
		commandProcessor.ToDispatcher = toDispatcher
		commandProcessor.ToDriver = toDriver

		commandProcessor.Dispatcher = dispatcher
		commandProcessor.Driver = driver

		toCU = NewMockPort(mockCtrl)
		toVMModules = NewMockPort(mockCtrl)

		commandProcessor.ToCUs = toCU
		commandProcessor.ToVMModules = toVMModules

		for i := 0; i < int(commandProcessor.numCUs); i++ {

			cus = append(cus, NewMockPort(mockCtrl))
			commandProcessor.CUs = append(commandProcessor.CUs, akita.NewLimitNumMsgPort(commandProcessor, 1))
			commandProcessor.CUs[i] = cus[i]
		}

		for i := 0; i < int(commandProcessor.numVMUnits); i++ {

			vmModules = append(vmModules, NewMockPort(mockCtrl))
			commandProcessor.VMModules = append(commandProcessor.VMModules, akita.NewLimitNumMsgPort(commandProcessor, 1))
			commandProcessor.VMModules[i] = vmModules[i]
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should forward kernel launching request to Dispatcher", func() {
		req := NewLaunchKernelReq(10,
			driver, commandProcessor.ToDriver)
		req.EventTime = 10

		toDispatcher.EXPECT().Send(gomock.AssignableToTypeOf(req))

		commandProcessor.Handle(req)
	})

	It("should delay forward kernel launching request to the Driver", func() {
		req := NewLaunchKernelReq(10,
			dispatcher, commandProcessor.ToDispatcher)

		engine.EXPECT().Schedule(
			gomock.AssignableToTypeOf(&ReplyKernelCompletionEvent{}))

		commandProcessor.Handle(req)
	})

	It("should handle a TLB shootdown request from the Driver and send a pipeline drain to CU", func() {
		vAddr := make([]uint64, 1)
		vAddr[0] = 0x1000

		req := NewShootdownCommand(10, nil, commandProcessor.ToDriver, vAddr, 1)
		req.EventTime = 10

		for i := 0; i < int(commandProcessor.numCUs); i++ {
			reqDrain := NewCUPipelineDrainReq(10, commandProcessor.ToCUs, commandProcessor.CUs[i])
			toCU.EXPECT().Send(gomock.AssignableToTypeOf(reqDrain))
		}

		commandProcessor.Handle(req)
	})

	It("should handle a CU drain ack from the CU and send a invalidation to all VM units", func() {

		vAddr := make([]uint64, 1)
		vAddr[0] = 0x1000
		shootDownreq := NewShootdownCommand(8, nil, commandProcessor.ToDriver, vAddr, 1)
		commandProcessor.curShootdownRequest = shootDownreq

		req := NewCUPipelineDrainRsp(10, nil, commandProcessor.ToCUs)
		req.drainPipelineComplete = true
		req.EventTime = 10

		commandProcessor.numCURecvdAck = commandProcessor.numCUs - 1

		for i := 0; i < int(commandProcessor.numVMUnits); i++ {

			reqShootdown := vm.NewPTEInvalidationReq(10, commandProcessor.ToVMModules, commandProcessor.VMModules[i], shootDownreq.PID, shootDownreq.VAddr)
			toVMModules.EXPECT().Send(gomock.AssignableToTypeOf(reqShootdown))
		}

		commandProcessor.Handle(req)

	})

	It("should handle a VM invalidation done from the VM units and send a ack to driver", func() {

		shootDownRsp := vm.NewInvalidationCompleteRsp(10, nil, commandProcessor.ToVMModules, "vm")
		shootDownRsp.InvalidationDone = true
		commandProcessor.numVMRecvdAck = commandProcessor.numVMUnits - 1

		shootDownComplete := NewShootdownCompleteRsp(10, commandProcessor.ToDriver, commandProcessor.Driver)
		toDriver.EXPECT().Send(gomock.AssignableToTypeOf(shootDownComplete))

		commandProcessor.Handle(shootDownRsp)

		Expect(commandProcessor.shootDownInProcess).To(BeFalse())
		Expect(commandProcessor.numVMRecvdAck).To(Equal(uint64(0)))
		Expect(commandProcessor.numCURecvdAck).To(Equal(uint64(0)))

	})

})
