package seed

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/spf13/cobra"
)

var SeedCmd = &cobra.Command{
	Use:         "seed",
	Short:       "Project declarative content onto the filesystem",
	Long:        "Copy files and apply small edits from a seed source onto the filesystem at boot. Bare files mirror verbatim; a .seed.yaml manifest overlays behavior — copy, merge, append — and decrypts secrets under the master key. Point --source at a mounted volume to seed a container from durable storage.",
	Annotations: map[string]string{"since": "next"},
	Example: `# Preview what apply would write
ws seed ls --source /mnt/seed

# Apply it, overwriting existing destinations
ws seed apply --source /mnt/seed --force`,
}

func init() {
	source, _ := config.Resolve("seed", "source")

	SeedCmd.PersistentFlags().String("source", source, "Seed source directory")
}
