package insts

import (
	"debug/elf"
	"fmt"
	"log"
	"strings"
)

// ExeUnit defines which execution unit should execute the instruction
type ExeUnit int

// Defines all possible execution units
const (
	ExeUnitVALU ExeUnit = iota
	ExeUnitScalar
	ExeUnitVMem
	ExeUnitBranch
	ExeUnitLDS
	ExeUnitGDS
	ExeUnitSpecial
)

// SDWASelectType defines the sub-dword selection type
type SDWASelect uint32

// Defines all possible sub-dword selection type
const (
	SDWASelectByte0 SDWASelect = 0x000000ff
	SDWASelectByte1 SDWASelect = 0x0000ff00
	SDWASelectByte2 SDWASelect = 0x00ff0000
	SDWASelectByte3 SDWASelect = 0xff000000
	SDWASelectWord0 SDWASelect = 0x0000ffff
	SDWASelectWord1 SDWASelect = 0xffff0000
	SDWASelectDWord SDWASelect = 0xffffffff
)

// sdwaSelectString stringify SDWA select types
func sdwaSelectString(sdwaSelect SDWASelect) string {
	switch sdwaSelect {
	case SDWASelectByte0:
		return "BYTE_0"
	case SDWASelectByte1:
		return "BYTE_1"
	case SDWASelectByte2:
		return "BYTE_2"
	case SDWASelectByte3:
		return "BYTE_3"
	case SDWASelectWord0:
		return "WORD_0"
	case SDWASelectWord1:
		return "WORD_1"
	case SDWASelectDWord:
		return "DWORD"
	default:
		log.Panic("unknown SDWASelect type")
		return ""
	}
}

// A InstType represents an instruction type. For example s_barrier instruction
// is a instruction type
type InstType struct {
	InstName  string
	Opcode    Opcode
	Format    *Format
	ID        int
	ExeUnit   ExeUnit
	DSTWidth  int
	SRC0Width int
	SRC1Width int
	SRC2Width int
	SDSTWidth int
}

// An Inst is a GCN3 instruction
type Inst struct {
	*Format
	*InstType
	ByteSize int
	PC       uint64

	Src0 *Operand
	Src1 *Operand
	Src2 *Operand
	Dst  *Operand
	SDst *Operand // For VOP3b

	Addr   *Operand
	Data   *Operand
	Data1  *Operand
	Base   *Operand
	Offset *Operand
	SImm16 *Operand

	Abs                 int
	Omod                int
	Neg                 int
	Offset0             uint8
	Offset1             uint8
	SystemLevelCoherent bool
	GlobalLevelCoherent bool
	TextureFailEnable   bool
	Imm                 bool
	Clamp               bool
	GDS                 bool
	VMCNT               int
	LKGMCNT             int

	//Fields for SDWA extensions
	IsSdwa    bool
	DstSel    SDWASelect
	DstUnused uint32
	Src0Sel   SDWASelect
	Src0Sext  bool
	Src0Neg   bool
	Src0Abs   bool
	Src1Sel   SDWASelect
	Src1Sext  bool
	Src1Neg   bool
	Src1Abs   bool
}

// NewInst creates a zero-filled instruction
func NewInst() *Inst {
	i := new(Inst)
	i.Format = new(Format)
	i.InstType = new(InstType)
	return i
}

func (i Inst) sop2String() string {
	return i.InstName + " " +
		i.Dst.String() + ", " +
		i.Src0.String() + ", " +
		i.Src1.String()
}

func (i Inst) vop1String() string {
	return i.InstName + " " +
		i.Dst.String() + ", " +
		i.Src0.String()
}

func (i Inst) flatString() string {
	var s string
	if i.Opcode >= 16 && i.Opcode <= 23 {
		s = i.InstName + " " + i.Dst.String() + ", " +
			i.Addr.String()
	} else if i.Opcode >= 24 && i.Opcode <= 31 {
		s = i.InstName + " " + i.Addr.String() + ", " +
			i.Data.String()
	}
	return s
}

func (i Inst) smemString() string {
	// TODO: Consider store instructions, and the case if imm = 0
	s := fmt.Sprintf("%s %s, %s, %#x",
		i.InstName, i.Data.String(), i.Base.String(), uint16(i.Offset.IntValue))
	return s
}

func (i Inst) soppString(file *elf.File) string {
	operandStr := ""
	if i.Opcode == 12 { // S_WAITCNT
		operandStr = i.waitcntOperandString()
	} else if i.Opcode >= 2 && i.Opcode <= 9 { // Branch
		symbolFound := false
		if file != nil {
			imm := int16(uint16(i.SImm16.IntValue))
			target := i.PC + uint64(imm*4) + 4
			symbols, _ := file.Symbols()
			for _, symbol := range symbols {
				if symbol.Value == target {
					operandStr = " " + symbol.Name
					symbolFound = true
				}
			}

		}
		if !symbolFound {
			operandStr = " " + i.SImm16.String()
		}
	} else if i.Opcode == 1 || i.Opcode == 10 {
		// Does not print anything
	} else {
		operandStr = " " + i.SImm16.String()
	}
	s := i.InstName + operandStr
	return s
}

