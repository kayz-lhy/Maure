package query

import "testing"

func TestExtractTermsFromTermQuery(t *testing.T) {
	q := NewTermQuery("Go")
	terms := ExtractTerms(q)
	if len(terms) != 1 || terms[0] != "go" {
		t.Fatalf("unexpected terms: %v", terms)
	}
}

func TestExtractTermsFromPhraseQuery(t *testing.T) {
	q := NewPhraseQuery("go", "language")
	terms := ExtractTerms(q)
	if len(terms) < 3 {
		t.Fatalf("unexpected terms: %v", terms)
	}
	if terms[0] != "go language" {
		t.Fatalf("expected phrase first, got %v", terms)
	}
}

func TestExtractTermsFromBooleanQuery(t *testing.T) {
	q := NewBooleanQuery()
	q.Add(NewTermQuery("go"), OccurMust, 1)
	q.Add(NewTermQuery("python"), OccurShould, 1)
	q.Add(NewTermQuery("java"), OccurMustNot, 1)

	terms := ExtractTerms(q)
	if len(terms) != 3 {
		t.Fatalf("unexpected terms: %v", terms)
	}
}

func TestExtractTermsDeduplicated(t *testing.T) {
	q := NewDisjunctionQuery(NewTermQuery("go"), NewTermQuery("go"))
	terms := ExtractTerms(q)
	if len(terms) != 1 || terms[0] != "go" {
		t.Fatalf("unexpected dedup terms: %v", terms)
	}
}
