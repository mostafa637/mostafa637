package ir

import (
	"fmt"
	"strings"

	llvmir "github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	"golang.org/x/arch/x86/x86asm"

	"github.com/mostafa637/ish-qt-qml/go-port/internal/x86translate"
)

// LowerRAXSubset lowers a small, explicit x86-64 subset into LLVM IR.
//
// The first milestone models RAX as an i64 function argument and supports
// NOP, MOV RAX, imm64, ADD/SUB/XOR RAX, imm, and RET. Unsupported instructions
// return an error; no machine-code encoder is implemented here.
func LowerRAXSubset(code []byte, base uint64) (*llvmir.Module, error) {
	decoded, err := x86translate.Decode(code, base)
	if err != nil {
		return nil, err
	}
	module := llvmir.NewModule()
	fn := module.NewFunc("ish_translated_block", types.I64, llvmir.NewParam("rax_in", types.I64))
	block := fn.NewBlock("entry")
	var rax value.Value = fn.Params[0]
	terminated := false

	for _, ins := range decoded {
		name := strings.ToUpper(ins.Inst.Op.String())
		switch name {
		case "NOP", "PAUSE":
			continue
		case "RET", "RETN", "RETF":
			block.NewRet(rax)
			terminated = true
		case "MOV", "MOVABS":
			imm, ok := ins.Inst.Args[1].(x86asm.Imm)
			if !ok || !isRAX(ins.Inst.Args[0]) {
				return nil, fmt.Errorf("0x%x: only MOV RAX, imm64 is supported", ins.PC)
			}
			rax = constant.NewInt(types.I64, int64(imm))
		case "ADD", "SUB", "XOR":
			if !isRAX(ins.Inst.Args[0]) {
				return nil, fmt.Errorf("0x%x: only %s RAX, imm is supported", ins.PC, name)
			}
			imm, ok := ins.Inst.Args[1].(x86asm.Imm)
			if !ok {
				return nil, fmt.Errorf("0x%x: %s source must be an immediate", ins.PC, name)
			}
			rhs := constant.NewInt(types.I64, int64(imm))
			switch name {
			case "ADD":
				rax = block.NewAdd(rax, rhs)
			case "SUB":
				rax = block.NewSub(rax, rhs)
			case "XOR":
				rax = block.NewXor(rax, rhs)
			}
		default:
			return nil, fmt.Errorf("0x%x: unsupported x86-64 instruction %s", ins.PC, ins.Text)
		}
		if terminated {
			break
		}
	}
	if !terminated {
		block.NewRet(rax)
	}
	return module, nil
}

// Register helpers keep the supported subset intentionally small.
func isRAX(arg interface{}) bool {
	return fmt.Sprint(arg) == "RAX" || fmt.Sprint(arg) == "EAX" || fmt.Sprint(arg) == "AX" || fmt.Sprint(arg) == "AL"
}
