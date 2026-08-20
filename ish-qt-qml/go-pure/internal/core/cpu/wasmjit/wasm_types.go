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
	out, err := b.run.Call(ctx, regs[0], regs[1], regs[2], regs[3], regs[4], regs[5], regs[6], regs[7], regs[8], regs[9], regs[10], regs[11], regs[12], regs[13], regs[14], regs[15])
	if err != nil {
		return 0, err
	}
	return out[0], nil
}

func (b *HostBlock) Memory() api.Memory { return b.memory }

func (b *HostBlock) Close(ctx context.Context) error {
	if b.stop == nil {
		return nil
	}
	return b.stop(ctx)
}
