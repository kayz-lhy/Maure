package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"maure/pkg/document"
	"maure/pkg/store"
)

// AddCommand 添加文档命令。
type AddCommand struct {
	*BaseCommand
	recursive bool
}

// NewAddCommand 创建添加命令。
func NewAddCommand() *AddCommand {
	cmd := &AddCommand{
		BaseCommand: NewBaseCommand("add", "maure add <file>", "添加文件到索引"),
	}
	cmd.desc = "添加单个文件到索引"
	cmd.flags.BoolVar(&cmd.recursive, "r", false, "递归添加目录")

	// 重置 flags 以包含新标志
	cmd.flags = flag.NewFlagSet("add", flag.ContinueOnError)
	cmd.flags.BoolVar(&cmd.recursive, "r", false, "递归添加目录")
	return cmd
}

// Execute 执行添加。
func (c *AddCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少文件路径"))
	}

	path := args[0]

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	count := 0
	if c.recursive {
		count, err = addDirectory(ctx.Writer, path)
	} else {
		var doc *document.Document
		doc, err = ReadDocument(path)
		if err == nil {
			_, err = ctx.Writer.AddDocument(doc)
			count = 1
		}
	}

	if err != nil {
		ExitWithError(fmt.Errorf("添加文档失败: %w", err))
	}

	// 提交
	if err := ctx.Writer.Commit(); err != nil {
		ExitWithError(fmt.Errorf("提交失败: %w", err))
	}

	fmt.Printf("已添加 %d 个文档\n", count)
	return nil
}

func addDirectory(writer store.IndexWriter, dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" || ext == ".json" {
			doc, err := ReadDocument(path)
			if err != nil {
				return err
			}
			_, err = writer.AddDocument(doc)
			if err == nil {
				count++
			}
		}
		return nil
	})
	return count, err
}

// AddDirCommand 批量添加命令。
type AddDirCommand struct {
	*BaseCommand
}

// NewAddDirCommand 创建批量添加命令。
func NewAddDirCommand() *AddDirCommand {
	cmd := &AddDirCommand{
		BaseCommand: NewBaseCommand("add-dir", "maure add-dir <dir>", "批量添加目录下文件"),
	}
	cmd.desc = "批量添加目录下所有支持的文件到索引"
	return cmd
}

// Execute 执行批量添加。
func (c *AddDirCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少目录路径"))
	}

	path := args[0]

	if !exists(path) {
		ExitWithError(fmt.Errorf("目录不存在: %s", path))
	}

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	count, err := addDirectory(ctx.Writer, path)
	if err != nil {
		ExitWithError(fmt.Errorf("遍历目录失败: %w", err))
	}

	if err := ctx.Writer.Commit(); err != nil {
		ExitWithError(fmt.Errorf("提交失败: %w", err))
	}

	fmt.Printf("已添加 %d 个文档\n", count)
	return nil
}

// ImportCommand 导入命令。
type ImportCommand struct {
	*BaseCommand
}

// NewImportCommand 创建导入命令。
func NewImportCommand() *ImportCommand {
	cmd := &ImportCommand{
		BaseCommand: NewBaseCommand("import", "maure import <file>", "从 JSON 导入文档"),
	}
	cmd.desc = "从 JSON 文件批量导入文档"
	return cmd
}

// ImportDocument 导入的文档格式。
type ImportDocument struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

// Execute 执行导入。
func (c *ImportCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少 JSON 文件路径"))
	}

	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		ExitWithError(fmt.Errorf("读取文件失败: %w", err))
	}

	var docs []ImportDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		ExitWithError(fmt.Errorf("解析 JSON 失败: %w", err))
	}

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	count := 0
	for _, d := range docs {
		doc := document.NewDocument()
		doc.SetID(d.ID)
		for name, value := range d.Fields {
			switch v := value.(type) {
			case string:
				doc.Add(document.NewTextField(name, v))
			case int:
				doc.Add(document.NewInt64Field(name, int64(v)))
			case float64:
				doc.Add(document.NewFloat64Field(name, v))
			case bool:
				doc.Add(document.NewStoredField(name, v))
			}
		}
		_, err = ctx.Writer.AddDocument(doc)
		if err == nil {
			count++
		}
	}

	if err := ctx.Writer.Commit(); err != nil {
		ExitWithError(fmt.Errorf("提交失败: %w", err))
	}

	fmt.Printf("已导入 %d 个文档\n", count)
	return nil
}

// DeleteDocCommand 删除文档命令。
type DeleteDocCommand struct {
	*BaseCommand
}

// NewDeleteDocCommand 创建删除文档命令。
func NewDeleteDocCommand() *DeleteDocCommand {
	cmd := &DeleteDocCommand{
		BaseCommand: NewBaseCommand("delete-doc", "maure delete-doc <doc-id>", "删除文档"),
	}
	cmd.desc = "根据文档 ID 删除文档"
	return cmd
}

// Execute 执行删除。
func (c *DeleteDocCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少文档 ID"))
	}

	var docID int64
	_, err := fmt.Sscanf(args[0], "%d", &docID)
	if err != nil {
		ExitWithError(fmt.Errorf("无效的文档 ID: %s", args[0]))
	}

	ctx, err := NewIndexContext(opts.IndexPath, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer ctx.Close()

	if err := ctx.Writer.Delete(docID); err != nil {
		ExitWithError(fmt.Errorf("删除文档失败: %w", err))
	}

	if err := ctx.Writer.Commit(); err != nil {
		ExitWithError(fmt.Errorf("提交失败: %w", err))
	}

	fmt.Printf("文档 %d 已删除\n", docID)
	return nil
}

// ListCommand 列出文档命令。
type ListCommand struct {
	*BaseCommand
	limit int
}

// NewListCommand 创建列出命令。
func NewListCommand() *ListCommand {
	cmd := &ListCommand{
		BaseCommand: NewBaseCommand("list", "maure list", "列出所有文档"),
	}
	cmd.desc = "列出索引中的所有文档"
	cmd.flags.IntVar(&cmd.limit, "n", 100, "最大显示数量")
	cmd.flags = flag.NewFlagSet("list", flag.ContinueOnError)
	cmd.flags.IntVar(&cmd.limit, "n", 100, "最大显示数量")
	return cmd
}

// Execute 执行列出。
func (c *ListCommand) Execute(args []string, opts GlobalOptions) error {
	path := opts.IndexPath
	if len(args) >= 1 {
		path = args[0]
	}

	if path == "" {
		path = "."
	}

	dir, err := store.NewFSDirectory(path)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer dir.Close()

	reader, err := dir.OpenIndexReader()
	if err != nil {
		ExitWithError(fmt.Errorf("打开读取器失败: %w", err))
	}
	defer reader.Close()

	fmt.Printf("文档总数: %d\n\n", reader.DocCount())

	count := 0
	for i := int64(1); i <= reader.DocCount() && count < c.limit; i++ {
		doc, err := reader.GetDocument(i)
		if err != nil {
			continue
		}
		fmt.Printf("DocID %d: %s\n", i, doc.ID())
		count++
	}

	return nil
}

// init 注册文档操作命令。
func init() {
	RegisterCommand(NewAddCommand())
	RegisterCommand(NewAddDirCommand())
	RegisterCommand(NewImportCommand())
	RegisterCommand(NewDeleteDocCommand())
	RegisterCommand(NewListCommand())
}
