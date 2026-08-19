package cpu

import (
	"errors"
	"testing"
)

func TestMemoryReadWriteAcrossPages(t *testing.T) {
	mem := NewMemory()
	if err := mem.Map(10, 2, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	payload := []byte("page-boundary")
	addr := Address(10*PageSize + PageSize - 4)
	if err := mem.Write(addr, payload); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(payload))
	if err := mem.Read(addr, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("read = %q, want %q", got, payload)
	}
	if err := mem.SetFlags(10, 2, PRead); err != nil {
		t.Fatal(err)
	}
	if err := mem.Write(addr, []byte("x")); !errors.Is(err, ErrProtection) {
		t.Fatalf("write protected = %v, want ErrProtection", err)
	}
	if err := mem.Unmap(10, 2); err != nil {
		t.Fatal(err)
	}
	if err := mem.Unmap(10, 1); !errors.Is(err, ErrUnmapped) {
		t.Fatalf("second unmap = %v, want ErrUnmapped", err)
	}
}

func TestMemoryGrowsDownAndCopyOnWrite(t *testing.T) {
	mem := NewMemory()
	if err := mem.Map(20, 1, PRead|PWrite|PGrowDown); err != nil {
		t.Fatal(err)
	}
	grown := make([]byte, 1)
	if err := mem.Read(Address(19*PageSize+3), grown); err != nil {
		t.Fatal(err)
	}
	if grown[0] != 0 {
		t.Fatalf("new grows-down byte = %d, want zero", grown[0])
	}
	if _, ok := mem.Page(19); !ok {
		t.Fatal("grows-down page was not mapped")
	}

	src := NewMemory()
	dst := NewMemory()
	backing := make([]byte, PageSize)
	copy(backing, []byte("source"))
	if err := src.MapBytes(30, 1, backing, 0, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := src.CopyOnWrite(src, dst, 30, 1); err != nil {
		t.Fatal(err)
	}
	if err := dst.Write(30*PageSize, []byte("target")); err != nil {
		t.Fatal(err)
	}
	srcGot := make([]byte, 6)
	dstGot := make([]byte, 6)
	if err := src.Read(30*PageSize, srcGot); err != nil {
		t.Fatal(err)
	}
	if err := dst.Read(30*PageSize, dstGot); err != nil {
		t.Fatal(err)
	}
	if string(srcGot) != "source" || string(dstGot) != "target" {
		t.Fatalf("COW source=%q destination=%q", srcGot, dstGot)
	}
}

func TestMemoryFindHole(t *testing.T) {
	mem := NewMemory()
	hole := mem.FindHole(3)
	if hole == BadPage {
		t.Fatal("FindHole returned BadPage for empty memory")
	}
	if !mem.IsHole(hole, 3) {
		t.Fatalf("selected range %d is not a hole", hole)
	}
	if err := mem.Map(hole+1, 1, PRead); err != nil {
		t.Fatal(err)
	}
	if mem.IsHole(hole, 3) {
		t.Fatal("range containing mapping reported as hole")
	}
}

func TestMachineStateLazyFlags(t *testing.T) {
	state := NewMachineState(NewMemory())
	state.SetLazyArithmetic(1, 1, 0, false, false, true)
	if !state.Flag(FlagZF) || !state.Flag(FlagPF) || state.Flag(FlagSF) || state.Flag(FlagCF) {
		t.Fatalf("lazy flags: zf=%v pf=%v sf=%v cf=%v", state.Flag(FlagZF), state.Flag(FlagPF), state.Flag(FlagSF), state.Flag(FlagCF))
	}
	state.CollapseFlags()
	if state.EFlags&FlagZF == 0 || state.EFlags&FlagPF == 0 {
		t.Fatalf("collapsed eflags = %#x", state.EFlags)
	}
	state.Set(EAX, 42)
	if state.Get(EAX) != 42 || state.EAXValue() != 42 {
		t.Fatal("EAX accessors disagree")
	}
}
