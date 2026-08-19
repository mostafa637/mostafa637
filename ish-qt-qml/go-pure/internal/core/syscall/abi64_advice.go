package syscall

const (
	fadviseNormal64     uint64 = 0
	fadviseRandom64     uint64 = 1
	fadviseSequential64 uint64 = 2
	fadviseWillNeed64   uint64 = 3
	fadviseDontNeed64   uint64 = 4
	fadviseNoReuse64    uint64 = 5
)

// fadvise64 implements the x86-64 fadvise64 ABI as a validated advisory
// operation. The guest filesystem has no host page-cache contract to forward,
// so valid advice is accepted after descriptor and signed-range validation.
func fadvise64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil {
		return int64(ENOSYS)
	}
	if _, err := ctx.GetFile(args[0]); err != nil {
		return int64(EBADF)
	}
	if args[1] > uint64(^uint64(0)>>1) || args[2] > uint64(^uint64(0)>>1) {
		return int64(EINVAL)
	}
	if args[3] > fadviseNoReuse64 {
		return int64(EINVAL)
	}
	return 0
}
