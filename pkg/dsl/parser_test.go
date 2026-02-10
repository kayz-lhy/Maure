package dsl

import "testing"

func TestParse_VersionScopeLimitSort(t *testing.T) {
	p := NewParser()
	parsed, err := p.Parse(`@v1 IN index("app"),index("ops") level:error LIMIT 10,20 SORT BY timestamp DESC,level ASC`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.Version != 1 {
		t.Fatalf("expected version 1, got %d", parsed.Version)
	}
	if len(parsed.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(parsed.Scopes))
	}
	if parsed.Limit == nil || parsed.Limit.From != 10 || parsed.Limit.Size != 20 {
		t.Fatalf("unexpected limit: %+v", parsed.Limit)
	}
	if len(parsed.Sort) != 2 || !parsed.Sort[0].Desc || parsed.Sort[1].Desc {
		t.Fatalf("unexpected sort: %+v", parsed.Sort)
	}
}

func TestParse_AdvancedExpressions(t *testing.T) {
	p := NewParser()
	cases := []string{
		`price:[100 TO 200] AND title:iph*`,
		`timestamp:[2026-02-10T09:00:00Z TO 2026-02-10T10:00:00Z]`,
		`name:roam~1 OR message:"db timeout"`,
	}
	for _, input := range cases {
		if _, err := p.Parse(input); err != nil {
			t.Fatalf("parse %q failed: %v", input, err)
		}
	}
}

func TestParse_RejectUnsupported(t *testing.T) {
	p := NewParser()
	bad := []string{
		`title:*phone`,
		`name:roam~2`,
		`price:[100 200]`,
	}
	for _, input := range bad {
		if _, err := p.Parse(input); err == nil {
			t.Fatalf("expected parse error for %q", input)
		}
	}
}

func TestTokenize_ScopeAndSort(t *testing.T) {
	got := tokenize(`@v1 IN index("app"),index("ops") level:error LIMIT 10,20 SORT BY ts DESC`)
	want := []string{"@v1", "IN", `index("app")`, ",", `index("ops")`, "level:error", "LIMIT", "10", ",", "20", "SORT", "BY", "ts", "DESC"}
	if len(got) != len(want) {
		t.Fatalf("token len mismatch\nwant=%v\ngot=%v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token mismatch at %d\nwant=%v\ngot=%v", i, want, got)
		}
	}
}
