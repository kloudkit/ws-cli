package seed

import (
	"fmt"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/seed"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

func command(use, short, long string, run func(cmd *cobra.Command, source string, args []string) error) *cobra.Command {
	cmd := cli.New(use, "0.5.0", short, long, func(cmd *cobra.Command, args []string) error {
		resolved, err := seed.ResolveSource(cli.String(cmd, "source"), cmd.Flags().Changed("source"))
		if err != nil {
			return err
		}

		return run(cmd, resolved, args)
	})
	cmd.SilenceUsage = true

	return cmd
}

var applyCmd = command("apply [dest...]", "Project seed content onto the filesystem",
	"Apply the seed source to the filesystem: mirror bare files and run the .seed.yaml operations. Writes only where the destination is missing unless --force; pass destinations to limit the run to those paths.",
	func(cmd *cobra.Command, source string, args []string) error {
		return seed.Apply(seed.Options{Source: source, Force: cli.Bool(cmd, "force"), Dests: args, MasterKey: cli.String(cmd, "master"), Out: cmd.OutOrStdout()})
	})

var lsCmd = command("ls", "List seed destinations and their behaviors",
	"List what apply would write, without touching the filesystem: each destination with its operation and whether it carries a secret or a template.",
	func(cmd *cobra.Command, source string, args []string) error {
		out := cmd.OutOrStdout()

		plan, err := seed.BuildPlan(source, false)
		if err != nil {
			return err
		}

		if len(plan.Ops) == 0 {
			styles.PrintEmpty(out, fmt.Sprintf("No seed entries in [%s]", source))
			return nil
		}

		items := make([]styles.DescriptionItem, len(plan.Ops))
		for i, op := range plan.Ops {
			items[i] = styles.DescriptionItem{Name: op.Dest, Description: describe(op)}
		}

		styles.PrintTitle(out, "Seed Entries", len(items))
		styles.PrintList(out, items)

		return nil
	})

var rotateCmd = command("rotate", "Re-encrypt managed secrets under a new master key",
	"Re-encrypt every managed secret from the old master key (--master) to a new one (--new-master), in place. All-or-nothing: it verifies every secret decrypts before writing anything.",
	func(cmd *cobra.Command, source string, args []string) error {
		return seed.Rotate(seed.RotateOptions{Source: source, MasterKey: cli.String(cmd, "master"), NewMasterKey: cli.String(cmd, "new-master"), Out: cmd.OutOrStdout()})
	})

var SeedCmd = cli.Group("seed", "0.5.0", "Project declarative content onto the filesystem",
	"Copy files and apply small edits from a seed source onto the filesystem at boot. Bare files mirror verbatim; a .seed.yaml manifest overlays behavior (copy, merge, append) and decrypts secrets under the master key. Point --source at a mounted volume to seed a container from durable storage.",
	`# Preview what apply would write
ws-cli seed ls --source /mnt/seed

# Apply it, overwriting existing destinations
ws-cli seed apply --source /mnt/seed --force`,
	applyCmd, lsCmd, rotateCmd)

func describe(op seed.ResolvedOp) string {
	parts := []string{string(op.Op)}

	if op.Secret {
		parts = append(parts, styles.Warning.Render("secret"))
	}

	if op.Template {
		parts = append(parts, styles.Info.Render("template"))
	}

	return strings.Join(parts, " ")
}

func init() {
	source, _ := config.Resolve("seed", "source")

	SeedCmd.PersistentFlags().String("source", source, "Seed source directory")
	applyCmd.Flags().Bool("force", false, "Overwrite existing destinations")
	applyCmd.Flags().String("master", "", "Master key or path to key file")
	rotateCmd.Flags().String("master", "", "Current master key or path to key file")
	rotateCmd.Flags().String("new-master", "", "New master key or path to key file")
}
