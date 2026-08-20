package cpu

import "errors"

func wasmFault(state *MachineState64, err error) uint64 {
	if errors.Is(err, ErrUnmapped) || errors.Is(err, ErrProtection) || errors.Is(err, ErrRange) {
		state.TrapNo = Trap64PageFault
		return state.TrapNo
	}
	if errors.Is(err, ErrUnsupported64) {
		state.TrapNo = Trap64InvalidOpcode
		return state.TrapNo
	}
	state.TrapNo = Trap64GeneralFault
	return state.TrapNo
}
