package command

import (
	"maure/pkg/document"
	"maure/pkg/highlight"
	"strings"
	"unicode"
)

// HighlightRange 表示字段中的高亮位置。
type HighlightRange struct {
	Field    string `json:"field"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Fragment string `json:"fragment"`
}

// SearchDocView 是 /search 可选返回的文档视图。
type SearchDocView struct {
	Summary string                 `json:"summary,omitempty"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// SearchHit 是带高亮信息的搜索结果。
type SearchHit struct {
	DocID      int64            `json:"doc_id"`
	Score      float32          `json:"score"`
	Highlights []HighlightRange `json:"highlights,omitempty"`
	Doc        *SearchDocView   `json:"doc,omitempty"`
}

func buildHighlightsForDoc(doc *document.Document, terms []string, highlighter *highlight.Highlighter) []HighlightRange {
	if doc == nil || len(terms) == 0 || highlighter == nil {
		return nil
	}

	for _, field := range doc.Fields {
		if field.FieldType != document.FieldTypeText && field.FieldType != document.FieldTypeString {
			continue
		}
		text := field.StringValue()
		if text == "" {
			continue
		}
		for _, term := range terms {
			fragment, ok := highlighter.Extract(text, term)
			if !ok {
				continue
			}
			return []HighlightRange{
				{
					Field:    field.Name,
					Start:    fragment.Start,
					End:      fragment.End,
					Fragment: fragment.Text,
				},
			}
		}

		// fuzzy/wildcard 命中可能不是精确子串，回退到近似高亮。
		for _, term := range terms {
			start, end, ok := findApproxSpan(text, term)
			if !ok {
				continue
			}
			return []HighlightRange{
				{
					Field:    field.Name,
					Start:    start,
					End:      end,
					Fragment: extractFragment(text, start, end, 160),
				},
			}
		}
	}
	return nil
}

type tokenSpan struct {
	text  string
	start int
	end   int
}

func findApproxSpan(text, term string) (int, int, bool) {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return 0, 0, false
	}

	for _, tok := range tokenizeSpans(text) {
		candidate := strings.ToLower(tok.text)
		if strings.HasPrefix(candidate, term) {
			return tok.start, tok.end, true
		}
		// 仅对较长词做近似匹配，降低误命中。
		if len([]rune(term)) < 3 {
			continue
		}
		if editDistanceAtMostOne(candidate, term) <= 1 {
			return tok.start, tok.end, true
		}
	}

	return 0, 0, false
}

func tokenizeSpans(text string) []tokenSpan {
	runes := []rune(text)
	tokens := make([]tokenSpan, 0, 8)
	start := -1

	for i, r := range runes {
		if isTokenRune(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			tokens = append(tokens, tokenSpan{
				text:  string(runes[start:i]),
				start: start,
				end:   i,
			})
			start = -1
		}
	}
	if start >= 0 {
		tokens = append(tokens, tokenSpan{
			text:  string(runes[start:]),
			start: start,
			end:   len(runes),
		})
	}

	return tokens
}

func isTokenRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func extractFragment(text string, start, end, max int) string {
	runes := []rune(text)
	if start < 0 || end > len(runes) || start >= end {
		return text
	}
	if max <= 0 || len(runes) <= max {
		return text
	}

	half := max / 2
	fragStart := start - half
	if fragStart < 0 {
		fragStart = 0
	}
	fragEnd := fragStart + max
	if fragEnd < end {
		fragEnd = end
	}
	if fragEnd > len(runes) {
		fragEnd = len(runes)
	}
	if fragEnd-fragStart > max {
		fragStart = fragEnd - max
		if fragStart < 0 {
			fragStart = 0
		}
	}

	return string(runes[fragStart:fragEnd])
}

func editDistanceAtMostOne(a, b string) int {
	if a == b {
		return 0
	}

	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)
	if absInt(la-lb) > 1 {
		return 2
	}

	if la == lb {
		diff := 0
		for i := 0; i < la; i++ {
			if ra[i] != rb[i] {
				diff++
				if diff > 1 {
					return 2
				}
			}
		}
		return diff
	}

	if la > lb {
		ra, rb = rb, ra
		la, lb = lb, la
	}

	i, j := 0, 0
	edits := 0
	for i < la && j < lb {
		if ra[i] == rb[j] {
			i++
			j++
			continue
		}
		edits++
		if edits > 1 {
			return 2
		}
		j++
	}
	if j < lb {
		edits++
	}
	if edits > 1 {
		return 2
	}
	return edits
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
