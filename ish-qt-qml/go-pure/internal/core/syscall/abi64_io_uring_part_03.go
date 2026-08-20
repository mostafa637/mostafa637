package syscall

func ioUringRingLengths64(sqEntries, cqEntries uint32) (uint64, uint64, uint64, bool) {
	sqBytes := uint64(ioUringRingHeader64) + uint64(sqEntries)*4
	cqBytes := uint64(ioUringCQEOffset64) + uint64(cqEntries)*ioUringCQESize64
	sqLength, okSQ := ioUringAlignPage64(sqBytes)
	cqLength, okCQ := ioUringAlignPage64(cqBytes)
	sqesLength, okSQE := ioUringAlignPage64(uint64(sqEntries) * ioUringSQESize64)
	return sqLength, cqLength, sqesLength, okSQ && okCQ && okSQE
}
