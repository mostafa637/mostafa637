package wasmjit

import (
	"context"
	"github.com/tetratelabs/wazero/api"
)

type SyscallHandler func(number uint64, args [6]uint64) uint64

type GuestBlock struct {
	PC    uint64
	Bytes []byte
	Arch  string
}

type HostBlock struct {
	Code   []byte
	Key    BlockKey
	run    api.Function
	memory api.Memory
	stop   func(context.Context) error
}

type BlockKey struct {
	Hash    [32]byte
	PC      uint64
	Arch    string
	Version uint32
}

func (b *HostBlock) Run(ctx context.Context, regs [16]uint64) (uint64, error) {
	out, err := b.RunRegs(ctx, regs)
	return out[0], err
}

func (b *HostBlock) RunRegs(ctx context.Context, regs [16]uint64) ([16]uint64, error) {
	values, err := b.run.Call(ctx, regArgs(regs)...)
	if err != nil {
		return [16]uint64{}, err
	}
	return regResults(values), nil
}

func regArgs(regs [16]uint64) []uint64 { return regs[:] }

func regResults(values []uint64) [16]uint64 {
	var out [16]uint64
	copy(out[:], values)
	return out
}

func (b *HostBlock) Memory() api.Memory { return b.memory }

func (b *HostBlock) Close(ctx context.Context) error {
	if b.stop == nil {
		return nil
	}
	return b.stop(ctx)
}
