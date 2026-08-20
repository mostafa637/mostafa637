package syscall

import (
	"encoding/binary"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func writeUtimensatTimespec64(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, atimeSec int64, atimeNsec int64, mtimeSec int64, mtimeNsec int64) {
	t.Helper()
	buffer := make([]byte, utimensatTimespec64Size)
	binary.LittleEndian.PutUint64(buffer[0:8], uint64(atimeSec))
	binary.LittleEndian.PutUint64(buffer[8:16], uint64(atimeNsec))
	binary.LittleEndian.PutUint64(buffer[16:24], uint64(mtimeSec))
	binary.LittleEndian.PutUint64(buffer[24:32], uint64(mtimeNsec))
	if err := memory.Write(address, buffer); err != nil {
		t.Fatal(err)
	}
}

func TestUtimensat64ExplicitTimes(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/hello", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	const pathAddress corecpu.Address64 = 0x10000
	const timesAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, pathAddress, "/hello")
	writeUtimensatTimespec64(t, memory, timesAddress, 1700000000, 123456789, 1800000000, 987654321)
	set64Syscall(state, Sys64Utimensat, atFDCWD64, uint64(pathAddress), uint64(timesAddress), 0)
	if resume, dispatchErr := dispatcher.Dispatch(state); dispatchErr != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("utimensat explicit: resume=%v err=%v rax=%d", resume, dispatchErr, int64(state.Get(corecpu.RAX)))
	}
	atime, mtime, err := fake.Times("/hello", false)
	if err != nil {
		t.Fatal(err)
	}
	if !atime.Equal(time.Unix(1700000000, 123456789)) || !mtime.Equal(time.Unix(1800000000, 987654321)) {
		t.Fatalf("timestamps = (%v, %v), want explicit values", atime, mtime)
	}
}

func TestUtimensat64NowOmitAndNofollow(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/target", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if err := fake.Symlink("/target", "/link", 0, 0); err != nil {
		t.Fatal(err)
	}
	const pathAddress corecpu.Address64 = 0x10000
	const timesAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, pathAddress, "/link")
	beforeTargetAtime, beforeTargetMtime, err := fake.Times("/target", false)
	if err != nil {
		t.Fatal(err)
	}
	writeUtimensatTimespec64(t, memory, timesAddress, utimeOmit64, 0, utimeNow64, 0)
	set64Syscall(state, Sys64Utimensat, atFDCWD64, uint64(pathAddress), uint64(timesAddress), atSymlinkNoFollow64)
	if resume, dispatchErr := dispatcher.Dispatch(state); dispatchErr != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("utimensat nofollow: resume=%v err=%v rax=%d", resume, dispatchErr, int64(state.Get(corecpu.RAX)))
	}
	afterTargetAtime, afterTargetMtime, err := fake.Times("/target", false)
	if err != nil {
		t.Fatal(err)
	}
	if !afterTargetAtime.Equal(beforeTargetAtime) || !afterTargetMtime.Equal(beforeTargetMtime) {
		t.Fatalf("target timestamps changed through nofollow: before=(%v,%v) after=(%v,%v)", beforeTargetAtime, beforeTargetMtime, afterTargetAtime, afterTargetMtime)
	}
	linkAtime, linkMtime, err := fake.Times("/link", true)
	if err != nil {
		t.Fatal(err)
	}
	if linkAtime.Equal(beforeTargetAtime) || linkMtime.Equal(beforeTargetMtime) {
		t.Fatalf("link timestamps were not updated: (%v,%v)", linkAtime, linkMtime)
	}
}

func TestUtimensat64Validation(t *testing.T) {
	_, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	const pathAddress corecpu.Address64 = 0x10000
	const timesAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, pathAddress, "/missing")
	writeUtimensatTimespec64(t, memory, timesAddress, 1, 1000000000, 1, 0)
	set64Syscall(state, Sys64Utimensat, atFDCWD64, uint64(pathAddress), uint64(timesAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	if got := int64(state.Get(corecpu.RAX)); got != int64(EINVAL) {
		t.Fatalf("invalid nanoseconds rax=%d, want %d", got, EINVAL)
	}
	set64Syscall(state, Sys64Utimensat, atFDCWD64, uint64(pathAddress), uint64(timesAddress), 1)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	if got := int64(state.Get(corecpu.RAX)); got != int64(EINVAL) {
		t.Fatalf("invalid flags rax=%d, want %d", got, EINVAL)
	}
	set64Syscall(state, Sys64Utimensat, atFDCWD64, uint64(pathAddress), 0x900000, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	if got := int64(state.Get(corecpu.RAX)); got != int64(EFAULT) {
		t.Fatalf("invalid times pointer rax=%d, want %d", got, EFAULT)
	}
}
