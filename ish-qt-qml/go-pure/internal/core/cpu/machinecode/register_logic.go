package machinecode

import (
	asm "github.com/twitchyliquid64/golang-asm"
	"github.com/twitchyliquid64/golang-asm/obj"
	"github.com/twitchyliquid64/golang-asm/obj/x86"
)

func emitX86MOVReg(b *asm.Builder, i Instruction) {
	b.AddInstruction(regProg(b, x86.AMOVQ, i))
}

func emitX86ADDReg(b *asm.Builder, i Instruction) {
	b.AddInstruction(regProg(b, x86.AADDQ, i))
}

func emitX86SUBReg(b *asm.Builder, i Instruction) {
	b.AddInstruction(regProg(b, x86.ASUBQ, i))
}

func regProg(b *asm.Builder, as obj.As, i Instruction) *obj.Prog {
	p := newX86Prog(b, as)
	p.From.Type, p.From.Reg = obj.TYPE_REG, x86Reg(i.Src)
	p.To.Type, p.To.Reg = obj.TYPE_REG, x86Reg(i.Dst)
	return p
}

func emitX86AND(b *asm.Builder, i Instruction) {
	b.AddInstruction(logicProg(b, x86.AANDQ, i))
}

func emitX86OR(b *asm.Builder, i Instruction) {
	b.AddInstruction(logicProg(b, x86.AORQ, i))
}

func emitX86XOR(b *asm.Builder, i Instruction) {
	b.AddInstruction(logicProg(b, x86.AXORQ, i))
}

func emitX86CMP(b *asm.Builder, i Instruction) {
	p := newX86Prog(b, x86.ACMPQ)
	p.From.Type, p.From.Reg = obj.TYPE_REG, x86Reg(i.Dst)
	p.To.Type, p.To.Offset = obj.TYPE_CONST, i.Imm
	b.AddInstruction(p)
}

func logicProg(b *asm.Builder, as obj.As, i Instruction) *obj.Prog {
	p := newX86Prog(b, as)
	x86ConstReg(p, i.Dst, i.Imm)
	return p
}
