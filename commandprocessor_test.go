package gcn3

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/akita"
)

var _ = Describe("CommandProcessor", func() {

	var (
		engine           *akita.MockEngine
		driver           *akita.MockComponent
		dispatcher       *akita.MockComponent
		commandProcessor *CommandProcessor
		connection       *akita.MockConnection
	)

	BeforeEach(func() {
		engine = akita.NewMockEngine()
		connection = akita.NewMockConnection()

		driver = akita.NewMockComponent("dispatcher")
		dispatcher = akita.NewMockComponent("dispatcher")
		commandProcessor = NewCommandProcessor("commandProcessor", engine)

		commandProcessor.Dispatcher = dispatcher.ToOutside
		commandProcessor.Driver = driver.ToOutside

		connection.PlugIn(commandProcessor.ToDispatcher)
		connection.PlugIn(commandProcessor.ToDriver)
	})

	It("should forward kernel launching request to Dispatcher", func() {
		req := NewLaunchKernelReq(10,
			driver.ToOutside, commandProcessor.ToDriver)
		req.SetEventTime(10)

		reqExpect := NewLaunchKernelReq(10,
			commandProcessor.ToDispatcher, dispatcher.ToOutside)
		connection.ExpectSend(reqExpect, nil)

		commandProcessor.Handle(req)

		Expect(connection.AllExpectedSent()).To(BeTrue())
	})

	It("should delay forward kernel launching request to the Driver", func() {
		req := NewLaunchKernelReq(10,
			dispatcher.ToOutside, commandProcessor.ToDispatcher)

		commandProcessor.Handle(req)

		Expect(engine.ScheduledEvent).To(HaveLen(1))
	})
})
