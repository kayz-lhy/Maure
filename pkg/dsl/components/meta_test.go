package components

import "testing"

func TestNormalizeKeyword(t *testing.T) {
	if NormalizeKeyword("and") != "AND" {
		t.Fatalf("keyword normalize failed")
	}
	if NormalizeKeyword("custom") != "custom" {
		t.Fatalf("non-keyword normalize failed")
	}
	if !IsBooleanBoundary("LIMIT") || !IsBooleanBoundary("OR") || IsBooleanBoundary("TERM") {
		t.Fatalf("boolean boundary mismatch")
	}
}
