package analyzer

import (
	"testing"
)

func TestStandardTokenizer(t *testing.T) {
	tokenizer := NewStandardTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "hello, world! how are you?",
			expected: []string{"hello", "world", "how", "are", "you"},
		},
		{
			name:     "numbers",
			input:    "test 123 abc",
			expected: []string{"test", "123", "abc"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only punctuation",
			input:    ",!?:;",
			expected: nil,
		},
		{
			name:     "unicode",
			input:    "hello 中文 world",
			expected: []string{"hello", "中文", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := tokenizer.Tokenize("content", tt.input)
			tokens := collectTokens(t, stream)

			if len(tokens) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			for i, token := range tokens {
				if token.Text != tt.expected[i] {
					t.Errorf("token %d: expected %q, got %q", i, tt.expected[i], token.Text)
				}
			}
		})
	}
}

func TestStandardAnalyzer(t *testing.T) {
	analyzer := NewStandardAnalyzer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic text",
			input:    "The quick brown fox jumps over the lazy dog",
			expected: []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"},
		},
		{
			name:     "lowercase",
			input:    "Hello WORLD",
			expected: []string{"hello", "world"},
		},
		{
			name:     "stop words removed",
			input:    "this is a test document",
			expected: []string{"test", "document"},
		},
		{
			name:     "short words filtered",
			input:    "I am a boy",
			expected: []string{"boy"},
		},
		{
			name:     "empty after filter",
			input:    "the an and or",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := analyzer.Analyze("content", tt.input)
			tokens := collectTokens(t, stream)

			if len(tokens) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			for i, token := range tokens {
				if token.Text != tt.expected[i] {
					t.Errorf("token %d: expected %q, got %q", i, tt.expected[i], token.Text)
				}
			}
		})
	}
}

func TestTokenPositions(t *testing.T) {
	analyzer := NewStandardAnalyzer()
	stream := analyzer.Analyze("content", "hello world test")

	tokens := collectTokens(t, stream)

	expected := []int{0, 1, 2}
	for i, token := range tokens {
		if token.Position != expected[i] {
			t.Errorf("token %d position: expected %d, got %d", i, expected[i], token.Position)
		}
	}
}

func TestTokenOffsets(t *testing.T) {
	tokenizer := NewStandardTokenizer()
	stream := tokenizer.Tokenize("content", "hello world")

	tokens := collectTokens(t, stream)

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	// "hello" at positions 0-5
	if tokens[0].Start != 0 || tokens[0].End != 5 {
		t.Errorf("token 0 offset: expected 0-5, got %d-%d", tokens[0].Start, tokens[0].End)
	}

	// "world" at positions 6-11
	if tokens[1].Start != 6 || tokens[1].End != 11 {
		t.Errorf("token 1 offset: expected 6-11, got %d-%d", tokens[1].Start, tokens[1].End)
	}
}

func TestTokenFieldName(t *testing.T) {
	analyzer := NewStandardAnalyzer()
	stream := analyzer.Analyze("title", "hello world")

	tokens := collectTokens(t, stream)

	for _, token := range tokens {
		if token.FieldName != "title" {
			t.Errorf("expected field name 'title', got %q", token.FieldName)
		}
	}
}

// collectTokens 收集 TokenStream 中的所有 Token。
func collectTokens(t *testing.T, stream TokenStream) []*Token {
	var tokens []*Token
	for stream.Next() {
		tokens = append(tokens, stream.Current())
	}
	stream.Close()
	return tokens
}

func BenchmarkStandardAnalyzer(b *testing.B) {
	analyzer := NewStandardAnalyzer()
	text := "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream := analyzer.Analyze("content", text)
		for stream.Next() {
			_ = stream.Current()
		}
		stream.Close()
	}
}
