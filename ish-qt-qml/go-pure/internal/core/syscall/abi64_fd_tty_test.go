package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestDispatcher64FcntlAndIoctl(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	if err := ctx.InstallFile(3, &corefd.File{}); err != nil {
		t.Fatal(err)
	}

	if got := dispatch64Runtime(t, dispatcher, state, Sys64Fcntl, 3, fcntlGetFD, 0); got != 0 {
		t.Fatalf("F_GETFD = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Fcntl, 3, fcntlSetFD, fdCloexec); got != 0 {
		t.Fatalf("F_SETFD = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Fcntl, 3, fcntlGetFD, 0); got != int64(fdCloexec) {
		t.Fatalf("F_GETFD after set = %d", got)
	}
	duplicate := dispatch64Runtime(t, dispatcher, state, Sys64Fcntl, 3, fcntlDupFD, 0)
	if duplicate < 4 {
		t.Fatalf("F_DUPFD = %d", duplicate)
	}
	cloexecDuplicate := dispatch64Runtime(t, dispatcher, state, Sys64Fcntl, 3, fcntlDupFDClO, uint64(duplicate+1))
	if cloexecDuplicate <= duplicate {
		t.Fatalf("F_DUPFD_CLOEXEC = %d", cloexecDuplicate)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Fcntl, uint64(cloexecDuplicate), fcntlGetFD, 0); got != int64(fdCloexec) {
		t.Fatalf("F_GETFD duplicate = %d", got)
	}

	winsizeAddress := area + 0x100
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Ioctl, 3, tiocgwinsz64, uint64(winsizeAddress)); got != 0 {
		t.Fatalf("TIOCGWINSZ = %d", got)
	}
	var winsize [8]byte
	if err := ctx.Memory.Read(winsizeAddress, winsize[:]); err != nil {
		t.Fatal(err)
	}
	if rows, cols := binary.LittleEndian.Uint16(winsize[0:2]), binary.LittleEndian.Uint16(winsize[2:4]); rows != 24 || cols != 80 {
		t.Fatalf("default winsize = %dx%d", cols, rows)
	}
	binary.LittleEndian.PutUint16(winsize[0:2], 40)
	binary.LittleEndian.PutUint16(winsize[2:4], 100)
	if err := ctx.Memory.Write(winsizeAddress, winsize[:]); err != nil {
		t.Fatal(err)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Ioctl, 3, tiocswinsz64, uint64(winsizeAddress)); got != 0 {
		t.Fatalf("TIOCSWINSZ = %d", got)
	}
	if ctx.WinRows != 40 || ctx.WinCols != 100 {
		t.Fatalf("stored winsize = %dx%d", ctx.WinCols, ctx.WinRows)
	}

	termiosAddress := area + 0x200
	var termios [termiosSize64]byte
	for index := range termios {
		termios[index] = byte(index + 1)
	}
	if err := ctx.Memory.Write(termiosAddress, termios[:]); err != nil {
		t.Fatal(err)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Ioctl, 3, tcsets64, uint64(termiosAddress)); got != 0 {
		t.Fatalf("TCSETS = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Ioctl, 3, tcgets64, uint64(area+0x280)); got != 0 {
		t.Fatalf("TCGETS = %d", got)
	}
	var returned [termiosSize64]byte
	if err := ctx.Memory.Read(area+0x280, returned[:]); err != nil {
		t.Fatal(err)
	}
	if returned != termios {
		t.Fatalf("termios round trip = %#v, want %#v", returned, termios)
	}

	if got := dispatch64Runtime(t, dispatcher, state, Sys64Ioctl, 3, tiocgwinsz64, uint64(area)+3*uint64(corecpu.Page64Size)); got != int64(EFAULT) {
		t.Fatalf("bad ioctl address = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Ioctl, 99, tiocgwinsz64, uint64(winsizeAddress)); got != int64(EBADF) {
		t.Fatalf("bad ioctl fd = %d", got)
	}
}
