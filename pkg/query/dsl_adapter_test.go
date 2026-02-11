package query

import (
	"maure/pkg/dsl"
	"testing"
)

func TestDSLAdapter_Compile(t *testing.T) {
	adapter := NewDSLAdapter()
	plan := &dsl.Plan{
		Version: 1,
		ExprTree: dsl.AndExpr{
			Left:  dsl.TermExpr{Field: "title", Value: "iphone"},
			Right: dsl.RangeExpr{Field: "price", Lower: "100", Upper: "300", Kind: dsl.RangeValueNumber, Inclusive: true},
		},
		Scopes: []dsl.Scope{{Kind: "index", Value: "app"}},
		Limit:  &dsl.LimitClause{From: 0, Size: 20},
		Sort:   []dsl.SortClause{{Field: "timestamp", Desc: true}},
	}
	exec, err := adapter.Compile(plan)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	qp, ok := exec.(*QueryPlan)
	if !ok {
		t.Fatalf("expected QueryPlan executable")
	}
	if qp.Query == nil || qp.Version != 1 || len(qp.Scopes) != 1 || qp.Limit == nil || len(qp.Sort) != 1 {
		t.Fatalf("unexpected query plan: %+v", qp)
	}
}

func TestDSLAdapter_NilPlan(t *testing.T) {
	adapter := NewDSLAdapter()
	exec, err := adapter.Compile(nil)
	if err != nil {
		t.Fatalf("compile nil plan failed: %v", err)
	}
	qp, ok := exec.(*QueryPlan)
	if !ok || qp == nil {
		t.Fatalf("expected query plan for nil input")
	}
}
