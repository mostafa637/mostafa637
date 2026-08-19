package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestABI64CopyFileRangeRegularFiles(t *testing.T) {
	fake, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	src, err := fake.Create("/src", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := fake.Create("/dst", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Write([]byte("abcdefghij")); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(3, &corefd.File{Reader: src, Writer: src, Closer: src, Seeker: src, Path: "/src"}); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(4, &corefd.File{Reader: dst, Writer: dst, Closer: dst, Seeker: dst, Path: "/dst"}); err != nil {
		t.Fatal(err)
	}
	const (
		inOffsetAddress  corecpu.Address64 = 0x10200
		outOffsetAddress corecpu.Address64 = 0x10208
	)

	set64Syscall(state, Sys64CopyFileRange, 3, 0, 4, 0, 5, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 5 {
		t.Fatalf("copy_file_range basic: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	basic := make([]byte, 5)
	if _, err := dst.ReadAt(basic, 0); err != nil {
		t.Fatal(err)
	}
	if string(basic) != "abcde" {
		t.Fatalf("basic destination=%q, want abcde", basic)
	}

	if _, err := src.Seek(8, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.Seek(9, 0); err != nil {
		t.Fatal(err)
	}
	if err := writeSigned64(ctx, uint64(inOffsetAddress), 2); err != nil {
		t.Fatal(err)
	}
	if err := writeSigned64(ctx, uint64(outOffsetAddress), 1); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64CopyFileRange, 3, uint64(inOffsetAddress), 4, uint64(outOffsetAddress), 4, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 4 {
		t.Fatalf("copy_file_range offsets: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	var updatedIn, updatedOut int64
	if updatedIn, err = readSigned64(ctx, uint64(inOffsetAddress)); err != nil {
		t.Fatal(err)
	}
	if updatedOut, err = readSigned64(ctx, uint64(outOffsetAddress)); err != nil {
		t.Fatal(err)
	}
	if updatedIn != 6 || updatedOut != 5 {
		t.Fatalf("updated offsets=%d/%d, want 6/5", updatedIn, updatedOut)
	}
	if position, err := src.Seek(0, 1); err != nil || position != 8 {
		t.Fatalf("source position=%d err=%v, want 8", position, err)
	}
	if position, err := dst.Seek(0, 1); err != nil || position != 9 {
		t.Fatalf("destination position=%d err=%v, want 9", position, err)
	}
	placed := make([]byte, 4)
	if _, err := dst.ReadAt(placed, 1); err != nil {
		t.Fatal(err)
	}
	if string(placed) != "cdef" {
		t.Fatalf("offset destination=%q, want cdef", placed)
	}

	if err := writeSigned64(ctx, uint64(inOffsetAddress), 100); err != nil {
		t.Fatal(err)
	}
	if err := writeSigned64(ctx, uint64(outOffsetAddress), 0); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64CopyFileRange, 3, uint64(inOffsetAddress), 4, uint64(outOffsetAddress), 4, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("copy_file_range EOF: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
}

func TestABI64CopyFileRangeValidation(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	src, err := fake.Create("/src", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := fake.Create("/dst", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(3, &corefd.File{Reader: src, Writer: src, Closer: src, Seeker: src, Path: "/src"}); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(4, &corefd.File{Reader: dst, Writer: dst, Closer: dst, Seeker: dst, Path: "/dst"}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64CopyFileRange, 3, 0, 4, 0, 1, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("copy_file_range flags: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	set64Syscall(state, Sys64CopyFileRange, 99, 0, 4, 0, 1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("copy_file_range invalid fd: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	set64Syscall(state, Sys64CopyFileRange, 3, 0, 4, 0, uint64(maxTransfer64+1), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("copy_file_range count: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64CopyFileRange, 3, 0x200000, 4, 0, 1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("copy_file_range invalid offset pointer: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64CopyFileRange, 3, 0, 3, 0, 1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("copy_file_range same fd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	_ = memory
}
