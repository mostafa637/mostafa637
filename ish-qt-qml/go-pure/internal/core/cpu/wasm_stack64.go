package cpu

func pushCall64(memory *Memory64, state *MachineState64, returnPC uint64) error {
	if memory == nil || state == nil {
		return ErrInvalid64Block
	}
	rsp := state.Regs[RSP]
	if rsp < 8 || !canonicalAddress64(Address64(rsp-8)) {
		return ErrRange
	}
	next := rsp - 8
	if err := memory.WriteUint64(Address64(next), returnPC); err != nil {
		return err
	}
	state.Regs[RSP] = next
	state.CallDepth++
	return nil
}

func popReturn64(memory *Memory64, state *MachineState64, cleanup uint64) (uint64, error) {
	if memory == nil || state == nil {
		return 0, ErrInvalid64Block
	}
	rsp := state.Regs[RSP]
	target, err := memory.ReadUint64(Address64(rsp))
	if err != nil {
		return 0, err
	}
	next := rsp + 8 + cleanup
	if next < rsp || !canonicalAddress64(Address64(next)) || !canonicalAddress64(Address64(target)) {
		return 0, ErrRange
	}
	state.Regs[RSP] = next
	if state.CallDepth > 0 {
		state.CallDepth--
	}
	return target, nil
}
