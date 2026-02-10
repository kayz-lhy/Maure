package logparser

import (
	"fmt"
	"testing"

	"maure/pkg/document"
)

type mockParser struct{}

func (m *mockParser) Parse(line string) (*document.Document, error) {
	if line == "" {
		return nil, fmt.Errorf("empty")
	}
	doc := document.NewDocument()
	doc.Add(document.NewTextField("message", line))
	return doc, nil
}

func TestParserRegistryRegisterAndGet(t *testing.T) {
	reg := NewParserRegistry()
	if err := reg.Register("mock", func() LogParser { return &mockParser{} }); err != nil {
		t.Fatalf("register mock parser failed: %v", err)
	}

	p, err := reg.Get("mock")
	if err != nil {
		t.Fatalf("get parser failed: %v", err)
	}
	doc, err := p.Parse("hello")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got := doc.Get("message"); got == nil || got.StringValue() != "hello" {
		t.Fatalf("unexpected parse result: %+v", got)
	}
}

func TestParserRegistryRejectReservedOrInvalid(t *testing.T) {
	reg := NewParserRegistry()
	if err := reg.Register(FormatAuto, func() LogParser { return &mockParser{} }); err == nil {
		t.Fatalf("expected reserved format error")
	}
	if err := reg.Register("", func() LogParser { return &mockParser{} }); err == nil {
		t.Fatalf("expected empty format error")
	}
	if err := reg.Register("x", nil); err == nil {
		t.Fatalf("expected nil constructor error")
	}
}
