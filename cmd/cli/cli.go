package cli

import "github.com/spf13/cobra"

type Run func(cmd *cobra.Command, args []string) error

func New(use, since, short, long string, run Run) *cobra.Command {
	return &cobra.Command{Use: use, Annotations: map[string]string{"since": since}, Short: short, Long: long, RunE: run}
}

func Group(use, since, short, long, example string, children ...*cobra.Command) *cobra.Command {
	cmd := New(use, since, short, long, nil)
	cmd.Example = example
	cmd.AddCommand(children...)

	return cmd
}

func Bool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func String(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func Int(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}
