package command

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"maure/pkg/document"
	"maure/pkg/logparser"
)

const maxLogLineBytes = 4 * 1024 * 1024

// ParseLogCommand 解析日志并写入索引。
type ParseLogCommand struct {
	*BaseCommand
	format string
}

// NewParseLogCommand 创建 parse-log 命令。
func NewParseLogCommand() *ParseLogCommand {
	cmd := &ParseLogCommand{
		BaseCommand: NewBaseCommand("parse-log", "maure parse-log <file>", "解析日志并导入索引"),
	}
	cmd.desc = "支持 JSON/Logback/auto 三种格式解析"
	cmd.flags = flag.NewFlagSet("parse-log", flag.ContinueOnError)
	cmd.flags.StringVar(&cmd.format, "log-format", logparser.FormatAuto, "日志格式: auto|json|logback")
	return cmd
}

// Execute 执行 parse-log。
func (c *ParseLogCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少日志文件路径"))
	}
	filePath := args[0]

	if !strings.EqualFold(c.format, logparser.FormatAuto) {
		if _, err := logparser.GetParser(c.format); err != nil {
			ExitWithError(err)
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		ExitWithError(fmt.Errorf("打开日志文件失败: %w", err))
	}
	defer file.Close()

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxLogLineBytes)
	added := 0
	skipped := 0
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parser, parserErr := logparser.GetParserForLine(c.format, line)
		if parserErr != nil {
			skipped++
			continue
		}

		doc, parseErr := parser.Parse(line)
		if parseErr != nil {
			skipped++
			continue
		}

		doc.Add(document.NewStringField("source", filePath))
		doc.Add(document.NewInt64Field("line_no", int64(lineNo)))
		doc.SetID(fmt.Sprintf("%s:%d", filePath, lineNo))

		if _, addErr := ctx.Writer.AddDocument(doc); addErr != nil {
			skipped++
			continue
		}
		added++
	}

	if err := scanner.Err(); err != nil {
		ExitWithError(fmt.Errorf("读取日志文件失败: %w", err))
	}

	if err := ctx.Writer.Commit(); err != nil {
		ExitWithError(fmt.Errorf("提交索引失败: %w", err))
	}

	fmt.Printf("日志解析完成: added=%d skipped=%d\n", added, skipped)
	return nil
}

func init() {
	RegisterCommand(NewParseLogCommand())
}
