package maurecobra

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"maure/cmd/maure/command"
)

type Config struct {
	RawArgs []string
	Stdout  io.Writer
	Stderr  io.Writer
}

func NewRootCommand(cfg Config) *cobra.Command {
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	stderr := cfg.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	opts := &command.GlobalOptions{
		IndexPath: ".",
		Format:    "text",
		Verbose:   false,
		Analyzer:  "standard",
	}

	rootCmd := &cobra.Command{
		Use:           "maure",
		Short:         "Maure Search Engine CLI",
		Long:          fmt.Sprintf("Maure Search Engine v%s", command.Version),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.IndexPath == "" {
				return fmt.Errorf("索引目录路径不能为空")
			}
			if opts.Format != "text" && opts.Format != "json" {
				return fmt.Errorf("无效输出格式: %s (支持 text/json)", opts.Format)
			}
			switch opts.Analyzer {
			case "standard", "ram":
				return nil
			default:
				return fmt.Errorf("无效分析器: %s (支持 standard/ram)", opts.Analyzer)
			}
		},
	}

	rootCmd.Version = command.Version
	rootCmd.SetVersionTemplate("Maure Search Engine v{{.Version}}\n")
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.PersistentFlags().StringVar(&opts.IndexPath, "index", ".", "索引目录路径")
	rootCmd.PersistentFlags().StringVar(&opts.Format, "format", "text", "输出格式 (text/json)")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().StringVar(&opts.Analyzer, "analyzer", "standard", "分析器类型")

	rootCmd.AddCommand(newVersionCommand())
	addIndexCommands(rootCmd, opts)
	addDocumentCommands(rootCmd, opts, stderr)
	addSearchCommands(rootCmd, opts, stderr, cfg.RawArgs)
	addServeCommands(rootCmd, opts)

	return rootCmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "显示版本",
		Aliases: []string{"--version"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Maure Search Engine v%s\n", command.Version)
		},
	}
}
