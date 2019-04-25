package l1v

import (
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/mem"
)

var _ = Describe("Coalescer", func() {
	var (
		mockCtrl     *gomock.Controller
		topPort      *MockPort
		transactions []*transaction
		dirBuf       *MockBuffer
		c            coalescer
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		topPort = NewMockPort(mockCtrl)
		dirBuf = NewMockBuffer(mockCtrl)
		transactions = nil
		c = coalescer{
			log2BlockSize: 6,
			topPort:       topPort,
			transactions:  &transactions,
			dirBuf:        dirBuf,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should do nothing if no req", func() {
		topPort.EXPECT().Peek().Return(nil)
		madeProgress := c.Tick(10)
		Expect(madeProgress).To(BeFalse())
	})

	Context("read", func() {
		var (
			read1 *mem.ReadReq
			read2 *mem.ReadReq
		)

		BeforeEach(func() {
			read1 = mem.NewReadReq(10, nil, nil, 0x100, 4)
			read2 = mem.NewReadReq(10, nil, nil, 0x104, 4)

			topPort.EXPECT().Peek().Return(read1)
			topPort.EXPECT().Retrieve(gomock.Any())
			topPort.EXPECT().Peek().Return(read2)
			topPort.EXPECT().Retrieve(gomock.Any())
			c.Tick(10)
			c.Tick(11)
		})

		Context("not coalescable", func() {
			It("should send to dir stage", func() {
				read3 := mem.NewReadReq(10, nil, nil, 0x148, 4)

				dirBuf.EXPECT().CanPush().
					Return(true)
				dirBuf.EXPECT().Push(gomock.Any()).
					Do(func(trans *transaction) {
						Expect(trans.preCoalesceTransactions).To(HaveLen(2))
					})
				topPort.EXPECT().Peek().Return(read3)
				topPort.EXPECT().Retrieve(gomock.Any())

				madeProgress := c.Tick(13)

				Expect(madeProgress).To(BeTrue())
				Expect(transactions).To(HaveLen(3))
				Expect(c.toCoalesce).To(HaveLen(1))
			})

			It("should stall if cannot send to dir", func() {
				read3 := mem.NewReadReq(10, nil, nil, 0x148, 4)

				dirBuf.EXPECT().CanPush().
					Return(false)
				topPort.EXPECT().Peek().Return(read3)

				madeProgress := c.Tick(13)

				Expect(madeProgress).To(BeFalse())
				Expect(transactions).To(HaveLen(2))
				Expect(c.toCoalesce).To(HaveLen(2))
			})
		})

		Context("last in wave, coalescable", func() {
			It("should send to dir stage", func() {
				read3 := mem.NewReadReq(10, nil, nil, 0x108, 4)
				read3.IsLastInWave = true

				dirBuf.EXPECT().CanPush().
					Return(true)
				dirBuf.EXPECT().Push(gomock.Any()).
					Do(func(trans *transaction) {
						Expect(trans.preCoalesceTransactions).To(HaveLen(3))
					})
				topPort.EXPECT().Peek().Return(read3)
				topPort.EXPECT().Retrieve(gomock.Any())

				madeProgress := c.Tick(13)

				Expect(madeProgress).To(BeTrue())
				Expect(transactions).To(HaveLen(3))
				Expect(c.toCoalesce).To(HaveLen(0))
			})

			It("should stall if cannot send", func() {
				read3 := mem.NewReadReq(10, nil, nil, 0x108, 4)
				read3.IsLastInWave = true

				dirBuf.EXPECT().CanPush().
					Return(false)
				topPort.EXPECT().Peek().Return(read3)

				madeProgress := c.Tick(13)

				Expect(madeProgress).To(BeFalse())
				Expect(transactions).To(HaveLen(2))
				Expect(c.toCoalesce).To(HaveLen(2))
			})
		})

		Context("last in wave, not coalescable", func() {
			It("should send to dir stage", func() {
				read3 := mem.NewReadReq(10, nil, nil, 0x148, 4)
				read3.IsLastInWave = true

				dirBuf.EXPECT().CanPush().
					Return(true).Times(2)
				dirBuf.EXPECT().Push(gomock.Any()).
					Do(func(trans *transaction) {
						Expect(trans.preCoalesceTransactions).To(HaveLen(2))
					})
				dirBuf.EXPECT().Push(gomock.Any()).
					Do(func(trans *transaction) {
						Expect(trans.preCoalesceTransactions).To(HaveLen(1))
					})

				topPort.EXPECT().Peek().Return(read3)
				topPort.EXPECT().Retrieve(gomock.Any())
				madeProgress := c.Tick(13)

				Expect(madeProgress).To(BeTrue())
				Expect(transactions).To(HaveLen(3))
				Expect(c.toCoalesce).To(HaveLen(0))
			})

			It("should stall is cannot send to dir stage", func() {
				read3 := mem.NewReadReq(10, nil, nil, 0x148, 4)
				read3.IsLastInWave = true

				dirBuf.EXPECT().CanPush().
					Return(false)

				topPort.EXPECT().Peek().Return(read3)
				madeProgress := c.Tick(13)

				Expect(madeProgress).To(BeFalse())
				Expect(transactions).To(HaveLen(2))
				Expect(c.toCoalesce).To(HaveLen(2))
			})

			It("should stall is cannot send to dir stage in the second time",
				func() {
					read3 := mem.NewReadReq(10, nil, nil, 0x148, 4)
					read3.IsLastInWave = true

					dirBuf.EXPECT().CanPush().
						Return(true)
					dirBuf.EXPECT().Push(gomock.Any()).
						Do(func(trans *transaction) {
							Expect(trans.preCoalesceTransactions).To(HaveLen(2))
						})
					dirBuf.EXPECT().CanPush().Return(false)
					topPort.EXPECT().Peek().Return(read3)

					madeProgress := c.Tick(13)

					Expect(madeProgress).To(BeTrue())
					Expect(transactions).To(HaveLen(2))
					Expect(c.toCoalesce).To(HaveLen(0))
				})
		})
	})

	Context("write", func() {
		It("should coalesce write", func() {
			write1 := mem.NewWriteReq(10, nil, nil, 0x104)
			write1.Data = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 9, 9}
			write1.DirtyMask = []bool{
				true, true, true, true,
				false, false, false, false,
				true, true, true, true,
			}

			write2 := mem.NewWriteReq(10, nil, nil, 0x108)
			write2.Data = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 9, 9}
			write2.DirtyMask = []bool{
				true, true, true, true,
				true, true, true, true,
				false, false, false, false,
			}
			write2.IsLastInWave = true

			topPort.EXPECT().Peek().Return(write1)
			topPort.EXPECT().Peek().Return(write2)
			topPort.EXPECT().Retrieve(gomock.Any()).Times(2)
			dirBuf.EXPECT().CanPush().Return(true)
			dirBuf.EXPECT().Push(gomock.Any()).Do(func(trans *transaction) {
				Expect(trans.write.Address).To(Equal(uint64(0x100)))
				Expect(trans.write.Data).To(Equal([]byte{
					0, 0, 0, 0,
					1, 2, 3, 4,
					1, 2, 3, 4,
					5, 6, 7, 8,
					0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0,
				}))
				Expect(trans.write.DirtyMask).To(Equal([]bool{
					false, false, false, false, true, true, true, true,
					true, true, true, true, true, true, true, true,
					false, false, false, false, false, false, false, false,
					false, false, false, false, false, false, false, false,
					false, false, false, false, false, false, false, false,
					false, false, false, false, false, false, false, false,
					false, false, false, false, false, false, false, false,
					false, false, false, false, false, false, false, false,
				}))
			})

			madeProgress := c.Tick(10)
			Expect(madeProgress).To(BeTrue())

			madeProgress = c.Tick(11)
			Expect(madeProgress).To(BeTrue())
		})
	})
})