func (i Inst) waitcntOperandString() string {
	operandStr := ""
	if i.VMCNT != 15 {
		operandStr += fmt.Sprintf(" vmcnt(%d)", i.VMCNT)
	}

	if i.LKGMCNT != 15 {
		operandStr += fmt.Sprintf(" lgkmcnt(%d)", i.LKGMCNT)
	}
	return operandStr
}

func (i Inst) vop2String() string {
	s := fmt.Sprintf("%s %s", i.InstName, i.Dst.String())

	switch i.Opcode {
	case 25, 26, 27, 28, 29, 30:
		s += ", vcc"
	}

	s += fmt.Sprintf(", %s, %s", i.Src0.String(), i.Src1.String())

	switch i.Opcode {
	case 0, 28, 29:
		s += ", vcc"
	}

	if i.IsSdwa {
		s += i.sdwaVOP2String()
	}

	return s
}

func (i Inst) sdwaVOP2String() string {
	s := ""

	s += " dst_sel:"
	s += sdwaSelectString(i.DstSel)
	s += " src0_sel:"
	s += sdwaSelectString(i.Src0Sel)
	s += " src1_sel:"
	s += sdwaSelectString(i.Src1Sel)

	return s
}

func (i Inst) vopcString() string {
	dst := "vcc"
	if strings.Contains(i.InstName, "cmpx") {
		dst = "exec"
	}

	return fmt.Sprintf("%s %s, %s, %s",
		i.InstName, dst, i.Src0.String(), i.Src1.String())
}

func (i Inst) sopcString() string {
	return fmt.Sprintf("%s %s, %s",
		i.InstName, i.Src0.String(), i.Src1.String())
}

func (i Inst) vop3aString() string {
	// TODO: Lots of things not considered here
	s := fmt.Sprintf("%s %s, %s, %s",
		i.InstName, i.Dst.String(),
		i.Src0.String(), i.Src1.String())

	if i.Src2 != nil {
		s += ", " + i.Src2.String()
	}
	return s
}

func (i Inst) vop3bString() string {
	s := i.InstName + " "

	if i.Dst != nil {
		s += i.Dst.String() + ", "
	}

	s += fmt.Sprintf("%s, %s, %s",
		i.SDst.String(),
		i.Src0.String(),
		i.Src1.String(),
	)

	if i.Opcode != 281 && i.Src2 != nil {
		s += ", " + i.Src2.String()
	}

	return s
}

func (i Inst) sop1String() string {
	return fmt.Sprintf("%s %s, %s", i.InstName, i.Dst.String(), i.Src0.String())
}

func (i Inst) sopkString() string {
	s := fmt.Sprintf("%s %s, 0x%x",
		i.InstName, i.Dst.String(), i.SImm16.IntValue)

	return s
}

func (i Inst) dsString() string {
	s := i.InstName + " "
	switch i.Opcode {
	case 54, 55, 56, 57, 58, 59, 60, 118, 119, 120:
		s += i.Dst.String() + ", "
	}

	s += i.Addr.String()

	if i.SRC0Width > 0 {
		s += ", " + i.Data.String()
	}

	if i.SRC1Width > 0 {
		s += ", " + i.Data1.String()
	}

	switch i.Opcode {
	case 13, 54:
		if i.Offset0 > 0 {
			s += fmt.Sprintf(" offset:%d", i.Offset0)
		}
	default:
		if i.Offset0 > 0 {
			s += fmt.Sprintf(" offset0:%d", i.Offset0)
		}

		if i.Offset1 > 0 {
			s += fmt.Sprintf(" offset1:%d", i.Offset1)
		}
	}

	return s
}

// String returns the disassembly of an instruction
func (i Inst) String(file *elf.File) string {
	switch i.FormatType {
	case SOP2:
		return i.sop2String()
	case SMEM:
		return i.smemString()
	case VOP1:
		return i.vop1String()
	case VOP2:
		return i.vop2String()
	case FLAT:
		return i.flatString()
	case SOPP:
		return i.soppString(file)
	case VOPC:
		return i.vopcString()
	case SOPC:
		return i.sopcString()
	case VOP3a:
		return i.vop3aString()
	case VOP3b:
		return i.vop3bString()
	case SOP1:
		return i.sop1String()
	case SOPK:
		return i.sopkString()
	case DS:
		return i.dsString()
	default:
		log.Panic("Unknown instruction format type.")
		return i.InstName
	}
}
