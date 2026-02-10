package dsl

import (
	"reflect"
	"strings"
	"testing"
)

func TestExprNodeMethodsAndKeyword(t *testing.T) {
	TermExpr{}.exprNode()
	PhraseExpr{}.exprNode()
	RangeExpr{}.exprNode()
	WildcardExpr{}.exprNode()
	FuzzyExpr{}.exprNode()
	AndExpr{}.exprNode()
	OrExpr{}.exprNode()
	NotExpr{}.exprNode()
	FilterNotExpr{}.exprNode()

	if got := normalizeKeyword("and"); got != "AND" {
		t.Fatalf("normalize keyword failed: %s", got)
	}
	if got := normalizeKeyword("custom"); got != "custom" {
		t.Fatalf("normalize non-keyword failed: %s", got)
	}
}

func TestStateHelpers(t *testing.T) {
	s := &state{}
	if s.hasMore() {
		t.Fatalf("expected no more tokens")
	}
	if got := s.peek(); got != "" {
		t.Fatalf("expected empty peek, got %q", got)
	}
	if got := s.consume(); got != "" {
		t.Fatalf("expected empty consume, got %q", got)
	}

	s = &state{tokens: []string{"A"}}
	if !s.hasMore() {
		t.Fatalf("expected has more")
	}
	if got := s.peekUpper(); got != "A" {
		t.Fatalf("unexpected peek upper: %s", got)
	}
	if got := s.consume(); got != "A" {
		t.Fatalf("unexpected consume: %s", got)
	}
}

func TestParseVersionDirect(t *testing.T) {
	s := &state{tokens: []string{"hello"}}
	if v, ok, err := s.parseVersion(); err != nil || ok || v != 0 {
		t.Fatalf("unexpected parseVersion result: v=%d ok=%v err=%v", v, ok, err)
	}

	s = &state{tokens: []string{"@v0"}}
	if _, _, err := s.parseVersion(); err == nil {
		t.Fatalf("expected invalid version error")
	}

	s = &state{tokens: []string{"@v2"}}
	if v, ok, err := s.parseVersion(); err != nil || !ok || v != 2 {
		t.Fatalf("unexpected parseVersion: v=%d ok=%v err=%v", v, ok, err)
	}
}

func TestParseScopeAndScopeItem(t *testing.T) {
	s := &state{tokens: []string{"hello"}}
	scopes, err := s.parseScope()
	if err != nil || scopes != nil {
		t.Fatalf("unexpected parseScope result: scopes=%v err=%v", scopes, err)
	}

	s = &state{tokens: []string{"IN"}}
	if _, err := s.parseScope(); err == nil {
		t.Fatalf("expected missing scope item error")
	}

	s = &state{tokens: []string{"IN", `index("app")`, ",", `index("ops")`}}
	scopes, err = s.parseScope()
	if err != nil {
		t.Fatalf("parse scope failed: %v", err)
	}
	if len(scopes) != 2 || scopes[0].Kind != "index" || scopes[1].Value != "ops" {
		t.Fatalf("unexpected scopes: %+v", scopes)
	}

	bad := []string{"index", "index(app)", `index("")`}
	for _, item := range bad {
		if _, err := parseScopeItem(item); err == nil {
			t.Fatalf("expected parseScopeItem error for %q", item)
		}
	}
	if got, err := parseScopeItem(`InDeX("x")`); err != nil || got.Kind != "index" || got.Value != "x" {
		t.Fatalf("unexpected parseScopeItem success: %+v err=%v", got, err)
	}
}

func TestParseLimitDirect(t *testing.T) {
	casesErr := []struct {
		name   string
		tokens []string
	}{
		{"missing", []string{"LIMIT"}},
		{"comma-no-size", []string{"LIMIT", "10", ","}},
		{"too-many", []string{"LIMIT", "1,2,3"}},
		{"size-zero", []string{"LIMIT", "0"}},
		{"size-nan", []string{"LIMIT", "abc"}},
		{"from-negative", []string{"LIMIT", "-1,10"}},
		{"from-nan", []string{"LIMIT", "a,10"}},
		{"size2-zero", []string{"LIMIT", "1,0"}},
	}
	for _, tc := range casesErr {
		t.Run(tc.name, func(t *testing.T) {
			s := &state{tokens: tc.tokens}
			if _, err := s.parseLimit(); err == nil {
				t.Fatalf("expected error")
			}
		})
	}

	s := &state{tokens: []string{"LIMIT", "25"}}
	limit, err := s.parseLimit()
	if err != nil || limit.From != 0 || limit.Size != 25 {
		t.Fatalf("unexpected limit: %+v err=%v", limit, err)
	}

	s = &state{tokens: []string{"LIMIT", "10", ",", "20"}}
	limit, err = s.parseLimit()
	if err != nil || limit.From != 10 || limit.Size != 20 {
		t.Fatalf("unexpected limit comma tokens: %+v err=%v", limit, err)
	}
}

