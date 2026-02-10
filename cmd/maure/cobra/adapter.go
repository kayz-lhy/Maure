package maurecobra

import (
	"flag"
	"io"

	"github.com/spf13/cobra"

	"maure/cmd/maure/command"
)

func attachLegacyFlags(cmd *cobra.Command, legacy command.Command) {
	legacyFlags := legacy.Flags()
	legacyFlags.SetOutput(io.Discard)
	legacyFlags.VisitAll(func(f *flag.Flag) {
		// no-op: ensure flags are initialized before binding.
		_ = f
	})
	cmd.Flags().AddGoFlagSet(legacyFlags)
}

func newLegacyCommand(use, short, long, example string, legacy command.Command, opts *command.GlobalOptions, argsValidator cobra.PositionalArgs, run func(args []string) error) *cobra.Command {
	c := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    argsValidator,
		RunE: func(cmd *cobra.Command, args []string) error {
			if run != nil {
				return run(args)
			}
			return legacy.Execute(args, *opts)
		},
	}
	attachLegacyFlags(c, legacy)
	return c
}
