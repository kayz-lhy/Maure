package query

import (
	"encoding/json"
	"maure/pkg/index"
	"os"
	"sort"
	"testing"
)

type dslCase struct {
	Name        string  `json:"name"`
	Query       string  `json:"query"`
	ExpectDocID []int64 `json:"expect_doc_ids"`
	ExpectError bool    `json:"expect_error"`
}

func TestDSLRegressionCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/dsl_cases.json")
	if err != nil {
		t.Fatalf("read testdata failed: %v", err)
	}
	var cases []dslCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("unmarshal testdata failed: %v", err)
	}
	if len(cases) == 0 {
		t.Fatalf("empty dsl testdata")
	}

	idx := createAdvancedIndex(t)
	defer idx.Close()
	parser := NewQueryParser()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			plan, err := parser.ParsePlan(tc.Query)
			if tc.ExpectError {
				if err == nil {
					t.Fatalf("expected parse error for query: %s", tc.Query)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse plan failed: %v", err)
			}
			if plan == nil || plan.Query == nil {
				t.Fatalf("expected non-nil query plan")
			}

			results, err := idx.Search(plan.Query, 100)
			if err != nil {
				t.Fatalf("search failed: %v", err)
			}
			got := collectDocIDs(results)
			want := append([]int64(nil), tc.ExpectDocID...)
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
			if !equalInt64Slice(got, want) {
				t.Fatalf("doc set mismatch\nquery=%s\nwant=%v\ngot=%v", tc.Query, want, got)
			}
		})
	}
}

func TestDSLBooleanCombinations_NoParseRegression(t *testing.T) {
	atoms := []string{
		`price:[100 TO 500]`,
		`title:iph*`,
		`name:roam~1`,
	}
	ops := []string{"AND", "OR"}

	idx := createAdvancedIndex(t)
	defer idx.Close()
	parser := NewQueryParser()

	for _, left := range atoms {
		for _, right := range atoms {
			for _, op := range ops {
				queries := []string{
					left + " " + op + " " + right,
					"(" + left + " " + op + " " + right + ")",
					left + " NOT " + right,
					"NOT (" + left + " " + op + " " + right + ")",
				}
				for _, q := range queries {
					plan, err := parser.ParsePlan(q)
					if err != nil {
						t.Fatalf("parse failed for %q: %v", q, err)
					}
					if plan == nil || plan.Query == nil {
						t.Fatalf("nil plan/query for %q", q)
					}
					if _, err := idx.Search(plan.Query, 100); err != nil {
						t.Fatalf("search failed for %q: %v", q, err)
					}
				}
			}
		}
	}
}

func collectDocIDs(results []index.ScoreDoc) []int64 {
	docs := make([]int64, 0, len(results))
	for _, r := range results {
		docs = append(docs, r.DocID)
	}
	return docs
}

func equalInt64Slice(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