func TestParseSortDirect(t *testing.T) {
	s := &state{tokens: []string{"SORT", "X"}}
	if _, err := s.parseSort(); err == nil {
		t.Fatalf("expected missing BY error")
	}

	s = &state{tokens: []string{"SORT", "BY"}}
	if _, err := s.parseSort(); err == nil {
		t.Fatalf("expected missing field error")
	}

	s = &state{tokens: []string{"SORT", "BY", "ts", "DESC", ",", "level", "ASC"}}
	items, err := s.parseSort()
	if err != nil {
		t.Fatalf("parse sort failed: %v", err)
	}
	if len(items) != 2 || !items[0].Desc || items[1].Desc {
		t.Fatalf("unexpected sort items: %+v", items)
	}
}

func TestParsePrimaryAndBooleanFlow(t *testing.T) {
	p := NewParser()
	if parsed, err := p.Parse(""); err != nil || parsed == nil {
		t.Fatalf("empty parse failed: parsed=%v err=%v", parsed, err)
	}
	good := []string{
		`(a OR b) AND c`,
		`a NOT b`,
		`NOT a`,
		`a b`,
		`"just phrase"`,
		`field:"x y"`,
	}
	for _, q := range good {
		if _, err := p.Parse(q); err != nil {
			t.Fatalf("parse %q failed: %v", q, err)
		}
	}

	if _, err := p.Parse(`(a OR b`); err == nil {
		t.Fatalf("expected missing closing parenthesis error")
	}

	if _, err := p.Parse(`a )`); err == nil {
		t.Fatalf("expected unexpected token error")
	}

	errCases := []string{
		`@vx a`,
		`IN`,
		`IN bad`,
		`a LIMIT x`,
		`a SORT X`,
		`a OR (`,
		`a AND (`,
		`a NOT (`,
		`a (`,
		`f:*abc`,
	}
	for _, q := range errCases {
		if _, err := p.Parse(q); err == nil {
			t.Fatalf("expected parse error for %q", q)
		}
	}

	st := &state{}
	if node, err := st.parsePrimary(); err != nil || node != nil {
		t.Fatalf("expected nil primary for empty state, got node=%#v err=%v", node, err)
	}

	st = &state{tokens: []string{"(", "a", "OR", "("}}
	if _, err := st.parsePrimary(); err == nil {
		t.Fatalf("expected parsePrimary nested error")
	}

	st = &state{tokens: []string{"NOT", "("}}
	if _, err := st.parseNot(); err == nil {
		t.Fatalf("expected parseNot nested error")
	}
}

