package env

import (
	"os"
	"testing"
)

func TestString(t *testing.T) {
	os.Unsetenv("FOO")

	if got := String("FOO", "bar"); got != "bar" {
		t.Fatalf("expected fallback 'bar', got %q", got)
	}

	os.Setenv("FOO", "baz")

	if got := String("FOO", "bar"); got != "baz" {
		t.Fatalf("expected env value 'baz', got %q", got)
	}
}

func TestMustString(t *testing.T) {
	os.Unsetenv("FOO")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when env var missing and no fallback")
		}
	}()

	MustString("FOO")
}

func TestMustStringFallback(t *testing.T) {
	os.Unsetenv("FOO")

	if got := MustString("FOO", "qux"); got != "qux" {
		t.Fatalf("expected fallback 'qux', got %q", got)
	}
}
