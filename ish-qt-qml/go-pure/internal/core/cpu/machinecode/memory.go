package machinecode

import (
	asm "github.com/twitchyliquid64/golang-asm"
	"github.com/twitchyliquid64/golang-asm/obj"
	"github.com/twitchyliquid64/golang-asm/obj/x86"
)

func emitX86Load64(b *asm.Builder, i Instruction) {
	p := newX86Prog(b, x86.AMOVQ)
	p.From.Type, p.From.Reg = obj.TYPE_MEM, x86Reg(i.Src)
	p.From.Offset = i.Imm
	p.To.Type, p.To.Reg = obj.TYPE_REG, x86Reg(i.Dst)
	b.AddInstruction(p)
}

func emitX86Store64(b *asm.Builder, i Instruction) {
	p := newX86Prog(b, x86.AMOVQ)
	p.From.Type, p.From.Reg = obj.TYPE_REG, x86Reg(i.Src)
	p.To.Type, p.To.Reg = obj.TYPE_MEM, x86Reg(i.Dst)
	p.To.Offset = i.Imm
	b.AddInstruction(p)
}
