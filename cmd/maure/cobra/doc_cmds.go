package maurecobra

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"maure/cmd/maure/command"
)

func addDocumentCommands(root *cobra.Command, opts *command.GlobalOptions, stderr io.Writer) {
	root.AddCommand(newLegacyCommand(
		"add <file>",
		"添加文件到索引",
		"添加单个文件到索引",
		"maure add ./docs/file.txt",
		command.NewAddCommand(),
		opts,
		cobra.ExactArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"add-dir <dir>",
		"批量添加目录",
		"批量添加目录下所有支持的文件到索引",
		"maure add-dir ./docs",
		command.NewAddDirCommand(),
		opts,
		cobra.ExactArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"import <file>",
		"从 JSON 导入文档",
		"从 JSON 文件批量导入文档",
		"maure import ./docs.json",
		command.NewImportCommand(),
		opts,
		cobra.ExactArgs(1),
		nil,
	))

	parseLogLegacy := command.NewParseLogCommand()
	parseLogCmd := newLegacyCommand(
		"parse-log <file>",
		"解析日志并导入索引",
		"支持 JSON/Logback/auto 三种格式解析",
		"maure parse-log --log-format=json ./logs/app.log\nmaure parse-log ./logs/app.log --format=json (deprecated)",
		parseLogLegacy,
		opts,
		cobra.ExactArgs(1),
		nil,
	)
	parseLogCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if changed := cmd.Flags().Changed("log-format"); !changed {
			if legacyChanged := cmd.Flags().Changed("format"); legacyChanged {
				value, err := cmd.Flags().GetString("format")
				if err == nil {
					_ = cmd.Flags().Set("log-format", value)
				}
				if stderr != nil {
					fmt.Fprintln(stderr, "Warning: parse-log flag --format is deprecated; use --log-format.")
				}
			}
		}
	}
	parseLogCmd.Flags().String("format", "", "(deprecated) 日志格式，请改用 --log-format")
	root.AddCommand(parseLogCmd)

	root.AddCommand(newLegacyCommand(
		"delete-doc <doc-id>",
		"删除文档",
		"根据文档 ID 删除文档",
		"maure delete-doc 1",
		command.NewDeleteDocCommand(),
		opts,
		cobra.ExactArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"list [path]",
		"列出所有文档",
		"列出索引中的所有文档",
		"maure list --index ./myindex",
		command.NewListCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))
}
