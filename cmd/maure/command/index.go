package command

import (
	"fmt"
	"os"

	"maure/pkg/store"
)

// InitCommand 初始化索引目录。
type InitCommand struct {
	*BaseCommand
}

// NewInitCommand 创建初始化命令。
func NewInitCommand() *InitCommand {
	cmd := &InitCommand{
		BaseCommand: NewBaseCommand("init", "maure init <path>", "初始化索引目录"),
	}
	cmd.desc = "创建新的索引目录"
	return cmd
}

// Execute 执行初始化。
func (c *InitCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少索引目录路径"))
	}

	path := args[0]

	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		ExitWithError(fmt.Errorf("创建目录失败: %w", err))
	}

	// 初始化 FSDirectory
	dir, err := store.NewFSDirectory(path)
	if err != nil {
		ExitWithError(fmt.Errorf("初始化索引失败: %w", err))
	}
	defer dir.Close()

	fmt.Printf("索引目录已创建: %s\n", path)
	return nil
}

// OpenCommand 打开索引目录。
type OpenCommand struct {
	*BaseCommand
}

// NewOpenCommand 创建打开命令。
func NewOpenCommand() *OpenCommand {
	cmd := &OpenCommand{
		BaseCommand: NewBaseCommand("open", "maure open <path>", "打开索引目录"),
	}
	cmd.desc = "打开现有的索引目录"
	return cmd
}

// Execute 执行打开。
func (c *OpenCommand) Execute(args []string, opts GlobalOptions) error {
	path := opts.IndexPath
	if len(args) >= 1 {
		path = args[0]
	}

	if path == "" {
		ExitWithError(fmt.Errorf("请指定索引目录路径"))
	}

	dir, err := store.NewFSDirectory(path)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}
	defer dir.Close()

	fmt.Printf("索引已打开: %s\n", path)
	return nil
}

// InfoCommand 显示索引信息。
type InfoCommand struct {
	*BaseCommand
}

// NewInfoCommand 创建信息命令。
func NewInfoCommand() *InfoCommand {
	cmd := &InfoCommand{
		BaseCommand: NewBaseCommand("info", "maure info", "显示索引信息"),
	}
	cmd.desc = "显示索引的基本信息"
	return cmd
}

// Execute 执行显示信息。
func (c *InfoCommand) Execute(args []string, opts GlobalOptions) error {
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

	files, _ := dir.ListFiles()
	manifest := dir.Manifest()

	fmt.Printf("索引目录: %s\n", path)
	fmt.Printf("文件数量: %d\n", len(files))
	fmt.Printf("快照文件: %s\n", manifest.SnapPath)
	fmt.Printf("最后文档ID: %d\n", manifest.LastDocID)
	return nil
}

// StatsCommand 显示索引统计。
type StatsCommand struct {
	*BaseCommand
}

// NewStatsCommand 创建统计命令。
func NewStatsCommand() *StatsCommand {
	cmd := &StatsCommand{
		BaseCommand: NewBaseCommand("stats", "maure stats", "显示索引统计"),
	}
	cmd.desc = "显示索引的详细统计信息"
	return cmd
}

// Execute 执行统计。
func (c *StatsCommand) Execute(args []string, opts GlobalOptions) error {
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

	terms := reader.GetTerms()

	fmt.Printf("文档数量: %d\n", reader.DocCount())
	fmt.Printf("词项数量: %d\n", len(terms))

	if len(terms) > 0 {
		if len(terms) <= 10 {
			fmt.Printf("词项列表: %v\n", terms)
		} else {
			fmt.Printf("部分词项: %v\n", terms[:10])
		}
	}

	return nil
}

// CompactCommand 压缩索引。
type CompactCommand struct {
	*BaseCommand
}

// NewCompactCommand 创建压缩命令。
func NewCompactCommand() *CompactCommand {
	cmd := &CompactCommand{
		BaseCommand: NewBaseCommand("compact", "maure compact", "优化/压缩索引"),
	}
	cmd.desc = "优化索引结构，回收空间"
	return cmd
}

// Execute 执行压缩。
func (c *CompactCommand) Execute(args []string, opts GlobalOptions) error {
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

	writer, err := dir.OpenIndexWriter()
	if err != nil {
		ExitWithError(fmt.Errorf("打开写入器失败: %w", err))
	}

	// 提交当前状态
	if err := writer.Commit(); err != nil {
		writer.Close()
		ExitWithError(fmt.Errorf("提交失败: %w", err))
	}
	writer.Close()

	fmt.Println("索引压缩完成")
	return nil
}

// DeleteCommand 删除索引。
type DeleteCommand struct {
	*BaseCommand
}

// NewDeleteCommand 创建删除命令。
func NewDeleteCommand() *DeleteCommand {
	cmd := &DeleteCommand{
		BaseCommand: NewBaseCommand("delete", "maure delete <path>", "删除索引目录"),
	}
	cmd.desc = "删除整个索引目录"
	return cmd
}

// Execute 执行删除。
func (c *DeleteCommand) Execute(args []string, opts GlobalOptions) error {
	if len(args) < 1 {
		ExitWithError(fmt.Errorf("缺少索引目录路径"))
	}

	path := args[0]

	if !exists(path) {
		ExitWithError(fmt.Errorf("目录不存在: %s", path))
	}

	if err := os.RemoveAll(path); err != nil {
		ExitWithError(fmt.Errorf("删除失败: %w", err))
	}

	fmt.Printf("索引已删除: %s\n", path)
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	RegisterCommand(NewInitCommand())
	RegisterCommand(NewOpenCommand())
	RegisterCommand(NewInfoCommand())
	RegisterCommand(NewStatsCommand())
	RegisterCommand(NewCompactCommand())
	RegisterCommand(NewDeleteCommand())
}
