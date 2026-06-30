package seed

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/spf13/cobra"
)

var SeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Project declarative content onto the filesystem",
}

func init() {
	source, _ := config.Resolve("seed", "source")

	SeedCmd.PersistentFlags().String("source", source, "Seed source directory")
}
