package cpu

import (
	"encoding/binary"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/wasmjit"
)

func memoryHandlers(memory *Memory64) (wasmjit.MemoryLoadHandler, wasmjit.MemoryStoreHandler) {
	if memory == nil {
		return nil, nil
	}
	return loadMemory(memory), storeMemory(memory)
}

func loadMemory(memory *Memory64) wasmjit.MemoryLoadHandler {
	return func(address uint64) uint64 {
		var data [8]byte
		if err := memory.Read(Address64(address), data[:]); err != nil {
			return 0
		}
		return binary.LittleEndian.Uint64(data[:])
	}
}

func storeMemory(memory *Memory64) wasmjit.MemoryStoreHandler {
	return func(address, value uint64) {
		var data [8]byte
		binary.LittleEndian.PutUint64(data[:], value)
		_ = memory.Write(Address64(address), data[:])
	}
}
