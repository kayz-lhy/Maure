package query

// ExtractTerms 从查询树中提取用于高亮的关键词，保持出现顺序并去重。
func ExtractTerms(q Query) []string {
	if q == nil {
		return nil
	}
	seen := make(map[string]struct{})
	terms := make([]string, 0, 8)

	var walk func(node Query)
	walk = func(node Query) {
		if node == nil {
			return
		}
		switch n := node.(type) {
		case *TermQuery:
			appendUnique(&terms, seen, n.Term)
		case *PhraseQuery:
			if len(n.terms) > 0 {
				phrase := joinTermsWithSpace(n.terms)
				appendUnique(&terms, seen, phrase)
				for _, term := range n.terms {
					appendUnique(&terms, seen, term)
				}
			}
		case *BooleanQuery:
			for _, clause := range n.clauses {
				walk(clause.query)
			}
		case *ConjunctionQuery:
			for _, sub := range n.queries {
				walk(sub)
			}
		case *DisjunctionQuery:
			for _, sub := range n.queries {
				walk(sub)
			}
		case *MustQuery:
			walk(n.query)
		case *ShouldQuery:
			for _, sub := range n.queries {
				walk(sub)
			}
		case *notQuery:
			walk(n.subQuery)
		case *WildcardQuery:
			appendUnique(&terms, seen, n.Prefix)
		case *FuzzyQuery:
			appendUnique(&terms, seen, n.Term)
		}
	}
	walk(q)
	return terms
}

func appendUnique(terms *[]string, seen map[string]struct{}, term string) {
	if term == "" {
		return
	}
	if _, ok := seen[term]; ok {
		return
	}
	seen[term] = struct{}{}
	*terms = append(*terms, term)
}

func joinTermsWithSpace(terms []string) string {
	if len(terms) == 0 {
		return ""
	}
	result := terms[0]
	for i := 1; i < len(terms); i++ {
		result += " " + terms[i]
	}
	return result
}
