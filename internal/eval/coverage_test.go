package eval

import (
	"context"
	"testing"
)

func TestAbsFloat_Negative(t *testing.T) {
	if got := absFloat(-5.0); got != 5.0 {
		t.Errorf("absFloat(-5.0) = %f, want 5.0", got)
	}
}

func TestAbsFloat_Zero(t *testing.T) {
	if got := absFloat(0); got != 0 {
		t.Errorf("absFloat(0) = %f, want 0", got)
	}
}

func TestAbsFloat_Positive(t *testing.T) {
	if got := absFloat(3.14); got != 3.14 {
		t.Errorf("absFloat(3.14) = %f, want 3.14", got)
	}
}

func TestStringSlicesEqual_BothNil(t *testing.T) {
	if !stringSlicesEqual(nil, nil) {
		t.Error("stringSlicesEqual(nil, nil) = false, want true")
	}
}

func TestStringSlicesEqual_BothEmpty(t *testing.T) {
	if !stringSlicesEqual([]string{}, []string{}) {
		t.Error("stringSlicesEqual([], []) = false, want true")
	}
}

func TestStringSlicesEqual_DifferentLengths(t *testing.T) {
	if stringSlicesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Error("stringSlicesEqual([a], [a,b]) = true, want false")
	}
}

func TestStringSlicesEqual_DifferentContent(t *testing.T) {
	if stringSlicesEqual([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("stringSlicesEqual([a,b], [a,c]) = true, want false")
	}
}

func TestStringSlicesEqual_Equal(t *testing.T) {
	if !stringSlicesEqual([]string{"a", "b", "c"}, []string{"a", "b", "c"}) {
		t.Error("stringSlicesEqual([a,b,c], [a,b,c]) = false, want true")
	}
}

func TestStringSlicesEqual_NilVsEmpty(t *testing.T) {
	if !stringSlicesEqual(nil, []string{}) {
		t.Error("stringSlicesEqual(nil, []) = false, want true")
	}
}

func TestStringSlicesEqual_EmptyVsNil(t *testing.T) {
	if !stringSlicesEqual([]string{}, nil) {
		t.Error("stringSlicesEqual([], nil) = false, want true")
	}
}

func TestRandStr_Length(t *testing.T) {
	for _, n := range []int{0, 1, 6, 10, 32} {
		got := randStr(n)
		if len(got) != n {
			t.Errorf("randStr(%d) length = %d, want %d", n, len(got), n)
		}
	}
}

func TestRandStr_AlphaNumeric(t *testing.T) {
	s := randStr(100)
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Errorf("randStr() contains non-alphanumeric char: %c", c)
		}
	}
}

func TestSaveResult_NilPool(t *testing.T) {
	e := &DecisionEvaluator{pool: nil}
	if err := e.saveResult(context.Background(), nil); err != nil {
		t.Errorf("saveResult with nil pool = %v, want nil", err)
	}
}
