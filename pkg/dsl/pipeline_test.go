package dsl

import "testing"

type dummyCompiler struct{}

func (dummyCompiler) Compile(plan *Plan) (Executable, error) {
	return plan, nil
}

func TestEngine_BuildPlanAndExecutable(t *testing.T) {
	engine := NewEngine(DefaultLexer{}, NewParser(), NoopValidator{}, DefaultPlanner{}, dummyCompiler{})
	plan, err := engine.BuildPlan(`@v1 IN index("app") price:[100 TO 300]`)
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if plan == nil || plan.ExprTree == nil {
		t.Fatalf("expected non-nil plan and expr")
	}
	if plan.Version != 1 {
		t.Fatalf("unexpected version: %d", plan.Version)
	}
	if len(plan.Scopes) != 1 {
		t.Fatalf("unexpected scopes: %+v", plan.Scopes)
	}

	exec, err := engine.BuildExecutable(`name:roam~1`)
	if err != nil {
		t.Fatalf("build executable failed: %v", err)
	}
	if exec == nil {
		t.Fatalf("expected non-nil executable")
	}
}
