package machinecode

import (
	asm "github.com/twitchyliquid64/golang-asm"
	"github.com/twitchyliquid64/golang-asm/obj"
	"github.com/twitchyliquid64/golang-asm/obj/arm64"
)

type arm64Handler func(*asm.Builder, Instruction)

var arm64Handlers = [...]arm64Handler{
	emitARM64NOP, emitARM64MOV, emitARM64ADD, emitARM64SUB, emitARM64RET,
}

func EmitARM64(in []Instruction) ([]byte, error) {
	b, err := asm.NewBuilder("arm64", len(in)+1)
	if err != nil {
		return nil, err
	}
	for _, inst := range in {
		if !validARM64Op(inst.Op) {
			return nil, ErrUnsupported
		}
		arm64Handlers[inst.Op](b, inst)
	}
	return b.Assemble(), nil
}

func validARM64Op(op Op) bool { return int(op) < len(arm64Handlers) }

func newARM64Prog(b *asm.Builder, as obj.As) *obj.Prog {
	p := b.NewProg()
	p.As = as
	return p
}

func arm64ConstReg(p *obj.Prog, reg int16, value int64) {
	p.From.Type, p.From.Offset = obj.TYPE_CONST, value
	p.To.Type, p.To.Reg = obj.TYPE_REG, arm64Reg(reg)
}

func arm64Reg(index int16) int16 { return arm64.REG_R0 + index }

func emitARM64NOP(b *asm.Builder, _ Instruction) {
	b.AddInstruction(newARM64Prog(b, obj.ANOP))
}

func emitARM64MOV(b *asm.Builder, i Instruction) {
	p := newARM64Prog(b, arm64.AMOVD)
	arm64ConstReg(p, i.Dst, i.Imm)
	b.AddInstruction(p)
}

func emitARM64ADD(b *asm.Builder, i Instruction) {
	p := newARM64Prog(b, arm64.AADD)
	p.From.Type, p.From.Offset = obj.TYPE_CONST, i.Imm
	p.Reg, p.To.Reg = arm64Reg(i.Src), arm64Reg(i.Dst)
	p.To.Type = obj.TYPE_REG
	b.AddInstruction(p)
}

func emitARM64SUB(b *asm.Builder, i Instruction) {
	p := newARM64Prog(b, arm64.ASUB)
	p.From.Type, p.From.Offset = obj.TYPE_CONST, i.Imm
	p.Reg, p.To.Reg = arm64Reg(i.Src), arm64Reg(i.Dst)
	p.To.Type = obj.TYPE_REG
	b.AddInstruction(p)
}

func emitARM64RET(b *asm.Builder, _ Instruction) {
	p := newARM64Prog(b, obj.ARET)
	p.To.Type, p.To.Reg = obj.TYPE_REG, arm64.REGLINK
	b.AddInstruction(p)
}
