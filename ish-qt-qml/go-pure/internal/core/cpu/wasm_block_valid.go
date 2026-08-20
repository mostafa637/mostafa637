package cpu

func (b *WasmBlock64) Valid(memory *Memory64) bool {
	if b == nil || memory == nil {
		return false
	}
	for page, generation := range b.Generations {
		current, mapped := memory.PageGeneration(page)
		if !mapped || current != generation {
			return false
		}
	}
	return true
}
