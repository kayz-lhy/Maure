package store

import (
	"maure/pkg/analyzer"
	"maure/pkg/document"
)

// TermStat 聚合单个词项在单个文档中的统计信息。
type TermStat struct {
	Freq      int
	Positions []int
}

// AnalyzeDocument 统一分析文档并返回词项统计和文档长度。
//
// 规则：
//   - 仅处理 Indexed 字段
//   - Tokenized 字段使用 analyzer 分词
//   - 非 Tokenized 字段整体作为一个词项
//   - docLength 仅统计 Tokenized 字段的 token 数量
func AnalyzeDocument(doc *document.Document, a analyzer.Analyzer) (map[string]TermStat, int) {
	stats := make(map[string]TermStat)
	docLength := 0

	for _, field := range doc.Fields {
		if !field.Indexed {
			continue
		}

		if field.Tokenized {
			stream := a.Analyze(field.Name, field.StringValue())
			for stream.Next() {
				tok := stream.Current()
				existing := stats[tok.Text]
				existing.Freq++
				existing.Positions = append(existing.Positions, tok.Position)
				stats[tok.Text] = existing
				docLength++
			}
			stream.Close()
			continue
		}

		term := field.StringValue()
		if term == "" {
			continue
		}
		existing := stats[term]
		existing.Freq++
		existing.Positions = append(existing.Positions, 0)
		stats[term] = existing
	}

	return stats, docLength
}