func TestParseFieldExpressionDirect(t *testing.T) {
	if _, ok, err := parseFieldExpression("no-colon"); err != nil || ok {
		t.Fatalf("expected no field expression")
	}
	if _, _, err := parseFieldExpression(":x"); err == nil {
		t.Fatalf("expected invalid field expression error")
	}
	if _, _, err := parseFieldExpression("f:"); err == nil {
		t.Fatalf("expected invalid field expression error")
	}

	errCases := []string{
		"f:[1 TO 2}",
		"f:[1 2]",
		"f:[ TO 2]",
		"f:[1 TO ]",
		"f:[x TO y]",
		"f:a?b",
		"f:*",
		"f:*abc",
		"f:a*b",
		"f:~1",
		"f:abc~2",
	}
	for _, c := range errCases {
		if _, _, err := parseFieldExpression(c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}

	node, ok, err := parseFieldExpression("f:{1 TO 2}")
	if err != nil || !ok {
		t.Fatalf("expected open range success: %v", err)
	}
	rg, ok := node.(RangeExpr)
	if !ok || rg.Inclusive {
		t.Fatalf("expected exclusive range, got %#v", node)
	}

	node, ok, err = parseFieldExpression("f:[1 TO 2]")
	if err != nil || !ok {
		t.Fatalf("expected range success: %v", err)
	}
	if _, ok := node.(RangeExpr); !ok {
		t.Fatalf("expected range node")
	}

	node, ok, err = parseFieldExpression("f:ab*")
	if err != nil || !ok {
		t.Fatalf("expected wildcard success: %v", err)
	}
	if _, ok := node.(WildcardExpr); !ok {
		t.Fatalf("expected wildcard node")
	}

	node, ok, err = parseFieldExpression("f:ab~1")
	if err != nil || !ok {
		t.Fatalf("expected fuzzy success: %v", err)
	}
	if _, ok := node.(FuzzyExpr); !ok {
		t.Fatalf("expected fuzzy node")
	}

	node, ok, err = parseFieldExpression("f:AbC")
	if err != nil || !ok {
		t.Fatalf("expected term success: %v", err)
	}
	te, ok := node.(TermExpr)
	if !ok || te.Value != "abc" {
		t.Fatalf("expected lowercased term, got %#v", node)
	}
}

func TestInferRangeKindAndTimeParse(t *testing.T) {
	if kind, err := inferRangeKind("1", "2"); err != nil || kind != RangeValueNumber {
		t.Fatalf("expected numeric kind, got kind=%v err=%v", kind, err)
	}
	if kind, err := inferRangeKind("2026-01-01", "2026-01-02"); err != nil || kind != RangeValueTime {
		t.Fatalf("expected time kind, got kind=%v err=%v", kind, err)
	}
	if _, err := inferRangeKind("x", "y"); err == nil {
		t.Fatalf("expected infer error")
	}

	timeInputs := []string{"2026-01-01T00:00:00Z", "2026-01-01 00:00:00", "2026-01-01"}
	for _, in := range timeInputs {
		if _, err := parseRangeTime(in); err != nil {
			t.Fatalf("parseRangeTime failed for %q: %v", in, err)
		}
	}
	if _, err := parseRangeTime("bad-time"); err == nil {
		t.Fatalf("expected parseRangeTime error")
	}
}

func TestSplitRangeBounds(t *testing.T) {
	l, u, ok := splitRangeBounds("100 TO 200")
	if !ok || l != "100" || u != "200" {
		t.Fatalf("unexpected split result: %v %q %q", ok, l, u)
	}

	l, u, ok = splitRangeBounds(" 2026-01-01T00:00:00Z   to   2026-02-01T00:00:00Z ")
	if !ok || l != "2026-01-01T00:00:00Z" || u != "2026-02-01T00:00:00Z" {
		t.Fatalf("unexpected case-insensitive split: %v %q %q", ok, l, u)
	}

	bad := []string{"", "100 200", "TO 200", "100 TO"}
	for _, in := range bad {
		if _, _, ok := splitRangeBounds(in); ok {
			t.Fatalf("expected split fail for %q", in)
		}
	}
}

func TestTokenizeBranches(t *testing.T) {
	if got := tokenize(""); got != nil {
		t.Fatalf("expected nil for empty tokenize, got %v", got)
	}

	cases := map[string][]string{
		`IN index("app")`:     {"IN", `index("app")`},
		`field:"hello world"`: {`field:"hello world"`},
		`a [`:                 {"a", "["},
		`price:[1 TO 2]`:      {`price:[1 TO 2]`},
		`price:{1 TO 2}`:      {`price:{1 TO 2}`},
		`"hello"`:             {`"hello"`},
		`(a OR b),c`:          {"(", "a", "OR", "b", ")", ",", "c"},
		`@v1 IN index("app"),index("ops") LIMIT 10,20`: {"@v1", "IN", `index("app")`, ",", `index("ops")`, "LIMIT", "10", ",", "20"},
	}
	for in, want := range cases {
		got := tokenize(in)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("tokenize mismatch\ninput=%q\nwant=%v\ngot=%v", in, want, got)
		}
	}

	withSpaces := tokenize(" a\t\n b ")
	if strings.Join(withSpaces, ",") != "a,b" {
		t.Fatalf("unexpected spaced tokenize: %v", withSpaces)
	}
}
