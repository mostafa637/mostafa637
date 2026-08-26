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

func TestMetricsForWidth(t *testing.T) {
	tests := []struct {
		width, button, horizontal, vertical, barHeight float32
	}{
		{width: 320, button: 32, horizontal: 6, vertical: 2, barHeight: 36},
		{width: 429, button: 32, horizontal: 6, vertical: 2, barHeight: 36},
		{width: 430, button: 36, horizontal: 10, vertical: 0, barHeight: 36},
		{width: 599, button: 36, horizontal: 10, vertical: 0, barHeight: 36},
		{width: 600, button: 43, horizontal: 15, vertical: 0, barHeight: 43},
	}
	for _, test := range tests {
		got := metricsForWidth(test.width)
		if got.button != test.button || got.horizontal != test.horizontal || got.vertical != test.vertical || got.barHeight != test.barHeight {
			t.Errorf("metricsForWidth(%v) = %+v; want button=%v horizontal=%v vertical=%v barHeight=%v", test.width, got, test.button, test.horizontal, test.vertical, test.barHeight)
		}
	}
}
