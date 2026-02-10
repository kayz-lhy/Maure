package maurecobra

import (
	"github.com/spf13/cobra"

	"maure/cmd/maure/command"
)

func addServeCommands(root *cobra.Command, opts *command.GlobalOptions) {
	root.AddCommand(newLegacyCommand(
		"serve [path]",
		"启动 HTTP API 服务",
		"启动 HTTP API 服务",
		"maure serve --port 8080",
		command.NewServeCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))
}
