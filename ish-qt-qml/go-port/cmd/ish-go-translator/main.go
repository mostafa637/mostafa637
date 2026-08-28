package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	translator "github.com/mostafa637/ish-qt-qml/go-port/internal/ir"
)

func main() {
	input := flag.String("hex", "48c7c0010000004883c002c3", "x86-64 bytes as hexadecimal")
	base := flag.Uint64("base", 0x1000, "virtual address of the first byte")
	out := flag.String("o", "", "write an object file using llc instead of stdout")
	llc := flag.String("llc", "llc-18", "LLVM llc executable used for machine-code generation")
	flag.Parse()

	code, err := hex.DecodeString(strings.TrimSpace(*input))
	if err != nil {
		fatal("invalid -hex: %v", err)
	}
	module, err := translator.LowerRAXSubset(code, *base)
	if err != nil {
		fatal("translation failed: %v", err)
	}
	irText := module.String()
	if *out == "" {
		fmt.Print(irText)
		return
	}

	irFile, err := os.CreateTemp("", "ish-go-*.ll")
	if err != nil {
		fatal("create temporary IR: %v", err)
	}
	irPath := irFile.Name()
	defer os.Remove(irPath)
	if _, err := irFile.WriteString(irText); err != nil {
		irFile.Close()
		fatal("write temporary IR: %v", err)
	}
	if err := irFile.Close(); err != nil {
		fatal("close temporary IR: %v", err)
	}
	cmd := exec.Command(*llc, "-filetype=obj", "-mtriple="+runtime.GOARCH+"-unknown-linux-gnu", "-o", *out, irPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("LLVM llc failed: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s for host %s\n", *out, runtime.GOARCH)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
