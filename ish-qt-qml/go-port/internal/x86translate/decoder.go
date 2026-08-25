package x86translate

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

// Instruction is one decoded x86-64 instruction with its original address.
type Instruction struct {
	PC   uint64
	Inst x86asm.Inst
	Text string
}

// Decode decodes a contiguous x86-64 byte sequence. It deliberately does not
// translate instructions: decoding and semantic lowering are separate stages.
func Decode(code []byte, base uint64) ([]Instruction, error) {
	var out []Instruction
	for off := 0; off < len(code); {
		inst, err := x86asm.Decode(code[off:], 64)
		if err != nil {
			return nil, fmt.Errorf("decode at 0x%x: %w", base+uint64(off), err)
		}
		if inst.Len <= 0 || off+inst.Len > len(code) {
			return nil, fmt.Errorf("invalid instruction length %d at 0x%x", inst.Len, base+uint64(off))
		}
		out = append(out, Instruction{
			PC:   base + uint64(off),
			Inst: inst,
			Text: x86asm.IntelSyntax(inst, base+uint64(off), nil),
		})
		off += inst.Len
	}
	return out, nil
}
