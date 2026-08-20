package llvmir

import (
	"strings"
	"testing"
)

func TestBuildProgram(t *testing.T) {
	module, err := Build(Program{Name: "block", Ops: []Op{{Kind: OpAdd, Value: 2}, {Kind: OpMul, Value: 3}}})
	if err != nil {
		t.Fatal(err)
	}
	text := module.String()
	for _, want := range []string{"define i64 @block", "add i64", "mul i64", "ret i64"} {
		if !strings.Contains(text, want) {
			t.Fatalf("IR missing %q:\n%s", want, text)
		}
	}
}

func TestRejectsUnknownOperation(t *testing.T) {
	_, err := Build(Program{Ops: []Op{{Kind: OpKind(99)}}})
	if err == nil {
		t.Fatal("expected invalid operation error")
	}
}
