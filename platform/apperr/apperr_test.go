package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew_CodeErrorAndUnwrap(t *testing.T) {
	e := New(404)
	if got := e.Code(); got != 404 {
		t.Fatalf("Code() = %d, want 404", got)
	}
	if got := e.Error(); got != "404" {
		t.Fatalf("Error() = %q, want %q", got, "404")
	}
	if e.Unwrap() != nil {
		t.Fatalf("Unwrap() = %v, want nil", e.Unwrap())
	}
}

func TestWrap_CodeErrorAndCause(t *testing.T) {
	cause := errors.New("boom")
	e := Wrap(500, cause)
	if got := e.Code(); got != 500 {
		t.Fatalf("Code() = %d, want 500", got)
	}
	if got := e.Error(); got != "500: boom" {
		t.Fatalf("Error() = %q, want %q", got, "500: boom")
	}
	if e.Unwrap() != cause {
		t.Fatalf("Unwrap() = %v, want cause", e.Unwrap())
	}
	if !errors.Is(e, cause) {
		t.Fatalf("errors.Is(e, cause) = false, want true")
	}
}

func TestIs_MatchesByCode(t *testing.T) {
	if !errors.Is(New(1), New(1)) {
		t.Fatalf("errors.Is(New(1), New(1)) = false, want true")
	}
	if errors.Is(New(1), New(2)) {
		t.Fatalf("errors.Is(New(1), New(2)) = true, want false")
	}
	// target that is not an *Error → Is returns false
	if New(1).Is(errors.New("plain")) {
		t.Fatalf("Is(non-apperr) = true, want false")
	}
	// code is matched even through a wrapped chain
	wrapped := fmt.Errorf("context: %w", Wrap(7, errors.New("c")))
	if !errors.Is(wrapped, New(7)) {
		t.Fatalf("errors.Is(wrapped, New(7)) = false, want true")
	}
}

func TestCodeOf(t *testing.T) {
	if code, ok := CodeOf(Wrap(42, errors.New("x"))); !ok || code != 42 {
		t.Fatalf("CodeOf = (%d, %v), want (42, true)", code, ok)
	}
	if code, ok := CodeOf(errors.New("plain")); ok || code != 0 {
		t.Fatalf("CodeOf(plain) = (%d, %v), want (0, false)", code, ok)
	}
	if code, ok := CodeOf(fmt.Errorf("ctx: %w", New(9))); !ok || code != 9 {
		t.Fatalf("CodeOf(wrapped) = (%d, %v), want (9, true)", code, ok)
	}
}

func TestHasCode(t *testing.T) {
	if !HasCode(New(3), 3) {
		t.Fatalf("HasCode(New(3), 3) = false, want true")
	}
	if HasCode(New(3), 4) {
		t.Fatalf("HasCode(New(3), 4) = true, want false")
	}
	if HasCode(errors.New("plain"), 3) {
		t.Fatalf("HasCode(plain, 3) = true, want false")
	}
}
