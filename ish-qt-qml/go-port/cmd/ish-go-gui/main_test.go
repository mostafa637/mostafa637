package main

import (
	"bytes"
	"testing"
)

func TestControlByteMatchesISH(t *testing.T) {
	tests := []struct {
		name string
		in   rune
		want byte
	}{
		{name: "lowercase c", in: 'c', want: 3},
		{name: "uppercase c", in: 'C', want: 3},
		{name: "space", in: ' ', want: 0},
		{name: "at", in: '@', want: 0},
		{name: "two", in: '2', want: 0},
		{name: "six", in: '6', want: 30},
		{name: "left bracket", in: '[', want: 27},
		{name: "backslash", in: '\\', want: 28},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := controlByte(test.in)
			if !ok || got != test.want {
				t.Fatalf("controlByte(%q) = %#02x, %v; want %#02x, true", test.in, got, ok, test.want)
			}
		})
	}
	if _, ok := controlByte('q'); !ok {
		t.Fatal("alphabetic control key must be accepted")
	}
	if _, ok := controlByte('!'); ok {
		t.Fatal("unsupported control key must be rejected")
	}
}

func TestTerminalInputBytes(t *testing.T) {
	got, next := terminalInputBytes("echo\n", false)
	if !bytes.Equal(got, []byte("echo\r")) || next {
		t.Fatalf("ordinary newline input = %q, next=%v; want carriage return and false", got, next)
	}
	got, next = terminalInputBytes("c", true)
	if !bytes.Equal(got, []byte{3}) || next {
		t.Fatalf("Control-c input = %#v, next=%v; want [3], false", got, next)
	}
	got, next = terminalInputBytes("ab", true)
	if !bytes.Equal(got, []byte("ab")) || next {
		t.Fatalf("multi-rune Control insertion = %q, next=%v; want ordinary text and false", got, next)
	}
	got, next = terminalInputBytes("!", true)
	if got != nil || next {
		t.Fatalf("unsupported one-rune Control insertion = %#v, next=%v; want nil and false", got, next)
	}
	got, next = terminalInputBytes("", true)
	if got != nil || !next {
		t.Fatalf("empty input = %#v, next=%v; want nil, true", got, next)
	}
}

func TestArrowDirectionFromDelta(t *testing.T) {
	tests := []struct {
		name, want string
		dx, dy     float32
	}{
		{name: "below slop", dx: 19, dy: 0, want: "none"},
		{name: "up", dx: 0, dy: -21, want: "up"},
		{name: "down", dx: 0, dy: 21, want: "down"},
		{name: "left", dx: -22, dy: 3, want: "left"},
		{name: "right", dx: 22, dy: -3, want: "right"},
	}
	labels := map[int]string{arrowNone: "none", 0: "up", 1: "down", 2: "left", 3: "right"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := labels[arrowDirectionFromDelta(test.dx, test.dy, 20)]
			if got != test.want {
				t.Fatalf("arrowDirectionFromDelta(%v, %v, 20) = %q; want %q", test.dx, test.dy, got, test.want)
			}
		})
	}
}

func TestMetricsForWidth(t *testing.T) {
	tests := []struct {
		width, button, horizontal, vertical, barHeight, gap float32
	}{
		{width: 320, button: 32, horizontal: 6, vertical: 6, barHeight: 36, gap: 6},
		{width: 429, button: 32, horizontal: 6, vertical: 6, barHeight: 36, gap: 6},
		{width: 430, button: 36, horizontal: 10, vertical: 8, barHeight: 36, gap: 6},
		{width: 599, button: 36, horizontal: 10, vertical: 8, barHeight: 36, gap: 6},
		{width: 600, button: 43, horizontal: 15, vertical: 8, barHeight: 43, gap: 6},
	}
	for _, test := range tests {
		got := metricsForWidth(test.width)
		if got.button != test.button || got.horizontal != test.horizontal || got.vertical != test.vertical || got.barHeight != test.barHeight || got.gap != test.gap {
			t.Errorf("metricsForWidth(%v) = %+v; want button=%v horizontal=%v vertical=%v barHeight=%v gap=%v", test.width, got, test.button, test.horizontal, test.vertical, test.barHeight, test.gap)
		}
	}
	if !metricsForWidth(390).showHideKeyboard {
		t.Fatal("phone layout must retain the hide-keyboard control")
	}
	if metricsForWidth(700).showHideKeyboard {
		t.Fatal("wide iPad layout must omit the hide-keyboard control")
	}
}
