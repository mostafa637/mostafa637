// Package fpu contains the first Pure Go boundary for iSH's x87-style
// extended floating-point values. It intentionally does not belong to storage
// or filesystem code.
package fpu

import x80 "github.com/jenska/float"

// Value is an emulator-facing wrapper around jenska/float's IEEE 754
// extended-precision X80 value. Keeping the third-party type behind this
// boundary lets the future instruction emulator replace implementation details
// without leaking them into kernel, fs, or UI packages.
type Value struct {
	raw x80.X80
}

func FromFloat64(v float64) Value {
	return Value{raw: x80.NewFromFloat64(v)}
}

func FromInt64(v int64) Value {
	return Value{raw: x80.Int64ToFloatX80(v)}
}

func FromRaw(v x80.X80) Value {
	return Value{raw: v}
}

func (v Value) Raw() x80.X80 {
	return v.raw
}

func (v Value) ToFloat64() float64 {
	return v.raw.ToFloat64()
}

func (v Value) ToInt64() int64 {
	return v.raw.ToInt64()
}

func (v Value) String() string {
	return v.raw.String()
}

func (v Value) Add(other Value) Value {
	return Value{raw: v.raw.Add(other.raw)}
}

func (v Value) Sub(other Value) Value {
	return Value{raw: v.raw.Sub(other.raw)}
}

func (v Value) Mul(other Value) Value {
	return Value{raw: v.raw.Mul(other.raw)}
}

func (v Value) Div(other Value) Value {
	return Value{raw: v.raw.Div(other.raw)}
}

func (v Value) Rem(other Value) Value {
	return Value{raw: v.raw.Rem(other.raw)}
}

func (v Value) Sqrt() Value {
	return Value{raw: v.raw.Sqrt()}
}

func (v Value) RoundToInt() Value {
	return Value{raw: v.raw.RoundToInt()}
}

func (v Value) Neg() Value {
	return Value{raw: x80.X80Zero.Sub(v.raw)}
}

func (v Value) Abs() Value {
	if v.raw.Lt(x80.X80Zero) {
		return v.Neg()
	}
	return v
}

func (v Value) Eq(other Value) bool {
	return v.raw.Eq(other.raw)
}

func (v Value) Lt(other Value) bool {
	return v.raw.Lt(other.raw)
}

func (v Value) Le(other Value) bool {
	return v.raw.Le(other.raw)
}

func (v Value) IsNaN() bool {
	return v.raw.IsNaN()
}

func (v Value) IsInf() bool {
	return v.raw.IsInf()
}

// Ln is exposed as a building block for the x87 transcendental instruction
// port. Exception and rounding policies remain owned by the emulator layer.
func (v Value) Ln() Value {
	return Value{raw: v.raw.Ln()}
}
