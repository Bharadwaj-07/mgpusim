package insts

import (
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
	InstName string
	Opcode   Opcode
	Format   *Format
	ID       int
	ExeUnit  ExeUnit
}

// An Inst is a GCN3 instruction
type Inst struct {
	*Format
	*InstType
	ByteSize int

	Src0 *Operand
	Src1 *Operand
	Src2 *Operand
	Dst  *Operand

	Addr   *Operand
	Data   *Operand
	Base   *Operand
	Offset *Operand
	SImm16 *Operand

	Abs                 int
	Omod                int
	Neg                 int
	SystemLevelCoherent bool
	GlobalLevelCoherent bool
	TextureFailEnable   bool
	Imm                 bool
	Clamp               bool

	//Fields for SDWA extensions
	IsSdwa     bool
	Dst_Sel    SDWASelect
	Dst_Unused uint32
	Src0_Sel   SDWASelect
	Src0_Sext  bool
	Src0_Neg   bool
	Src0_Abs   bool
	Src1_Sel   SDWASelect
	Src1_Sext  bool
	Src1_Neg   bool
	Src1_Abs   bool
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

func (i Inst) soppString() string {
	operandStr := ""
	if i.Opcode == 12 { // S_WAITCNT
		if extractBits(uint32(i.SImm16.IntValue), 0, 3) == 0 {
			operandStr += " vmcnt(0)"
		}
		if extractBits(uint32(i.SImm16.IntValue), 8, 12) == 0 {
			operandStr += " lgkmcnt(0)"
		}
	} else if i.Opcode == 1 || i.Opcode == 10 {

	} else {
		operandStr = " " + i.SImm16.String()
	}
	s := i.InstName + operandStr
	return s
}

func (i Inst) vop2String() string {
	s := fmt.Sprintf("%s %s", i.InstName, i.Dst.String())

	switch i.Opcode {
	case 25, 26, 27, 28, 29, 30:
		s += ", vcc"
	}

	s += fmt.Sprintf(", %s, %s", i.Src0.String(), i.Src1.String())

	switch i.Opcode {
	case 28, 29:
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
	s += sdwaSelectString(i.Dst_Sel)
	s += " src0_sel:"
	s += sdwaSelectString(i.Src0_Sel)
	s += " src1_sel:"
	s += sdwaSelectString(i.Src1_Sel)

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

func (i Inst) vop3String() string {
	// TODO: Lots of things not considered here
	s := fmt.Sprintf("%s %s, %s, %s",
		i.InstName, i.Dst.String(),
		i.Src0.String(), i.Src1.String())

	if i.Src2 != nil {
		s += ", " + i.Src2.String()
	}
	return s
}

func (i Inst) sop1String() string {
	return fmt.Sprintf("%s %s, %s", i.InstName, i.Dst.String(), i.Src0.String())
}

func (i Inst) sopkString() string {
	s := fmt.Sprintf("%s,%s,%s",
		i.InstName, i.Dst.String(), i.SImm16.String())

	return s
}
func (i Inst) String() string {
	switch i.FormatType {
	case Sop2:
		return i.sop2String()
	case Smem:
		return i.smemString()
	case Vop1:
		return i.vop1String()
	case Vop2:
		return i.vop2String()
	case Flat:
		return i.flatString()
	case Sopp:
		return i.soppString()
	case Vopc:
		return i.vopcString()
	case Sopc:
		return i.sopcString()
	case Vop3:
		return i.vop3String()
	case Sop1:
		return i.sop1String()
	case Sopk:
		return i.sopkString()
	default:
		return i.InstName
	}
}
