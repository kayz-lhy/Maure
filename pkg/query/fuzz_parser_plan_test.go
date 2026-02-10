package query

import "testing"

func FuzzParsePlan(f *testing.F) {
	p := NewQueryParser()
	seeds := []string{
		"go",
		"go AND programming",
		"price:[100 TO 300] AND title:iph*",
		"name:roam~1 OR title:iph*",
		"price:[100 TO 500] NOT title:iph*",
		`@v1 IN index("app") price:[100 TO 300] LIMIT 10,20 SORT BY timestamp DESC`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_, _ = p.ParsePlan(input)
	})
}
