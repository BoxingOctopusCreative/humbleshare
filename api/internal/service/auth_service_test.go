package service

import (
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  Ryan@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "ryan@example.com" {
		t.Fatalf("got %q", got)
	}
	if _, err := normalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHashTokenStable(t *testing.T) {
	a := hashToken("abc")
	b := hashToken("abc")
	if a != b || a == "" {
		t.Fatalf("hash mismatch %q %q", a, b)
	}
	if hashToken("abc") == hashToken("abd") {
		t.Fatal("different inputs should hash differently")
	}
}
