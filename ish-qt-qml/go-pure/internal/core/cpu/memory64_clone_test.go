package cpu

import "testing"

func TestMemory64CloneIsIndependent(t *testing.T) {
	parent := NewMemory64()
	const address Address64 = 0x4000
	if err := parent.MapBytes(address, []byte("parent"), PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	child := parent.Clone()
	if child == nil {
		t.Fatal("Clone returned nil")
	}
	if err := child.Write(address, []byte("child!")); err != nil {
		t.Fatal(err)
	}
	parentData := make([]byte, 6)
	childData := make([]byte, 6)
	if err := parent.Read(address, parentData); err != nil {
		t.Fatal(err)
	}
	if err := child.Read(address, childData); err != nil {
		t.Fatal(err)
	}
	if string(parentData) != "parent" || string(childData) != "child!" {
		t.Fatalf("parent=%q child=%q", parentData, childData)
	}
}

func TestMachineState64CloneUsesChildMemory(t *testing.T) {
	parentMemory := NewMemory64()
	childMemory := parentMemory.Clone()
	parent := NewMachineState64(parentMemory)
	parent.RIP = 0x1234
	parent.Set(RAX, 0xfeed)
	child := parent.Clone(childMemory)
	if child == nil || child.Memory != childMemory || child.RIP != parent.RIP || child.Get(RAX) != parent.Get(RAX) {
		t.Fatalf("unexpected cloned state: %#v", child)
	}
	child.RIP = 0x5678
	child.Set(RAX, 0xbeef)
	if parent.RIP != 0x1234 || parent.Get(RAX) != 0xfeed {
		t.Fatalf("parent state changed: rip=%#x rax=%#x", parent.RIP, parent.Get(RAX))
	}
}
