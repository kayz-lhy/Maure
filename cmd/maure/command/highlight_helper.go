package command

import (
	"maure/pkg/document"
	"maure/pkg/highlight"
)

// HighlightRange 表示字段中的高亮位置。
type HighlightRange struct {
	Field    string `json:"field"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Fragment string `json:"fragment"`
}

// SearchHit 是带高亮信息的搜索结果。
type SearchHit struct {
	DocID      int64            `json:"doc_id"`
	Score      float32          `json:"score"`
	Highlights []HighlightRange `json:"highlights,omitempty"`
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
	}
	return nil
}
