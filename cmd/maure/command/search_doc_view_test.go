package command

import (
	"testing"

	"maure/pkg/document"
)

func TestParseFieldsParam(t *testing.T) {
	fields, err := parseFieldsParam("message, level ,timestamp,message")
	if err != nil {
		t.Fatalf("parse fields failed: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 deduplicated fields, got %d", len(fields))
	}
	if fields[0] != "message" || fields[1] != "level" || fields[2] != "timestamp" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}

func TestParseFieldsParamInvalid(t *testing.T) {
	_, err := parseFieldsParam("message;drop")
	if err == nil {
		t.Fatalf("expected invalid field error")
	}

	_, err = parseFieldsParam("message,,level")
	if err == nil {
		t.Fatalf("expected empty field error")
	}
}

func TestBuildDocView(t *testing.T) {
	doc := document.NewDocument()
	doc.SetID("doc-1")
	doc.Add(document.NewTextField("message", "request failed"))
	doc.Add(document.NewStringField("level", "error"))

	view := buildDocView(doc, true, []string{"message", "level"})
	if view == nil {
		t.Fatalf("expected doc view")
	}
	if view.Summary == "" {
		t.Fatalf("expected summary")
	}
	if len(view.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(view.Fields))
	}
}
