package command

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"maure/pkg/index"
	"maure/pkg/query"
)

// SearchCommand 搜索命令。
type SearchCommand struct {
	*BaseCommand
	topN   int
	output string
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

	// 加载现有文档
	reader := ctx.Reader
	for i := int64(1); i <= reader.DocCount(); i++ {
		doc, err := reader.GetDocument(i)
		if err != nil {
			continue
		}
		ramIdx.Add(doc)
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

	// 输出结果
	switch c.output {
	case "json":
		outputJSON(os.Stdout, results)
	default:
		outputText(os.Stdout, results)
	}

	return nil
}

func outputText(w *os.File, results []index.ScoreDoc) {
	fmt.Fprintf(w, "找到 %d 个结果:\n\n", len(results))

	for i, r := range results {
		fmt.Fprintf(w, "[%d] DocID=%d Score=%.4f\n", i+1, r.DocID, r.Score)
	}
}

func outputJSON(w *os.File, results []index.ScoreDoc) {
	fmt.Fprintf(w, `{"total":%d,"results":[`, len(results))
	for i, r := range results {
		if i > 0 {
			fmt.Fprintf(w, ",")
		}
		fmt.Fprintf(w, `{"doc_id":%d,"score":%.4f}`, r.DocID, r.Score)
	}
	fmt.Fprintf(w, "]}")
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
		ramIdx.Add(doc)
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
