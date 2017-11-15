package emu

import (
	"bytes"
	"encoding/binary"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/yaotsu/gcn3/insts"
)

var _ = Describe("ScratchpadPreparer", func() {
	var (
		sp *ScratchpadPreparerImpl
		wf *Wavefront
	)

	BeforeEach(func() {
		sp = NewScratchpadPreparerImpl()
		wf = NewWavefront(nil)
	})

	It("should prepare for SOP2", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Sop2
		inst.Src0 = insts.NewSRegOperand(0, 0, 1)
		inst.Src1 = insts.NewIntOperand(1, 1)
		wf.inst = inst

		wf.WriteReg(insts.SReg(0), 1, 0, insts.Uint32ToBytes(517))
		wf.SCC = 1

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsSOP2()
		binary.Read(bytes.NewBuffer(wf.scratchpad), binary.LittleEndian, layout)
		Expect(layout.SRC0).To(Equal(uint64(517)))
		Expect(layout.SRC1).To(Equal(uint64(1)))
		Expect(layout.SCC).To(Equal(byte(1)))

	})

	It("should prepare for VOP1", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vop1
		inst.Src0 = insts.NewVRegOperand(0, 0, 1)
		wf.inst = inst

		for i := 0; i < 64; i++ {
			wf.WriteReg(insts.VReg(0), 1, i, insts.Uint32ToBytes(uint32(i)))
		}
		wf.WriteReg(insts.Regs[insts.Vcc], 1, 0,
			insts.Uint64ToBytes(uint64(0xffff0000ffff0000)))

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsVOP1()
		for i := 0; i < 64; i++ {
			Expect(layout.SRC0[i]).To(Equal(uint64(i)))
		}
		Expect(layout.VCC).To(Equal(uint64(0xffff0000ffff0000)))
	})

	It("should prepare for VOP2", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vop2
		inst.Src0 = insts.NewVRegOperand(0, 0, 2)
		inst.Src1 = insts.NewVRegOperand(2, 2, 2)
		wf.inst = inst

		for i := 0; i < 64; i++ {
			wf.WriteReg(insts.VReg(0), 2, i, insts.Uint64ToBytes(uint64(i)))
			wf.WriteReg(insts.VReg(2), 2, i, insts.Uint64ToBytes(uint64(i)))
		}
		wf.WriteReg(insts.Regs[insts.Vcc], 1, 0,
			insts.Uint64ToBytes(uint64(0xffff0000ffff0000)))

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsVOP3A()
		for i := 0; i < 64; i++ {
			Expect(layout.SRC0[i]).To(Equal(uint64(i)))
			Expect(layout.SRC1[i]).To(Equal(uint64(i)))
		}
		Expect(layout.VCC).To(Equal(uint64(0xffff0000ffff0000)))
	})

	It("should prepare for VOP3", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vop3
		inst.Src0 = insts.NewVRegOperand(0, 0, 2)
		inst.Src1 = insts.NewVRegOperand(2, 2, 2)
		inst.Src2 = insts.NewIntOperand(1, 1)
		wf.inst = inst

		for i := 0; i < 64; i++ {
			wf.WriteReg(insts.VReg(0), 2, i, insts.Uint64ToBytes(uint64(i)))
			wf.WriteReg(insts.VReg(2), 2, i, insts.Uint64ToBytes(uint64(i)))
		}
		wf.WriteReg(insts.Regs[insts.Vcc], 1, 0,
			insts.Uint64ToBytes(uint64(0xffff0000ffff0000)))

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsVOP3A()
		for i := 0; i < 64; i++ {
			Expect(layout.SRC0[i]).To(Equal(uint64(i)))
			Expect(layout.SRC1[i]).To(Equal(uint64(i)))
			Expect(layout.SRC2[i]).To(Equal(uint64(1)))
		}
		Expect(layout.VCC).To(Equal(uint64(0xffff0000ffff0000)))
	})

	It("should prepare for VOPC", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vopc
		inst.Src0 = insts.NewVRegOperand(0, 0, 1)
		inst.Src1 = insts.NewVRegOperand(1, 1, 1)
		wf.inst = inst

		for i := 0; i < 64; i++ {
			wf.WriteReg(insts.VReg(0), 1, i, insts.Uint64ToBytes(uint64(i)))
			wf.WriteReg(insts.VReg(1), 1, i, insts.Uint64ToBytes(uint64(i+1)))
		}
		wf.VCC = 1
		wf.Exec = 0xff

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsVOPC()
		Expect(layout.EXEC).To(Equal(uint64(0xff)))
		for i := 0; i < 64; i++ {
			Expect(layout.SRC0[i]).To(Equal(uint64(i)))
			Expect(layout.SRC1[i]).To(Equal(uint64(i + 1)))
		}
	})

	It("should prepare for Flat", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Flat
		inst.Addr = insts.NewVRegOperand(0, 0, 2)
		inst.Data = insts.NewVRegOperand(2, 2, 4)
		wf.inst = inst

		for i := 0; i < 64; i++ {
			wf.WriteReg(insts.VReg(0), 2, i,
				insts.Uint64ToBytes(uint64(i+1024)))
			wf.WriteReg(insts.VReg(2), 1, i, insts.Uint32ToBytes(uint32(i)))
			wf.WriteReg(insts.VReg(3), 1, i, insts.Uint32ToBytes(uint32(i)))
			wf.WriteReg(insts.VReg(4), 1, i, insts.Uint32ToBytes(uint32(i)))
			wf.WriteReg(insts.VReg(5), 1, i, insts.Uint32ToBytes(uint32(i)))
		}

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsFlat()
		for i := 0; i < 64; i++ {
			log.Printf("iter %d", i)
			Expect(layout.ADDR[i]).To(Equal(uint64(i + 1024)))
			Expect(layout.DATA[i*4+0]).To(Equal(uint32(i)))
			Expect(layout.DATA[i*4+1]).To(Equal(uint32(i)))
			Expect(layout.DATA[i*4+2]).To(Equal(uint32(i)))
			Expect(layout.DATA[i*4+3]).To(Equal(uint32(i)))
		}
	})

	It("should prepare for SMEM", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Smem
		inst.Opcode = 18
		inst.Data = insts.NewSRegOperand(0, 0, 4)
		inst.Offset = insts.NewIntOperand(1, 1)
		inst.Base = insts.NewSRegOperand(4, 4, 2)
		wf.inst = inst

		wf.WriteReg(insts.SReg(0), 1, 0, insts.Uint32ToBytes(100))
		wf.WriteReg(insts.SReg(1), 1, 0, insts.Uint32ToBytes(101))
		wf.WriteReg(insts.SReg(2), 1, 0, insts.Uint32ToBytes(102))
		wf.WriteReg(insts.SReg(3), 1, 0, insts.Uint32ToBytes(103))
		wf.WriteReg(insts.SReg(4), 2, 0, insts.Uint64ToBytes(1024))

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsSMEM()
		Expect(layout.DATA[0]).To(Equal(uint32(100)))
		Expect(layout.DATA[1]).To(Equal(uint32(101)))
		Expect(layout.DATA[2]).To(Equal(uint32(102)))
		Expect(layout.DATA[3]).To(Equal(uint32(103)))
		Expect(layout.Offset).To(Equal(uint64(1)))
		Expect(layout.Base).To(Equal(uint64(1024)))

	})

	It("should prepare for SOPP", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Sopp
		inst.SImm16 = insts.NewIntOperand(1, 1)
		wf.PC = 160
		wf.SCC = 1
		wf.inst = inst

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsSOPP()
		Expect(layout.IMM).To(Equal(uint64(1)))
		Expect(layout.PC).To(Equal(uint64(160)))
		Expect(layout.SCC).To(Equal(byte(1)))
	})

	It("should prepare for SOPC", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Sopc
		inst.Src0 = insts.NewSRegOperand(0, 0, 1)
		wf.WriteReg(insts.SReg(0), 1, 0, insts.Uint32ToBytes(100))
		inst.Src1 = insts.NewIntOperand(192, 64)
		wf.inst = inst

		sp.Prepare(wf, wf)

		layout := wf.Scratchpad().AsSOPC()
		Expect(layout.SRC0).To(Equal(uint64(100)))
		Expect(layout.SRC1).To(Equal(uint64(64)))
	})

	It("should commit for SOP2", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Sop2
		inst.Dst = insts.NewSRegOperand(0, 0, 1)
		wf.inst = inst

		layout := wf.Scratchpad().AsSOP2()
		layout.DST = 517
		layout.SCC = 1

		sp.Commit(wf, wf)

		Expect(wf.SCC).To(Equal(byte(1)))
		Expect(wf.SRegValue(0)).To(Equal(uint32(517)))
	})

	It("should commit for VOP1", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vop1
		inst.Dst = insts.NewVRegOperand(0, 0, 1)
		wf.inst = inst

		layout := wf.Scratchpad().AsVOP1()
		for i := 0; i < 64; i++ {
			layout.DST[i] = uint64(i)
		}
		layout.VCC = uint64(0xffff0000ffff0000)

		sp.Commit(wf, wf)

		for i := 0; i < 64; i++ {
			Expect(wf.VRegValue(i, 0)).To(Equal(uint32(i)))
		}
		Expect(wf.VCC).To(Equal(uint64(0xffff0000ffff0000)))
	})

	It("should commit for VOP2", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vop2
		inst.Dst = insts.NewVRegOperand(0, 0, 1)
		wf.inst = inst

		layout := wf.Scratchpad().AsVOP2()
		for i := 0; i < 64; i++ {
			layout.DST[i] = uint64(i)
		}
		layout.VCC = uint64(0xffff0000ffff0000)

		sp.Commit(wf, wf)

		for i := 0; i < 64; i++ {
			Expect(wf.VRegValue(i, 0)).To(Equal(uint32(i)))
		}
		Expect(wf.VCC).To(Equal(uint64(0xffff0000ffff0000)))
	})

	It("should commit for VOP3", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vop3
		inst.Dst = insts.NewVRegOperand(0, 0, 1)
		wf.inst = inst

		layout := wf.Scratchpad().AsVOP3A()
		for i := 0; i < 64; i++ {
			layout.DST[i] = uint64(i)
		}
		layout.VCC = uint64(0xffff0000ffff0000)

		sp.Commit(wf, wf)

		for i := 0; i < 64; i++ {
			Expect(wf.VRegValue(i, 0)).To(Equal(uint32(i)))
		}
		Expect(wf.VCC).To(Equal(uint64(0xffff0000ffff0000)))
	})

	It("should commit VOPC", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Vopc
		wf.inst = inst

		layout := wf.Scratchpad().AsVOPC()
		layout.VCC = uint64(0xff)

		sp.Commit(wf, wf)

		Expect(wf.VCC).To(Equal(uint64(0xff)))
	})

	It("should commit for FLAT", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Flat
		inst.Dst = insts.NewVRegOperand(3, 3, 4)
		wf.inst = inst

		layout := wf.Scratchpad().AsFlat()
		for i := 0; i < 64; i++ {
			layout.DST[i*4+0] = uint32(i)
			layout.DST[i*4+1] = uint32(i)
			layout.DST[i*4+2] = uint32(i)
			layout.DST[i*4+3] = uint32(i)
		}

		sp.Commit(wf, wf)

		for i := 0; i < 64; i++ {
			Expect(wf.VRegValue(i, 3)).To(Equal(uint32(i)))
			Expect(wf.VRegValue(i, 4)).To(Equal(uint32(i)))
			Expect(wf.VRegValue(i, 5)).To(Equal(uint32(i)))
			Expect(wf.VRegValue(i, 6)).To(Equal(uint32(i)))
		}
	})

	It("should commit for SMEM", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Smem
		inst.Opcode = 4
		inst.Data = insts.NewSRegOperand(0, 0, 16)
		wf.inst = inst

		layout := wf.Scratchpad().AsSMEM()
		for i := 0; i < 16; i++ {
			layout.DST[i] = uint32(i)
		}

		sp.Commit(wf, wf)

		for i := 0; i < 16; i++ {
			Expect(wf.SRegValue(i)).To(Equal(uint32(i)))
		}
	})

	It("should commit for SOPC", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Sopc
		wf.inst = inst

		layout := wf.Scratchpad().AsSOPC()
		layout.SCC = 1

		sp.Commit(wf, wf)

		Expect(wf.SCC).To(Equal(byte(1)))
	})

	It("should commit for SOPP", func() {
		inst := insts.NewInst()
		inst.FormatType = insts.Sopp
		wf.inst = inst
		wf.PC = 0

		layout := wf.Scratchpad().AsSOPP()
		layout.PC = 164

		sp.Commit(wf, wf)

		Expect(wf.PC).To(Equal(uint64(164)))
	})

})
