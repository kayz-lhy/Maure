package logparser

import "testing"

func TestJSONParserParse(t *testing.T) {
	parser := &JSONParser{}
	line := `{"timestamp":"2026-02-10T09:00:00Z","level":"error","logger":"api","message":"request failed","code":500}`

	doc, err := parser.Parse(line)
	if err != nil {
		t.Fatalf("parse json log failed: %v", err)
	}

	if got := doc.Get("level"); got == nil || got.StringValue() != "ERROR" {
		t.Fatalf("expected normalized level=ERROR, got %+v", got)
	}
	if got := doc.Get("message"); got == nil || got.StringValue() != "request failed" {
		t.Fatalf("expected message field, got %+v", got)
	}
	if got := doc.Get("code"); got == nil {
		t.Fatalf("expected code field")
	}
}

func TestLogbackParserParse(t *testing.T) {
	parser := &LogbackParser{}
	line := "2026-02-10 09:30:01.123 INFO com.example.OrderService - order created"

	doc, err := parser.Parse(line)
	if err != nil {
		t.Fatalf("parse logback failed: %v", err)
	}

	if got := doc.Get("level"); got == nil || got.StringValue() != "INFO" {
		t.Fatalf("expected normalized level=INFO, got %+v", got)
	}
	if got := doc.Get("logger"); got == nil || got.StringValue() != "com.example.OrderService" {
		t.Fatalf("expected logger field, got %+v", got)
	}
	if got := doc.Get("message"); got == nil || got.StringValue() != "order created" {
		t.Fatalf("expected message field, got %+v", got)
	}
}

func TestDetectFormat(t *testing.T) {
	if got := DetectFormat(`{"k":"v"}`); got != FormatJSON {
		t.Fatalf("expected json, got %s", got)
	}
	if got := DetectFormat("2026-02-10 09:30:01.123 INFO App - ok"); got != FormatLogback {
		t.Fatalf("expected logback, got %s", got)
	}
}

func TestJSONParserInvalidLine(t *testing.T) {
	parser := &JSONParser{}
	if _, err := parser.Parse(`{"level":`); err == nil {
		t.Fatalf("expected json parse error")
	}
}

func TestLogbackParserInvalidLine(t *testing.T) {
	parser := &LogbackParser{}
	if _, err := parser.Parse("not a logback line"); err == nil {
		t.Fatalf("expected logback parse error")
	}
}
