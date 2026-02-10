package command

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"maure/pkg/aggregate"
	"maure/pkg/document"
	"maure/pkg/highlight"
	"maure/pkg/index"
	"maure/pkg/query"
)

// SearchCommand 搜索命令。
type SearchCommand struct {
	*BaseCommand
	topN   int
	output string
	agg    string
	group  string
}

// NewSearchCommand 创建搜索命令。
func NewSearchCommand() *SearchCommand {
	cmd := &SearchCommand{
		BaseCommand: NewBaseCommand("search", "maure search <query>", "搜索文档"),
	}
	cmd.desc = "搜索索引中的文档"
	cmd.flags.IntVar(&cmd.topN, "n", 10, "返回结果数量")
	cmd.flags.StringVar(&cmd.output, "o", "text", "输出格式 (text/json)")
	cmd.flags = flag.NewFlagSet("search", flag.ContinueOnError)
	cmd.flags.IntVar(&cmd.topN, "n", 10, "返回结果数量")
	cmd.flags.StringVar(&cmd.output, "o", "text", "输出格式 (text/json)")
	cmd.flags.StringVar(&cmd.agg, "agg", "", "聚合函数（支持: count）")
	cmd.flags.StringVar(&cmd.group, "group", "", "分组方式（如: level 或 time(5m)）")
	return cmd
}

// Execute 执行搜索。
func (c *SearchCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少查询语句"))
	}

	queryStr := args[0]

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	// 创建内存索引
	ramIdx := index.NewRAMIndex(ctx.Analyzer)
	docIDMap := make(map[int64]int64)

	// 加载现有文档
	reader := ctx.Reader
	for i := int64(1); i <= reader.DocCount(); i++ {
		doc, err := reader.GetDocument(i)
		if err != nil {
			continue
		}
		ramDocID, err := ramIdx.Add(doc)
		if err != nil {
			continue
		}
		docIDMap[ramDocID] = i
	}

	// 解析查询
	parsedQuery, err := query.NewQueryParser().Parse(queryStr)
	if err != nil {
		ExitWithError(fmt.Errorf("解析查询失败: %w", err))
	}

	if parsedQuery == nil {
		fmt.Println("未找到匹配结果")
		return nil
	}

	// 执行搜索
	results, err := ramIdx.Search(parsedQuery, c.topN)
	if err != nil {
		ExitWithError(fmt.Errorf("搜索失败: %w", err))
	}

	terms := query.ExtractTerms(parsedQuery)
	highlighter := highlight.NewHighlighter()
	hits := make([]SearchHit, 0, len(results))
	docsForAgg := make([]*document.Document, 0, len(results))
	for _, r := range results {
		sourceDocID := r.DocID
		if mappedDocID, ok := docIDMap[r.DocID]; ok {
			sourceDocID = mappedDocID
		}

		var highlights []HighlightRange
		doc, err := reader.GetDocument(sourceDocID)
		if err == nil {
			highlights = buildHighlightsForDoc(doc, terms, highlighter)
			docsForAgg = append(docsForAgg, doc)
		}

		hits = append(hits, SearchHit{
			DocID:      sourceDocID,
			Score:      r.Score,
			Highlights: highlights,
		})
	}

	aggResult, err := aggregate.Build(docsForAgg, c.agg, c.group)
	if err != nil {
		ExitWithError(fmt.Errorf("聚合失败: %w", err))
	}
	showCount := c.agg != ""

	// 输出结果
	switch c.output {
	case "json":
		outputJSON(os.Stdout, hits, aggResult, showCount)
	default:
		outputText(os.Stdout, hits, aggResult, showCount)
	}

	return nil
}

func outputText(w *os.File, hits []SearchHit, aggResult *aggregate.Result, showCount bool) {
	fmt.Fprintf(w, "找到 %d 个结果:\n\n", len(hits))

	for i, hit := range hits {
		fmt.Fprintf(w, "[%d] DocID=%d Score=%.4f\n", i+1, hit.DocID, hit.Score)
		if len(hit.Highlights) > 0 {
			hl := hit.Highlights[0]
			fmt.Fprintf(w, "    Highlight field=%s range=[%d,%d) fragment=%q\n", hl.Field, hl.Start, hl.End, hl.Fragment)
		}
	}

	if aggResult != nil {
		if showCount {
			fmt.Fprintf(w, "\nAgg count=%d\n", aggResult.Count)
		}
		if len(aggResult.Buckets) > 0 {
			fmt.Fprintln(w, "\nAgg buckets:")
			for _, b := range aggResult.Buckets {
				fmt.Fprintf(w, "  %s: %d\n", b.Key, b.Count)
			}
		}
	}
}

