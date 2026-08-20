package machinecode

import (
	asm "github.com/twitchyliquid64/golang-asm"
	"github.com/twitchyliquid64/golang-asm/obj/x86"
)

func emitX86Syscall(b *asm.Builder, _ Instruction) {
	p := b.NewProg()
	p.As = x86.ASYSCALL
	b.AddInstruction(p)
}
