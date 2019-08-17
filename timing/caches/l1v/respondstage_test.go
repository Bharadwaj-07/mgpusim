package l1v

import (
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/akita"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/util/akitaext"
)

var _ = Describe("Respond Stage", func() {
	var (
		mockCtrl *gomock.Controller
		cache    *Cache
		topPort  *MockPort
		s        *respondStage
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		topPort = NewMockPort(mockCtrl)
		cache = &Cache{
			TopPort: topPort,
		}
		cache.TickingComponent = akitaext.NewTickingComponent(
			"cache", nil, 1, cache)
		s = &respondStage{cache: cache}

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should do nothing if there is no transaction", func() {
		madeProgress := s.Tick(10)
		Expect(madeProgress).To(BeFalse())
	})

	Context("read", func() {
		var (
			read  *mem.ReadReq
			trans *transaction
		)

		BeforeEach(func() {
			read = mem.NewReadReq(5, nil, nil, 0x100, 4)
			trans = &transaction{read: read}
			cache.transactions = append(cache.transactions, trans)
		})

		It("should do nothing if the transaction is not ready", func() {
			madeProgress := s.Tick(10)
			Expect(madeProgress).To(BeFalse())
		})

		It("should stall if cannot send to top", func() {
			trans.data = []byte{1, 2, 3, 4}
			trans.done = true
			topPort.EXPECT().Send(gomock.Any()).Return(&akita.SendError{})

			madeProgress := s.Tick(10)

			Expect(madeProgress).To(BeFalse())
		})

		It("should send data ready to top", func() {
			trans.data = []byte{1, 2, 3, 4}
			trans.done = true
			topPort.EXPECT().Send(gomock.Any()).
				Do(func(dr *mem.DataReadyRsp) {
					Expect(dr.RespondTo).To(Equal(read.GetID()))
					Expect(dr.Data).To(Equal([]byte{1, 2, 3, 4}))
				})

			madeProgress := s.Tick(10)

			Expect(madeProgress).To(BeTrue())
			Expect(cache.transactions).NotTo(ContainElement((trans)))
		})
	})

	Context("write", func() {
		var (
			write *mem.WriteReq
			trans *transaction
		)

		BeforeEach(func() {
			write = mem.NewWriteReq(5, nil, nil, 0x100)
			trans = &transaction{write: write}
			cache.transactions = append(cache.transactions, trans)
		})

		It("should do nothing if the transaction is not ready", func() {
			madeProgress := s.Tick(10)
			Expect(madeProgress).To(BeFalse())
		})

		It("should stall if cannot send to top", func() {
			trans.done = true
			topPort.EXPECT().Send(gomock.Any()).Return(&akita.SendError{})

			madeProgress := s.Tick(10)

			Expect(madeProgress).To(BeFalse())
		})

		It("should send data ready to top", func() {
			trans.data = []byte{1, 2, 3, 4}
			trans.done = true
			topPort.EXPECT().Send(gomock.Any()).
				Do(func(done *mem.DoneRsp) {
					Expect(done.RespondTo).To(Equal(write.GetID()))
				})

			madeProgress := s.Tick(10)

			Expect(madeProgress).To(BeTrue())
			Expect(cache.transactions).NotTo(ContainElement((trans)))
		})
	})

})