func outputJSON(w *os.File, hits []SearchHit, aggResult *aggregate.Result, showCount bool) {
	fmt.Fprintf(w, `{"total":%d,"results":[`, len(hits))
	for i, hit := range hits {
		if i > 0 {
			fmt.Fprintf(w, ",")
		}
		fmt.Fprintf(w, `{"doc_id":%d,"score":%.4f`, hit.DocID, hit.Score)
		if len(hit.Highlights) > 0 {
			fmt.Fprintf(w, `,"highlights":[`)
			for j, hl := range hit.Highlights {
				if j > 0 {
					fmt.Fprintf(w, ",")
				}
				fmt.Fprintf(w, `{"field":%q,"start":%d,"end":%d,"fragment":%q}`, hl.Field, hl.Start, hl.End, hl.Fragment)
			}
			fmt.Fprintf(w, `]`)
		}
		fmt.Fprintf(w, `}`)
	}
	fmt.Fprintf(w, "]")
	if aggResult != nil && (showCount || len(aggResult.Buckets) > 0) {
		fmt.Fprintf(w, `,"aggregations":{`)
		wrote := false
		if showCount {
			fmt.Fprintf(w, `"count":%d`, aggResult.Count)
			wrote = true
		}
		if len(aggResult.Buckets) > 0 {
			if wrote {
				fmt.Fprintf(w, ",")
			}
			fmt.Fprintf(w, `"buckets":[`)
			for i, b := range aggResult.Buckets {
				if i > 0 {
					fmt.Fprintf(w, ",")
				}
				fmt.Fprintf(w, `{"key":%q,"count":%d}`, b.Key, b.Count)
			}
			fmt.Fprintf(w, "]")
		}
		fmt.Fprintf(w, "}")
	}
	fmt.Fprintf(w, "}")
}

// CountCommand 统计命令。
type CountCommand struct {
	*BaseCommand
}

// NewCountCommand 创建统计命令。
func NewCountCommand() *CountCommand {
	cmd := &CountCommand{
		BaseCommand: NewBaseCommand("count", "maure count <query>", "统计匹配文档数"),
	}
	cmd.desc = "统计匹配查询的文档数量"
	return cmd
}

// Execute 执行统计。
func (c *CountCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少查询语句"))
	}

	queryStr := args[0]

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	ramIdx := index.NewRAMIndex(ctx.Analyzer)
	reader := ctx.Reader
	for i := int64(1); i <= reader.DocCount(); i++ {
		doc, err := reader.GetDocument(i)
		if err != nil {
			continue
		}
		if _, err := ramIdx.Add(doc); err != nil {
			continue
		}
	}

	parsedQuery, err := query.NewQueryParser().Parse(queryStr)
	if err != nil {
		ExitWithError(fmt.Errorf("解析查询失败: %w", err))
	}

	if parsedQuery == nil {
		fmt.Println("0")
		return nil
	}

	results, err := ramIdx.Search(parsedQuery, 10000)
	if err != nil {
		ExitWithError(fmt.Errorf("搜索失败: %w", err))
	}

	fmt.Println(len(results))
	return nil
}

// TermsCommand 词项命令。
type TermsCommand struct {
	*BaseCommand
	prefix string
	limit  int
}

// NewTermsCommand 创建词项命令。
func NewTermsCommand() *TermsCommand {
	cmd := &TermsCommand{
		BaseCommand: NewBaseCommand("terms", "maure terms [prefix]", "列出词项"),
	}
	cmd.desc = "列出索引中的所有词项"
	cmd.flags.StringVar(&cmd.prefix, "p", "", "前缀过滤")
	cmd.flags.IntVar(&cmd.limit, "n", 100, "最大显示数量")
	cmd.flags = flag.NewFlagSet("terms", flag.ContinueOnError)
	cmd.flags.StringVar(&cmd.prefix, "p", "", "前缀过滤")
	cmd.flags.IntVar(&cmd.limit, "n", 100, "最大显示数量")
	return cmd
}

// Execute 执行列出词项。
func (c *TermsCommand) Execute(args []string, opts GlobalOptions) error {
	path := opts.IndexPath
	if len(args) >= 1 {
		path = args[0]
	}

	if path == "" {
		path = "."
	}

	ctx, err := NewIndexContext(path, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	terms := ctx.Reader.GetTerms()
	if c.prefix != "" {
		filtered := make([]string, 0)
		for _, t := range terms {
			if len(t) >= len(c.prefix) && t[:len(c.prefix)] == c.prefix {
				filtered = append(filtered, t)
			}
		}
		terms = filtered
	}

	if len(terms) > c.limit {
		terms = terms[:c.limit]
	}

	sort.Strings(terms)

	fmt.Printf("词项总数: %d\n", len(ctx.Reader.GetTerms()))
	fmt.Printf("显示数量: %d\n\n", len(terms))
	for _, t := range terms {
		fmt.Println(t)
	}

	return nil
}

// SimilarityCommand 切换评分算法。
type SimilarityCommand struct {
	*BaseCommand
}

// NewSimilarityCommand 创建评分算法命令。
func NewSimilarityCommand() *SimilarityCommand {
	cmd := &SimilarityCommand{
		BaseCommand: NewBaseCommand("similarity", "maure similarity [bm25|tfidf]", "设置评分算法"),
	}
	cmd.desc = "查询或设置评分算法"
	return cmd
}

// Execute 执行设置。
func (c *SimilarityCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) >= 1 {
		algo := args[0]
		switch algo {
		case "bm25":
			fmt.Println("评分算法: BM25")
		case "tfidf":
			fmt.Println("评分算法: TF-IDF")
		default:
			fmt.Printf("未知评分算法: %s\n", algo)
		}
	} else {
		fmt.Println("当前评分算法: BM25")
		fmt.Println("可用: bm25, tfidf")
	}
	return nil
}

func init() {
	RegisterCommand(NewSearchCommand())
	RegisterCommand(NewCountCommand())
	RegisterCommand(NewTermsCommand())
	RegisterCommand(NewSimilarityCommand())
}
