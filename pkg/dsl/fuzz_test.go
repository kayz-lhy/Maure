package dsl

import "testing"

func FuzzParse(f *testing.F) {
	p := NewParser()
	seeds := []string{
		"go",
		"go AND language",
		"(go OR rust) AND NOT java",
		`title:iph* AND price:[100 TO 300]`,
		`name:roam~1`,
		`@v1 IN index("app") level:error LIMIT 10,20 SORT BY timestamp DESC`,
		`"db timeout"`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_, _ = p.Parse(input)
	})
}
