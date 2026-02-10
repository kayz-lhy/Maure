package logparser

import (
	"encoding/json"
	"testing"
)

func TestBuildDocumentFromFieldsCommonKeysNormalized(t *testing.T) {
	fields := map[string]interface{}{
		"message":   "request timeout",
		"level":     "warn",
		"logger":    "gateway",
		"timestamp": "2026-02-10 09:30:01.123",
		"code":      json.Number("504"),
	}

	doc := BuildDocumentFromFields(`{"message":"request timeout"}`, fields)

	if got := doc.Get("message"); got == nil || got.StringValue() != "request timeout" {
		t.Fatalf("unexpected message field: %+v", got)
	}
	if got := doc.Get("level"); got == nil || got.StringValue() != "WARN" {
		t.Fatalf("unexpected level field: %+v", got)
	}
	if got := doc.Get("logger"); got == nil || got.StringValue() != "gateway" {
		t.Fatalf("unexpected logger field: %+v", got)
	}
	if got := doc.Get("timestamp"); got == nil || got.StringValue() == "2026-02-10 09:30:01.123" {
		t.Fatalf("timestamp should be normalized to RFC3339, got: %+v", got)
	}
	if got := doc.Get("code"); got == nil {
		t.Fatalf("expected extra field code")
	}
}

func TestNormalizeTimestampFallback(t *testing.T) {
	input := "not-a-time"
	if got := normalizeTimestamp(input); got != input {
		t.Fatalf("expected fallback original value, got %s", got)
	}
}

func TestGetParserForLine(t *testing.T) {
	if _, err := GetParserForLine(FormatAuto, `{"a":1}`); err != nil {
		t.Fatalf("auto parser for json failed: %v", err)
	}
	if _, err := GetParserForLine(FormatAuto, "2026-02-10 09:30:01.123 INFO App - ok"); err != nil {
		t.Fatalf("auto parser for logback failed: %v", err)
	}
	if _, err := GetParserForLine("unknown", "hello"); err == nil {
		t.Fatalf("expected unsupported format error")
	}
}
