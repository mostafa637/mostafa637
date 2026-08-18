package fpu

import (
	"math"
	"testing"
)

func TestExtendedArithmetic(t *testing.T) {
	one := FromInt64(1)
	three := FromInt64(3)
	result := one.Div(three).Mul(three)
	if math.Abs(result.ToFloat64()-1) > 1e-15 {
		t.Fatalf("1/3*3 = %.17g, want 1", result.ToFloat64())
	}

	sqrtTwo := FromFloat64(2).Sqrt()
	if math.Abs(sqrtTwo.Mul(sqrtTwo).ToFloat64()-2) > 1e-15 {
		t.Fatalf("sqrt(2)^2 = %.17g, want 2", sqrtTwo.Mul(sqrtTwo).ToFloat64())
	}
}

func TestClassificationAndComparison(t *testing.T) {
	if !FromRaw(FromFloat64(math.Inf(1)).Raw()).IsInf() {
		t.Fatal("positive infinity was not classified as infinity")
	}
	if !FromRaw(FromFloat64(math.NaN()).Raw()).IsNaN() {
		t.Fatal("NaN was not classified as NaN")
	}
	if !FromInt64(-2).Lt(FromInt64(1)) {
		t.Fatal("comparison failed")
	}
	if !FromInt64(-2).Abs().Eq(FromInt64(2)) {
		t.Fatal("absolute value failed")
	}
}
