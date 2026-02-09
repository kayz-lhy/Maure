package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"maure/cmd/maure/command"
)

func main() {
	// 解析全局选项
	globalOpts := command.GlobalOptions{
		IndexPath: ".",
		Format:    "text",
		Verbose:   false,
		Analyzer:  "standard",
	}

	flag.StringVar(&globalOpts.IndexPath, "index", ".", "索引目录路径")
	flag.StringVar(&globalOpts.Format, "format", "text", "输出格式 (text/json)")
	flag.BoolVar(&globalOpts.Verbose, "v", false, "详细输出")
	flag.StringVar(&globalOpts.Analyzer, "analyzer", "standard", "分析器类型")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Maure Search Engine v%s

Usage: %s [options] <command> [arguments]

Options:
`, command.Version, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Commands:
  init <path>        初始化索引目录
  info [path]        显示索引信息
  stats [path]       显示索引统计
  compact [path]     优化/压缩索引

  add <file>         添加文件到索引
  add-dir <dir>      批量添加目录
  import <file>      从 JSON 导入
  delete-doc <id>    删除文档
  list [path]        列出所有文档

  search <query>     搜索文档
  count <query>      统计匹配数
  terms [prefix]     列出词项

  serve [--port]     启动 HTTP 服务
  version            显示版本
  help              显示帮助

Examples:
  %s init ./myindex
  %s add ./docs/file.txt
  %s search "golang tutorial"
  %s serve

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	}

	flag.Parse()

	// 处理 help 命令
	if flag.NArg() == 0 || flag.Arg(0) == "help" || flag.Arg(0) == "--help" {
		flag.Usage()
		os.Exit(0)
	}

	// 获取子命令
	cmdName := flag.Arg(0)
	var cmdArgs []string
	if flag.NArg() > 1 {
		cmdArgs = flag.Args()[1:]
	}

	// 处理内置命令
	switch cmdName {
	case "version", "-v", "--version":
		fmt.Printf("Maure Search Engine v%s\n", command.Version)
		os.Exit(0)
	case "help", "--help":
		flag.Usage()
		os.Exit(0)
	}

	// 查找并执行命令
	cmd, ok := command.GetCommand(cmdName)
	if !ok {
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", cmdName)
		fmt.Fprintf(os.Stderr, "使用 '%s help' 查看可用命令\n", os.Args[0])
		os.Exit(1)
	}

	// 解析命令特定选项
	cmdFlags := cmd.Flags()
	if err := cmdFlags.Parse(cmdArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 获取剩余参数
	remainingArgs := cmdFlags.Args()

	// 如果第一个参数不是以 - 开头，作为路径参数
	var execArgs []string
	for _, arg := range remainingArgs {
		if !strings.HasPrefix(arg, "-") {
			execArgs = append(execArgs, arg)
		}
	}

	if len(execArgs) > 0 && (execArgs[0] == "help" || execArgs[0] == "--help") {
		fmt.Printf("\n=== %s ===\n\n", cmd.Name())
		fmt.Printf("Usage: %s\n\n", cmd.Usage())
		fmt.Printf("%s\n", cmd.Description())
		os.Exit(0)
	}

	// 执行命令
	if err := cmd.Execute(execArgs, globalOpts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
