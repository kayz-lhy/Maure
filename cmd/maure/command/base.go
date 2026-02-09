// Package command 定义了 CLI 命令的接口和实现。
package command

import (
	"flag"
	"fmt"
	"io"
	"os"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/store"
)

// Version CLI 版本号。
const Version = "0.1.0"

// 全局选项
type GlobalOptions struct {
	IndexPath string
	Format    string
	Verbose   bool
	Analyzer  string
}

// Command 定义了 CLI 命令的接口。
type Command interface {
	Name() string
	Usage() string
	Description() string
	Flags() *flag.FlagSet
	Execute(args []string, opts GlobalOptions) error
}

// 命令注册表
var commands = make(map[string]Command)

// RegisterCommand 注册命令。
func RegisterCommand(cmd Command) {
	commands[cmd.Name()] = cmd
}

// GetCommand 获取命令。
func GetCommand(name string) (Command, bool) {
	cmd, ok := commands[name]
	return cmd, ok
}

// ListCommands 列出所有命令。
func ListCommands() []Command {
	cmds := make([]Command, 0, len(commands))
	for _, cmd := range commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// BaseCommand 提供命令的公共功能。
type BaseCommand struct {
	flags *flag.FlagSet
	usage string
	desc  string
}

// NewBaseCommand 创建基础命令。
func NewBaseCommand(name, usage, desc string) *BaseCommand {
	return &BaseCommand{
		flags: flag.NewFlagSet(name, flag.ContinueOnError),
		usage: usage,
		desc:  desc,
	}
}

// Name 返回命令名。
func (b *BaseCommand) Name() string {
	return b.flags.Name()
}

// Usage 返回命令用法。
func (b *BaseCommand) Usage() string {
	return b.usage
}

// Description 返回命令描述。
func (b *BaseCommand) Description() string {
	return b.desc
}

// Flags 返回标志集。
func (b *BaseCommand) Flags() *flag.FlagSet {
	return b.flags
}

// IndexContext 索引操作上下文。
type IndexContext struct {
	Dir      store.Directory
	Writer   store.IndexWriter
	Reader   store.IndexReader
	Analyzer analyzer.Analyzer
}

// NewIndexContext 创建索引上下文。
func NewIndexContext(path string, opts GlobalOptions) (*IndexContext, error) {
	var dir store.Directory
	var err error

	if opts.Analyzer == "ram" {
		dir = store.NewRAMDirectory()
	} else {
		dir, err = store.NewFSDirectory(path)
		if err != nil {
			return nil, fmt.Errorf("打开索引目录: %w", err)
		}
	}

	writer, err := dir.CreateIndexWriter()
	if err != nil {
		return nil, fmt.Errorf("创建写入器: %w", err)
	}

	reader, err := dir.OpenIndexReader()
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("创建读取器: %w", err)
	}

	var ana analyzer.Analyzer
	if opts.Analyzer == "standard" {
		ana = analyzer.NewStandardAnalyzer()
	} else {
		ana = analyzer.NewStandardAnalyzer()
	}

	return &IndexContext{
		Dir:      dir,
		Writer:   writer,
		Reader:   reader,
		Analyzer: ana,
	}, nil
}

// Close 关闭上下文。
func (c *IndexContext) Close() error {
	if c.Writer != nil {
		c.Writer.Close()
	}
	if c.Reader != nil {
		c.Reader.Close()
	}
	if c.Dir != nil {
		return c.Dir.Close()
	}
	return nil
}

// OutputFormatter 输出格式化接口。
type OutputFormatter interface {
	Header(title string)
	Footer()
	Write(data interface{})
}

// TextFormatter 文本格式化器。
type TextFormatter struct {
	writer io.Writer
}

// NewTextFormatter 创建文本格式化器。
func NewTextFormatter(w io.Writer) *TextFormatter {
	return &TextFormatter{writer: w}
}

// Header 打印标题。
func (f *TextFormatter) Header(title string) {
	fmt.Fprintf(f.writer, "\n=== %s ===\n\n", title)
}

// Footer 打印结尾。
func (f *TextFormatter) Footer() {
	fmt.Fprintln(f.writer)
}

// Write 写入数据。
func (f *TextFormatter) Write(data interface{}) {
	fmt.Fprintf(f.writer, "%v\n", data)
}

// JSONFormatter JSON 格式化器。
type JSONFormatter struct {
	writer io.Writer
}

// NewJSONFormatter 创建 JSON 格式化器。
func NewJSONFormatter(w io.Writer) *JSONFormatter {
	return &JSONFormatter{writer: w}
}

// Header 打印标题。
func (f *JSONFormatter) Header(title string) {
	fmt.Fprintf(f.writer, "{\n  \"title\": %q,\n", title)
	fmt.Fprintf(f.writer, "  \"data\": ")
}

// Footer 打印结尾。
func (f *JSONFormatter) Footer() {
	fmt.Fprintf(f.writer, "}\n")
}

// Write 写入数据。
func (f *JSONFormatter) Write(data interface{}) {
	fmt.Fprintf(f.writer, "%v", data)
}

// NewFormatter 创建格式化器。
func NewFormatter(format string, w io.Writer) OutputFormatter {
	switch format {
	case "json":
		return NewJSONFormatter(w)
	default:
		return NewTextFormatter(w)
	}
}

// ReadDocument 从文件读取文档。
func ReadDocument(path string) (*document.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := document.NewDocument()
	doc.SetID(path)

	// 简单处理：整个文件内容作为 content 字段
	doc.Add(document.NewTextField("content", string(data)))
	doc.Add(document.NewStringField("path", path))

	return doc, nil
}

// ExitWithError 退出并打印错误。
func ExitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
