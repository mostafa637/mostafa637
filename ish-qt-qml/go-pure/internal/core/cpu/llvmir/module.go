package llvmir

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func Build(p Program) (*ir.Module, error) {
	if err := validateProgram(p); err != nil {
		return nil, err
	}
	module := ir.NewModule()
	function := module.NewFunc(p.functionName(), types.I64, ir.NewParam("input", types.I64))
	entry := function.NewBlock("entry")
	result := emitOps(entry, function.Params[0], p.Ops)
	entry.NewRet(result)
	return module, nil
}

func validateProgram(p Program) error {
	for _, op := range p.Ops {
		if !validOp(op.Kind) {
			return fmt.Errorf("llvmir: unsupported op %d", op.Kind)
		}
	}
	return nil
}

func validOp(kind OpKind) bool {
	return kind <= OpMul
}

func emitOps(block *ir.Block, input value.Value, ops []Op) value.Value {
	result := input
	for _, op := range ops {
		result = emitOp(block, result, op)
	}
	return result
}
