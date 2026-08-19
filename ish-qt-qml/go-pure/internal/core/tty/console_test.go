package tty

import (
	"bytes"
	"testing"

	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestConsoleInstall(t *testing.T) {
	input := bytes.NewBufferString("input")
	output := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	table := corefd.New()
	if err := NewConsole(input, output, errOut).Install(table); err != nil {
		t.Fatal(err)
	}
	in, err := table.Get(0)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := in.Read(buf); err != nil || string(buf) != "input" {
		t.Fatalf("stdin read = %q, %v", buf, err)
	}
	out, err := table.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("out")); err != nil {
		t.Fatal(err)
	}
	if output.String() != "out" {
		t.Fatalf("stdout = %q", output.String())
	}
	errFile, err := table.Get(2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := errFile.Write([]byte("err")); err != nil {
		t.Fatal(err)
	}
	if errOut.String() != "err" {
		t.Fatalf("stderr = %q", errOut.String())
	}
}
