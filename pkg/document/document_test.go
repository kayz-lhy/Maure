package document

import (
	"testing"
)

func TestNewTextField(t *testing.T) {
	field := NewTextField("title", "Hello World")

	if field.Name != "title" {
		t.Errorf("expected name 'title', got %q", field.Name)
	}
	if field.Value != "Hello World" {
		t.Errorf("expected value 'Hello World', got %v", field.Value)
	}
	if field.FieldType != FieldTypeText {
		t.Errorf("expected type FieldTypeText, got %v", field.FieldType)
	}
	if !field.Indexed {
		t.Error("expected Indexed to be true")
	}
	if !field.Tokenized {
		t.Error("expected Tokenized to be true")
	}
	if !field.Stored {
		t.Error("expected Stored to be true")
	}
}

func TestNewStringField(t *testing.T) {
	field := NewStringField("status", "active")

	if field.FieldType != FieldTypeString {
		t.Errorf("expected type FieldTypeString, got %v", field.FieldType)
	}
	if field.Tokenized {
		t.Error("expected Tokenized to be false for string field")
	}
}

func TestFieldStringValue(t *testing.T) {
	tests := []struct {
		name     string
		field    *Field
		expected string
	}{
		{
			name:     "string field",
			field:    NewStringField("name", "test"),
			expected: "test",
		},
		{
			name:     "text field",
			field:    NewTextField("content", "hello"),
			expected: "hello",
		},
		{
			name:     "int64 field",
			field:    NewInt64Field("count", 42),
			expected: "42",
		},
		{
			name:     "float64 field",
			field:    NewFloat64Field("price", 3.14),
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.field.StringValue(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewDocument(t *testing.T) {
	doc := NewDocument()

	if len(doc.Fields) != 0 {
		t.Errorf("expected empty fields, got %d", len(doc.Fields))
	}
	if doc.Boost != 1.0 {
		t.Errorf("expected boost 1.0, got %f", doc.Boost)
	}
}

func TestDocumentAdd(t *testing.T) {
	doc := NewDocument()

	doc.Add(NewTextField("title", "Test Title"))
	doc.Add(NewTextField("content", "Test Content"))

	if len(doc.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(doc.Fields))
	}
}

func TestDocumentGet(t *testing.T) {
	doc := NewDocument()
	doc.Add(NewTextField("title", "Test Title"))
	doc.Add(NewStringField("status", "active"))

	t.Run("existing field", func(t *testing.T) {
		field := doc.Get("title")
		if field == nil {
			t.Fatal("expected field, got nil")
		}
		if field.StringValue() != "Test Title" {
			t.Errorf("expected 'Test Title', got %q", field.StringValue())
		}
	})

	t.Run("non-existing field", func(t *testing.T) {
		field := doc.Get("nonexistent")
		if field != nil {
			t.Errorf("expected nil, got %v", field)
		}
	})
}

func TestDocumentGetAll(t *testing.T) {
	doc := NewDocument()
	doc.Add(NewStringField("tag", "go"))
	doc.Add(NewTextField("title", "Test"))
	doc.Add(NewStringField("tag", "programming"))

	tags := doc.GetAll("tag")

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestDocumentString(t *testing.T) {
	doc := NewDocument()
	doc.Add(NewTextField("title", "Hello"))
	doc.Add(NewInt64Field("count", 42))

	s := doc.String()

	expected := "Document{title:Hello, count:42}"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestDocumentBoost(t *testing.T) {
	doc := NewDocument()

	doc.SetBoost(2.0)
	if doc.BoostValue() != 2.0 {
		t.Errorf("expected boost 2.0, got %f", doc.BoostValue())
	}
}

func TestDocumentID(t *testing.T) {
	doc := NewDocument()

	doc.SetID("doc-123")
	if doc.ID() != "doc-123" {
		t.Errorf("expected id 'doc-123', got %q", doc.ID())
	}
}

func TestFieldType(t *testing.T) {
	tests := []struct {
		ft       FieldType
		expected string
	}{
		{FieldTypeText, "text"},
		{FieldTypeString, "string"},
		{FieldTypeInt64, "int64"},
		{FieldTypeFloat64, "float64"},
		{FieldTypeBool, "bool"},
		{FieldTypeDate, "date"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.ft.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewStoredField(t *testing.T) {
	field := NewStoredField("meta", "value")

	if !field.Stored {
		t.Error("expected Stored to be true")
	}
	if field.Indexed {
		t.Error("expected Indexed to be false")
	}
	if field.Tokenized {
		t.Error("expected Tokenized to be false")
	}
}

func TestNumberValue(t *testing.T) {
	doc := NewDocument()
	doc.Add(NewInt64Field("count", 123))

	count := doc.Get("count")
	if count.NumberValue() != 123 {
		t.Errorf("expected 123, got %d", count.NumberValue())
	}
}

func BenchmarkDocumentAdd(b *testing.B) {
	doc := NewDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc.Fields = doc.Fields[:0]
		for j := 0; j < 10; j++ {
			doc.Add(NewTextField("field", "value"))
		}
	}
}
