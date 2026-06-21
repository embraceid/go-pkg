package pointer

import "testing"

func TestValReturnsPointerToValue(t *testing.T) {
	p := Val("cakapp")

	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != "cakapp" {
		t.Fatalf("expected pointer value cakapp, got %q", *p)
	}
}

func TestExtractReturnsZeroValueForNilPointer(t *testing.T) {
	got := Extract[string](nil)

	if got != "" {
		t.Fatalf("expected zero string, got %q", got)
	}
}

func TestExtractReturnsDereferencedValue(t *testing.T) {
	value := 42

	if got := Extract(&value); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestEmptyNilReturnsNilForZeroValue(t *testing.T) {
	if got := EmptyNil(0); got != nil {
		t.Fatalf("expected nil for zero value, got %v", *got)
	}
}

func TestEmptyNilReturnsPointerForNonZeroValue(t *testing.T) {
	got := EmptyNil("fish")

	if got == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *got != "fish" {
		t.Fatalf("expected fish, got %q", *got)
	}
}

func TestSliceValReturnsNilForNilSlice(t *testing.T) {
	if got := SliceVal[string](nil); got != nil {
		t.Fatalf("expected nil for nil slice, got %v", *got)
	}
}

func TestSliceValReturnsPointerForEmptySlice(t *testing.T) {
	got := SliceVal([]string{})

	if got == nil {
		t.Fatal("expected non-nil pointer for empty slice")
	}
	if len(*got) != 0 {
		t.Fatalf("expected empty slice, got %v", *got)
	}
}

func TestSliceValReturnsPointerForNonEmptySlice(t *testing.T) {
	got := SliceVal([]string{"a", "b"})

	if got == nil {
		t.Fatal("expected non-nil pointer")
	}
	if (*got)[0] != "a" || (*got)[1] != "b" {
		t.Fatalf("expected [a b], got %v", *got)
	}
}
