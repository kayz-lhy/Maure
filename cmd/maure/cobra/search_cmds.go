package maurecobra

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"maure/cmd/maure/command"
)

func addSearchCommands(root *cobra.Command, opts *command.GlobalOptions, stderr io.Writer, rawArgs []string) {
	searchLegacy := command.NewSearchCommand()
	searchCmd := newLegacyCommand(
		"search <query>",
		"搜索文档",
		"搜索索引中的文档",
		"maure search --group=level \"error OR timeout\"\nmaure search \"error OR timeout\" --group=level (deprecated)",
		searchLegacy,
		opts,
		cobra.MinimumNArgs(1),
		func(args []string) error {
			if HasLegacyFlagOrder(rawArgs, "search") && stderr != nil {
				fmt.Fprintln(stderr, "Warning: legacy argument order is deprecated; prefer: maure search --group=level \"query\".")
			}
			return searchLegacy.Execute(args, *opts)
		},
	)
	root.AddCommand(searchCmd)

	countLegacy := command.NewCountCommand()
	countCmd := newLegacyCommand(
		"count <query>",
		"统计匹配文档数",
		"统计匹配查询的文档数量",
		"maure count \"error\"",
		countLegacy,
		opts,
		cobra.MinimumNArgs(1),
		func(args []string) error {
			if HasLegacyFlagOrder(rawArgs, "count") && stderr != nil {
				fmt.Fprintln(stderr, "Warning: legacy argument order is deprecated; prefer: maure count --index=./index \"query\".")
			}
			return countLegacy.Execute(args, *opts)
		},
	)
	root.AddCommand(countCmd)

	root.AddCommand(newLegacyCommand(
		"terms [path]",
		"列出词项",
		"列出索引中的所有词项",
		"maure terms --p=req",
		command.NewTermsCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))

	root.AddCommand(newLegacyCommand(
		"similarity [bm25|tfidf]",
		"设置评分算法",
		"查询或设置评分算法",
		"maure similarity bm25",
		command.NewSimilarityCommand(),
		opts,
		cobra.MaximumNArgs(1),
		nil,
	))
}
