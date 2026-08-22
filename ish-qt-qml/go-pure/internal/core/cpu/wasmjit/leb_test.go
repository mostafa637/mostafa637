package wasmjit

import (
	"fmt"
	"testing"
)

func TestLEB128(t *testing.T) {
	tests := []struct {
		val int64
		hex string
	}{
		{0, "00"},
		{1, "01"},
		{-1, "7f"},
		{64, "c000"},
		{-64, "40"},
		{4294967295, "ffffffff0f"},
	}

	for _, tc := range tests {
		out := appendSLEB(nil, tc.val)
		h := fmt.Sprintf("%x", out)
		if h != tc.hex {
			t.Errorf("SLEB(%d) = %s, want %s", tc.val, h, tc.hex)
		}
	}
}
