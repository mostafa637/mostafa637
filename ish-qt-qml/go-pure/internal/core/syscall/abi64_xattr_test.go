package syscall

import (
	"os"
	"strings"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

func dispatchXattr64(t *testing.T, dispatcher *Dispatcher64, state *corecpu.MachineState64, number Number64, args ...uint64) int64 {
	t.Helper()
	set64Syscall(state, number, args...)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume {
		t.Fatalf("syscall %#x: resume=%v err=%v", number, resume, err)
	}
	return int64(state.Get(corecpu.RAX))
}

func TestXattr64PathAndFDVariants(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/hello", 0o644, 1000, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("hello")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	const (
		pathAddress   corecpu.Address64 = 0x10000
		nameAddress   corecpu.Address64 = 0x10100
		name2Address  corecpu.Address64 = 0x10200
		valueAddress  corecpu.Address64 = 0x10300
		value2Address corecpu.Address64 = 0x10400
		outputAddress corecpu.Address64 = 0x10500
		listAddress   corecpu.Address64 = 0x10600
	)
	writeABI64CString(t, memory, pathAddress, "/hello")
	writeABI64CString(t, memory, nameAddress, "user.test")
	writeABI64CString(t, memory, name2Address, "user.fd")
	if err := memory.Write(valueAddress, []byte("value")); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(value2Address, []byte("fd-value")); err != nil {
		t.Fatal(err)
	}

	if got := dispatchXattr64(t, dispatcher, state, Sys64Setxattr, uint64(pathAddress), uint64(nameAddress), uint64(valueAddress), 5, 1); got != 0 {
		t.Fatalf("setxattr = %d, want 0", got)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Setxattr, uint64(pathAddress), uint64(nameAddress), uint64(valueAddress), 5, 1); got != int64(EEXIST) {
		t.Fatalf("duplicate setxattr = %d, want %d", got, EEXIST)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Getxattr, uint64(pathAddress), uint64(nameAddress), 0, 0); got != 5 {
		t.Fatalf("getxattr size = %d, want 5", got)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Getxattr, uint64(pathAddress), uint64(nameAddress), uint64(outputAddress), 4); got != int64(ERANGE) {
		t.Fatalf("short getxattr = %d, want %d", got, ERANGE)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Getxattr, uint64(pathAddress), uint64(nameAddress), uint64(outputAddress), 5); got != 5 {
		t.Fatalf("getxattr = %d, want 5", got)
	}
	var output [5]byte
	if err := memory.Read(outputAddress, output[:]); err != nil {
		t.Fatal(err)
	}
	if string(output[:]) != "value" {
		t.Fatalf("getxattr output = %q", output[:])
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Listxattr, uint64(pathAddress), 0, 0); got != int64(len("user.test\x00")) {
		t.Fatalf("listxattr size = %d", got)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Listxattr, uint64(pathAddress), uint64(listAddress), uint64(len("user.test\x00"))); got != int64(len("user.test\x00")) {
		t.Fatalf("listxattr = %d", got)
	}
	var listed [32]byte
	if err := memory.Read(listAddress, listed[:]); err != nil {
		t.Fatal(err)
	}
	if string(listed[:len("user.test\x00")]) != "user.test\x00" {
		t.Fatalf("listxattr output = %q", listed[:len("user.test\x00")])
	}

	fdFile, err := fake.OpenFile("/hello", os.O_RDWR, 0, corefsStatRegular64())
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(42, &corefd.File{Reader: fdFile, Writer: fdFile, Closer: fdFile, Seeker: fdFile, Path: "/hello"}); err != nil {
		t.Fatal(err)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Fsetxattr, 42, uint64(name2Address), uint64(value2Address), 8, 0); got != 0 {
		t.Fatalf("fsetxattr = %d", got)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Fgetxattr, 42, uint64(name2Address), uint64(outputAddress), 8); got != 8 {
		t.Fatalf("fgetxattr = %d", got)
	}
	var fdOutput [8]byte
	if err := memory.Read(outputAddress, fdOutput[:]); err != nil {
		t.Fatal(err)
	}
	if string(fdOutput[:]) != "fd-value" {
		t.Fatalf("fgetxattr output = %q", fdOutput[:])
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Flistxattr, 42, uint64(listAddress), 64); got <= 0 {
		t.Fatalf("flistxattr = %d", got)
	}
	var fdListed [64]byte
	if err := memory.Read(listAddress, fdListed[:]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fdListed[:]), "user.fd\x00") {
		t.Fatalf("flistxattr output = %q", fdListed[:])
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Removexattr, uint64(pathAddress), uint64(nameAddress)); got != 0 {
		t.Fatalf("removexattr = %d", got)
	}
	if got := dispatchXattr64(t, dispatcher, state, Sys64Getxattr, uint64(pathAddress), uint64(nameAddress), 0, 0); got != int64(ENODATA) {
		t.Fatalf("missing getxattr = %d, want %d", got, ENODATA)
	}
}

func corefsStatRegular64() corefs.IshStat {
	return corefs.IshStat{Mode: corefs.ModeRegular | 0o644, UID: 0, GID: 0}
}
