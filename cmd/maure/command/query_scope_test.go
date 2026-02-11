package command

import (
	"testing"

	"maure/pkg/dsl"
	"maure/pkg/query"
)

func TestApplyScopeQuery_NoIndexFieldPassthrough(t *testing.T) {
	base := query.NewTermQuery("iphone")
	got, err := applyScopeQuery(base, []dsl.Scope{{Kind: "index", Value: "app"}}, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != base {
		t.Fatalf("expected passthrough query when index field is absent")
	}
}

func TestApplyScopeQuery_WithIndexFieldBuildsFilter(t *testing.T) {
	base := query.NewTermQuery("iphone")
	got, err := applyScopeQuery(base, []dsl.Scope{{Kind: "index", Value: "app"}}, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil query")
	}
	if _, ok := got.(*query.ConjunctionQuery); !ok {
		t.Fatalf("expected conjunction query, got %T", got)
	}
}

func TestApplyScopeQuery_UnsupportedScope(t *testing.T) {
	_, err := applyScopeQuery(query.NewTermQuery("iphone"), []dsl.Scope{{Kind: "tenant", Value: "t1"}}, true, false)
	if err == nil {
		t.Fatalf("expected unsupported scope error")
	}
}

func TestApplyScopeQuery_ForceInRequiresScope(t *testing.T) {
	_, err := applyScopeQuery(query.NewTermQuery("iphone"), nil, true, true)
	if err == nil {
		t.Fatalf("expected force-in missing scope error")
	}
}

func TestApplyScopeQuery_ForceInRequiresIndexField(t *testing.T) {
	_, err := applyScopeQuery(query.NewTermQuery("iphone"), []dsl.Scope{{Kind: "index", Value: "app"}}, false, true)
	if err == nil {
		t.Fatalf("expected force-in missing index field error")
	}
}
