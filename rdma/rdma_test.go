package rdma

import (
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/mem"
	"gitlab.com/yaotsu/mem/cache"
)

func TestRDMA(t *testing.T) {
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "RDMA")
}

var _ = Describe("Engine", func() {
	var (
		engine        core.Engine
		rdmaEngine    *Engine
		outsideConn   *core.MockConnection
		insideConn    *core.MockConnection
		localModules  *cache.SingleLowModuleFinder
		remoteModules *cache.SingleLowModuleFinder
		localCache    *core.MockComponent
		remoteGPU     *core.MockComponent
	)

	BeforeEach(func() {
		engine = core.NewMockEngine()
		localCache = core.NewMockComponent("LocalCache")
		remoteGPU = core.NewMockComponent("RemoveGPU")
		localModules = new(cache.SingleLowModuleFinder)
		localModules.LowModule = localCache.ToOutside
		remoteModules = new(cache.SingleLowModuleFinder)
		remoteModules.LowModule = remoteGPU.ToOutside
		rdmaEngine = NewEngine("RDMAEngine", engine, localModules, remoteModules)

		outsideConn = core.NewMockConnection()
		outsideConn.PlugIn(rdmaEngine.ToOutside)

		insideConn = core.NewMockConnection()
		insideConn.PlugIn(rdmaEngine.ToInside)
	})

	Context("Read from inside", func() {
		var read *mem.ReadReq

		BeforeEach(func() {
			read = mem.NewReadReq(6,
				localCache.ToOutside, rdmaEngine.ToInside, 0x100, 64)
			rdmaEngine.ToInside.Recv(read)
		})

		It("should send read to outside", func() {
			expectRead := mem.NewReadReq(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, 0x100, 64)
			outsideConn.ExpectSend(expectRead, nil)

			rdmaEngine.processReqFromInside(10)

			Expect(outsideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc[read.ID]).To(
				BeIdenticalTo(localCache.ToOutside))
		})

		It("should wait if outside connection is busy", func() {
			expectRead := mem.NewReadReq(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, 0x100, 64)
			outsideConn.ExpectSend(expectRead, core.NewSendError())

			rdmaEngine.processReqFromInside(10)

			Expect(outsideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(1))
		})
	})

	Context("Write from inside", func() {
		var write *mem.WriteReq

		BeforeEach(func() {
			write = mem.NewWriteReq(6, localCache.ToOutside, rdmaEngine.ToInside, 0x100)
			rdmaEngine.ToInside.Recv(write)
		})

		It("should send write to outside", func() {
			expectRead := mem.NewWriteReq(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, 0x100)
			outsideConn.ExpectSend(expectRead, nil)

			rdmaEngine.processReqFromInside(10)

			Expect(outsideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc[write.ID]).To(
				BeIdenticalTo(localCache.ToOutside))
		})

		It("should wait if outside connection is busy", func() {
			expectRead := mem.NewWriteReq(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, 0x100)
			outsideConn.ExpectSend(expectRead, core.NewSendError())

			rdmaEngine.processReqFromInside(10)

			Expect(outsideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(1))
		})
	})

	Context("DataReady from inside", func() {
		var (
			read      *mem.ReadReq
			dataReady *mem.DataReadyRsp
		)

		BeforeEach(func() {
			// Send a read from outside
			read = mem.NewReadReq(6,
				remoteGPU.ToOutside, rdmaEngine.ToOutside, 0x100, 64)
			rdmaEngine.ToOutside.Recv(read)
			expectRead := mem.NewReadReq(7,
				rdmaEngine.ToInside, localCache.ToOutside, 0x100, 64)
			insideConn.ExpectSend(expectRead, nil)
			rdmaEngine.processReqFromOutside(7)

			dataReady = mem.NewDataReadyRsp(12,
				localCache.ToOutside, rdmaEngine.ToInside, read.ID)
			rdmaEngine.ToInside.Recv(dataReady)
		})

		It("should send data ready to outside", func() {
			expectDataReady := mem.NewDataReadyRsp(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, read.ID)
			outsideConn.ExpectSend(expectDataReady, nil)

			rdmaEngine.processReqFromInside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc).NotTo(HaveKey(read.ID))
		})

		It("should wait if outside connection is busy", func() {
			expectDataReady := mem.NewDataReadyRsp(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, read.ID)
			outsideConn.ExpectSend(expectDataReady, core.NewSendError())

			rdmaEngine.processReqFromInside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(1))
			Expect(rdmaEngine.originalSrc).To(HaveKey(read.ID))
		})
	})

	Context("write-done from inside", func() {
		var (
			write *mem.WriteReq
			done  *mem.DoneRsp
		)

		BeforeEach(func() {
			// Send a write from inside
			write = mem.NewWriteReq(6,
				remoteGPU.ToOutside, rdmaEngine.ToOutside, 0x100)
			rdmaEngine.ToOutside.Recv(write)
			expectWrite := mem.NewWriteReq(7,
				rdmaEngine.ToInside, localCache.ToOutside, 0x100)
			insideConn.ExpectSend(expectWrite, nil)
			rdmaEngine.processReqFromOutside(7)

			done = mem.NewDoneRsp(9,
				localCache.ToOutside, rdmaEngine.ToInside, write.ID)
			rdmaEngine.ToInside.Recv(done)
		})

		It("should send done to outside", func() {
			expectDone := mem.NewDoneRsp(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, write.ID)
			outsideConn.ExpectSend(expectDone, nil)

			rdmaEngine.processReqFromInside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc).NotTo(HaveKey(write.ID))
		})

		It("should wait if outside connection is busy", func() {
			expectDone := mem.NewDoneRsp(10,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, write.ID)
			outsideConn.ExpectSend(expectDone, core.NewSendError())

			rdmaEngine.processReqFromInside(10)

			Expect(outsideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToInside.Buf).To(HaveLen(1))
			Expect(rdmaEngine.originalSrc).To(HaveKey(write.ID))
		})
	})

	Context("Read from outside", func() {
		var read *mem.ReadReq

		BeforeEach(func() {
			read = mem.NewReadReq(6,
				remoteGPU.ToOutside, rdmaEngine.ToOutside, 0x100, 64)
			rdmaEngine.ToOutside.Recv(read)
		})

		It("should send read to inside", func() {
			expectRead := mem.NewReadReq(10,
				rdmaEngine.ToInside, localCache.ToOutside, 0x100, 64)
			insideConn.ExpectSend(expectRead, nil)

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc[read.ID]).To(
				BeIdenticalTo(remoteGPU.ToOutside))
		})

		It("should wait if outside connection is busy", func() {
			expectRead := mem.NewReadReq(10,
				rdmaEngine.ToInside, localCache.ToOutside, 0x100, 64)
			insideConn.ExpectSend(expectRead, core.NewSendError())

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(1))
		})
	})

	Context("Write from outside", func() {
		var write *mem.WriteReq

		BeforeEach(func() {
			write = mem.NewWriteReq(6,
				remoteGPU.ToOutside, rdmaEngine.ToOutside, 0x100)
			rdmaEngine.ToOutside.Recv(write)
		})

		It("should send write to inside", func() {
			expectRead := mem.NewWriteReq(10,
				rdmaEngine.ToInside, localCache.ToOutside, 0x100)
			insideConn.ExpectSend(expectRead, nil)

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc[write.ID]).To(
				BeIdenticalTo(remoteGPU.ToOutside))
		})

		It("should wait if outside connection is busy", func() {
			expectRead := mem.NewWriteReq(10,
				rdmaEngine.ToInside, localCache.ToOutside, 0x100)
			insideConn.ExpectSend(expectRead, core.NewSendError())

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(1))
		})
	})

	Context("DataReady from outside", func() {
		var (
			read      *mem.ReadReq
			dataReady *mem.DataReadyRsp
		)

		BeforeEach(func() {
			// Send a read from inside
			read = mem.NewReadReq(6,
				localCache.ToOutside, rdmaEngine.ToInside, 0x100, 64)
			rdmaEngine.ToInside.Recv(read)
			expectRead := mem.NewReadReq(7,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, 0x100, 64)
			outsideConn.ExpectSend(expectRead, nil)
			rdmaEngine.processReqFromInside(7)

			dataReady = mem.NewDataReadyRsp(12,
				remoteGPU.ToOutside, rdmaEngine.ToOutside, read.ID)
			rdmaEngine.ToOutside.Recv(dataReady)
		})

		It("should send data ready to outside", func() {
			expectDataReady := mem.NewDataReadyRsp(10,
				rdmaEngine.ToInside, localCache.ToOutside, read.ID)
			insideConn.ExpectSend(expectDataReady, nil)

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc).NotTo(HaveKey(read.ID))
		})

		It("should wait if outside connection is busy", func() {
			expectDataReady := mem.NewDataReadyRsp(10,
				rdmaEngine.ToInside, localCache.ToOutside, read.ID)
			insideConn.ExpectSend(expectDataReady, core.NewSendError())

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(1))
			Expect(rdmaEngine.originalSrc).To(HaveKey(read.ID))
		})
	})

	Context("write-done from outside", func() {
		var (
			write *mem.WriteReq
			done  *mem.DoneRsp
		)

		BeforeEach(func() {
			// Send a write from inside
			write = mem.NewWriteReq(6,
				localCache.ToOutside, rdmaEngine.ToInside, 0x100)
			rdmaEngine.ToInside.Recv(write)
			expectWrite := mem.NewWriteReq(7,
				rdmaEngine.ToOutside, remoteGPU.ToOutside, 0x100)
			outsideConn.ExpectSend(expectWrite, nil)
			rdmaEngine.processReqFromInside(7)

			done = mem.NewDoneRsp(9,
				remoteGPU.ToOutside, rdmaEngine.ToOutside, write.ID)
			rdmaEngine.ToOutside.Recv(done)
		})

		It("should send data ready to outside", func() {
			expectDone := mem.NewDoneRsp(10,
				rdmaEngine.ToInside, localCache.ToOutside, write.ID)
			insideConn.ExpectSend(expectDone, nil)

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(0))
			Expect(rdmaEngine.originalSrc).NotTo(HaveKey(write.ID))
		})

		It("should wait if outside connection is busy", func() {
			expectDone := mem.NewDoneRsp(10,
				rdmaEngine.ToInside, localCache.ToOutside, write.ID)
			insideConn.ExpectSend(expectDone, core.NewSendError())

			rdmaEngine.processReqFromOutside(10)

			Expect(insideConn.AllExpectedSent()).To(BeTrue())
			Expect(rdmaEngine.ToOutside.Buf).To(HaveLen(1))
			Expect(rdmaEngine.originalSrc).To(HaveKey(write.ID))
		})
	})
})
