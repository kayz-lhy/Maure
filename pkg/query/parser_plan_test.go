package query

import "testing"

func TestQueryParser_ParsePlan(t *testing.T) {
	p := NewQueryParser()
	plan, err := p.ParsePlan(`@v1 IN index("app") price:[100 TO 300] LIMIT 5,10 SORT BY timestamp DESC`)
	if err != nil {
		t.Fatalf("parse plan failed: %v", err)
	}
	if plan == nil || plan.Query == nil {
		t.Fatalf("expected non-nil plan and query")
	}
	if plan.Version != 1 {
		t.Fatalf("expected version 1, got %d", plan.Version)
	}
	if len(plan.Scopes) != 1 || plan.Scopes[0].Kind != "index" || plan.Scopes[0].Value != "app" {
		t.Fatalf("unexpected scopes: %+v", plan.Scopes)
	}
	if plan.Limit == nil || plan.Limit.From != 5 || plan.Limit.Size != 10 {
		t.Fatalf("unexpected limit: %+v", plan.Limit)
	}
	if len(plan.Sort) != 1 || !plan.Sort[0].Desc {
		t.Fatalf("unexpected sort: %+v", plan.Sort)
	}
}
