package cpu

import "testing"

func TestMemory64MapReadWriteAndGeneration(t *testing.T) {
	memory := NewMemory64()
	start := Address64(0x7fff0000)
	if err := memory.Map(start, 2*Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	before, ok := memory.PageGeneration(Page64(start >> Page64Bits))
	if !ok {
		t.Fatal("mapped page is missing")
	}
	value := []byte("x86-64 guest")
	address := start + Address64(Page64Size) - 4
	if err := memory.Write(address, value); err != nil {
		t.Fatal(err)
	}
	var got = make([]byte, len(value))
	if err := memory.Read(address, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(value) {
		t.Fatalf("read %q, want %q", got, value)
	}
	after, ok := memory.PageGeneration(Page64(start >> Page64Bits))
	if !ok || after <= before {
		t.Fatalf("write did not advance page generation: before=%d after=%d", before, after)
	}
}

func TestMemory64ProtectionAndCanonicalRange(t *testing.T) {
	memory := NewMemory64()
	address := Address64(0x400000)
	if err := memory.Map(address, Page64Size, PRead); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(address, []byte{1}); err != ErrProtection {
		t.Fatalf("write error = %v, want %v", err, ErrProtection)
	}
	if err := memory.Read(Address64(0x0000800000000000), []byte{0}); err != ErrRange {
		t.Fatalf("non-canonical read error = %v, want %v", err, ErrRange)
	}
	if err := memory.Map(Address64(0xffff800000400000), Page64Size, PRead|PWrite); err != nil {
		t.Fatalf("high canonical map: %v", err)
	}
}

func TestMachineState64LazyFlagsAndRegisters(t *testing.T) {
	state := NewMachineState64(NewMemory64())
	state.Set(R15, 0xfeedbeefcafebabe)
	if state.Get(R15) != 0xfeedbeefcafebabe {
		t.Fatal("R15 round trip failed")
	}
	state.SetLazyArithmetic(1, 1, 0, false, false, true)
	if !state.Flag(Flag64ZF) || !state.Flag(Flag64PF) || state.Flag(Flag64SF) || state.Flag(Flag64CF) {
		t.Fatalf("lazy flags: zf=%v pf=%v sf=%v cf=%v", state.Flag(Flag64ZF), state.Flag(Flag64PF), state.Flag(Flag64SF), state.Flag(Flag64CF))
	}
	state.CollapseFlags()
	if state.RFLAGS&Flag64ZF == 0 {
		t.Fatal("ZF was not collapsed")
	}
}
