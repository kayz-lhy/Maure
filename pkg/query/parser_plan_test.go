package query

import "testing"

func TestQueryParser_ParsePlan(t *testing.T) {
	p := NewQueryParser()
	plan, err := p.ParsePlan(`@v1 REQUIRE_IN IN index("app") price:[100 TO 300] LIMIT 5,10 SORT BY timestamp DESC`)
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
	if !plan.RequireIn {
		t.Fatalf("expected RequireIn=true")
	}
	if plan.Limit == nil || plan.Limit.From != 5 || plan.Limit.Size != 10 {
		t.Fatalf("unexpected limit: %+v", plan.Limit)
	}
	if len(plan.Sort) != 1 || !plan.Sort[0].Desc {
		t.Fatalf("unexpected sort: %+v", plan.Sort)
	}
}

func TestQueryParser_Parse_RejectPlanOnlyMetadata(t *testing.T) {
	p := NewQueryParser()
	if _, err := p.Parse(`@v1 IN index("app") price:[100 TO 300] LIMIT 0,20 SORT BY timestamp DESC`); err == nil {
		t.Fatalf("expected Parse to reject plan-only metadata")
	}
}
