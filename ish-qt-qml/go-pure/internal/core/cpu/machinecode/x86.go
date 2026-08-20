package machinecode

import (
	asm "github.com/twitchyliquid64/golang-asm"
	"github.com/twitchyliquid64/golang-asm/obj"
	"github.com/twitchyliquid64/golang-asm/obj/x86"
)

type x86Handler func(*asm.Builder, Instruction)

var x86Handlers = [...]x86Handler{
	emitX86NOP, emitX86MOV, emitX86ADD, emitX86SUB, emitX86RET, emitX86Syscall,
	emitX86MOVReg, emitX86ADDReg, emitX86SUBReg, emitX86AND, emitX86OR, emitX86XOR, emitX86CMP,
	emitX86Load64, emitX86Store64,
}

func EmitX86(in []Instruction) ([]byte, error) {
	b, err := asm.NewBuilder("amd64", len(in)+1)
	if err != nil {
		return nil, err
	}
	for _, inst := range in {
		if !validX86Op(inst.Op) {
			return nil, ErrUnsupported
		}
		x86Handlers[inst.Op](b, inst)
	}
	return b.Assemble(), nil
}

func validX86Op(op Op) bool { return int(op) < len(x86Handlers) }

func newX86Prog(b *asm.Builder, as obj.As) *obj.Prog {
	p := b.NewProg()
	p.As = as
	return p
}

func x86ConstReg(p *obj.Prog, reg int16, value int64) {
	p.From.Type, p.From.Offset = obj.TYPE_CONST, value
	p.To.Type, p.To.Reg = obj.TYPE_REG, x86Reg(reg)
}

func x86Reg(index int16) int16 { return x86.REG_AX + index }

func emitX86NOP(b *asm.Builder, _ Instruction) {
	p := newX86Prog(b, x86.ANOPL)
	p.From.Type, p.From.Reg = obj.TYPE_REG, x86.REG_AX
	b.AddInstruction(p)
}

func emitX86MOV(b *asm.Builder, i Instruction) {
	p := newX86Prog(b, x86.AMOVQ)
	x86ConstReg(p, i.Dst, i.Imm)
	b.AddInstruction(p)
}

func emitX86ADD(b *asm.Builder, i Instruction) {
	p := newX86Prog(b, x86.AADDQ)
	x86ConstReg(p, i.Dst, i.Imm)
	b.AddInstruction(p)
}

func emitX86SUB(b *asm.Builder, i Instruction) {
	p := newX86Prog(b, x86.ASUBQ)
	x86ConstReg(p, i.Dst, i.Imm)
	b.AddInstruction(p)
}

func emitX86RET(b *asm.Builder, _ Instruction) {
	b.AddInstruction(newX86Prog(b, obj.ARET))
}
