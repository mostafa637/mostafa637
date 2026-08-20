package llvmir

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func emitOp(block *ir.Block, current value.Value, op Op) value.Value {
	switch op.Kind {
	case OpConst:
		return constant.NewInt(types.I64, int64(op.Value))
	case OpAdd:
		return block.NewAdd(current, constant.NewInt(types.I64, int64(op.Value)))
	case OpSub:
		return block.NewSub(current, constant.NewInt(types.I64, int64(op.Value)))
	case OpMul:
		return block.NewMul(current, constant.NewInt(types.I64, int64(op.Value)))
	default:
		return current
	}
}
