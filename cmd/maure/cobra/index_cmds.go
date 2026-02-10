package maurecobra

import (
	"github.com/spf13/cobra"

	"maure/cmd/maure/command"
)

func addIndexCommands(root *cobra.Command, opts *command.GlobalOptions) {
	root.AddCommand(newLegacyCommand(
		"init <path>",
		"初始化索引目录",
		"创建新的索引目录",
		"maure init ./myindex",
		command.NewInitCommand(),
		opts,
		cobra.ExactArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"open [path]",
		"打开索引目录",
		"打开现有的索引目录",
		"maure open ./myindex",
		command.NewOpenCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"info [path]",
		"显示索引信息",
		"显示索引的基本信息",
		"maure info ./myindex",
		command.NewInfoCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"stats [path]",
		"显示索引统计",
		"显示索引的详细统计信息",
		"maure stats ./myindex",
		command.NewStatsCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"compact [path]",
		"优化/压缩索引",
		"优化索引结构，回收空间",
		"maure compact ./myindex",
		command.NewCompactCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"delete <path>",
		"删除索引目录",
		"删除整个索引目录",
		"maure delete ./myindex",
		command.NewDeleteCommand(),
		opts,
		cobra.ExactArgs(1),
		nil,
	))
}
