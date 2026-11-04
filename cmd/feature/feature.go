package feature

import (
	"fmt"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

const scaffoldTemplate = `---
- name: Install %s
  gather_facts: false
  hosts: workspace

  tasks:
    - name: Say hello
      ansible.builtin.debug:
        msg: Hello world! 👋
`

var skippableSections = []struct{ flag, key, desc string }{
	{"skip-extensions", "skip_extensions", "Skip installing VSCode extensions"},
	{"skip-completion", "skip_completion", "Skip configuring shell completion"},
	{"skip-repository", "skip_repository", "Skip enabling the vendor APT repository"},
}

var installCmd = cli.New("install <name>", "0.2.0", "Install additional pre-configured features",
	"Run a feature's playbook to install it. Pass variables with --opt KEY=VAL, and skip parts you do not want with --skip-extensions, --skip-completion, or --skip-repository.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		featurePath, err := features.ResolveFeaturePath(featureDirs(cmd), args[0])
		if err != nil {
			return err
		}

		if _, err := features.ParseFeatureFile(featurePath); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		styles.PrintTitle(out, "Installing "+args[0])

		if err := features.RunPlaybook(featurePath, buildVars(cmd)); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		styles.PrintSuccess(out, "Feature installed", styles.Pair{Key: "Feature", Value: args[0]})

		return nil
	})

var listCmd = cli.New("list", "0.2.0", "List available features that can be installed",
	"List the features you can install, marking where each comes from — the shipped set, a workspace override, or your own ~/.ws/features.d.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		result, err := features.ListFeatures(featureDirs(cmd))
		if err != nil {
			return fmt.Errorf("failed to list features: %w", err)
		}

		for _, w := range result.Warnings {
			styles.PrintWarning(out, w)
		}

		if len(result.Features) == 0 {
			styles.PrintEmpty(out, "No features found")
			return nil
		}

		items := make([]styles.DescriptionItem, len(result.Features))
		for i, f := range result.Features {
			items[i] = styles.DescriptionItem{Name: f.Name, Description: f.Description}
			if f.Source != features.SourceSystem {
				items[i].Description += " " + styles.Aside(string(f.Source))
			}
		}

		styles.PrintTitle(out, "Features Available", len(items))
		styles.PrintList(out, items)
		styles.PrintHints(out,
			styles.Hint{Command: "ws feature install <name>", Description: "Install a feature"},
			styles.Hint{Command: "ws feature info <name>", Description: "View feature details"},
		)

		return nil
	})

var infoCmd = cli.New("info <name>", "0.2.0", "Show detailed information about a feature",
	"Show a feature's description and the variables it accepts, so you know what --opt values install will take.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		feature, err := features.InfoFeature(featureDirs(cmd), args[0])
		if err != nil {
			return fmt.Errorf("failed to get feature info: %w", err)
		}

		styles.PrintTitle(out, feature.Name)
		styles.PrintText(out, feature.Description)

		if len(feature.Vars) > 0 {
			fmt.Fprintln(out)
			styles.PrintSubHeader(out, "Options")
			styles.PrintBullets(out, feature.Vars)
		}

		return nil
	})

var storeCmd = cli.New("store", "0.2.0", "List packages available in the feature store",
	"List the artifacts published to the feature store (WS_FEATURES_STORE_URL) — the offline mirror features install from when the network is locked down.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		storeURL, _ := config.Resolve("features", "store_url")
		if storeURL == "" {
			styles.PrintWarning(out, "Feature store not configured (set WS_FEATURES_STORE_URL)")
			return nil
		}

		manifest, err := features.FetchStoreManifest(storeURL)
		if err != nil {
			return fmt.Errorf("failed to fetch store manifest: %w", err)
		}

		if len(manifest.Artifacts) == 0 {
			styles.PrintEmpty(out, "Feature store is empty")
			return nil
		}

		items := make([]styles.DescriptionItem, len(manifest.Artifacts))
		for i, a := range manifest.Artifacts {
			items[i] = styles.DescriptionItem{Name: a.Name, Description: a.Version}
		}

		styles.PrintTitle(out, "Feature Store", len(items))
		styles.PrintList(out, items)

		if manifest.Built != "" {
			fmt.Fprintln(out)
			styles.PrintMuted(out, "Built: "+manifest.Built)
		}

		return nil
	})

var newCmd = cli.New("new [name]", "0.2.0", "Print boilerplate for a custom feature playbook",
	`Print a starter feature playbook to stdout.

Redirect it into ~/.ws/features.d/<name>.yaml, then extend it and install
with "ws-cli feature install <name>":

  ws-cli feature new redis > ~/.ws/features.d/redis.yaml`,
	func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), scaffoldTemplate, append(args, "<feature>")[0])
		return nil
	})

var FeatureCmd = cli.Group("feature", "0.2.0", "Install additional pre-configured features",
	"Install and inspect optional workspace features — Ansible playbooks that add tools on top of the base image. Ships a curated set; --root points at your own under ~/.ws/features.d.",
	`# List the features you can install
ws feature list

# Install one, skipping its VSCode extensions
ws feature install redis --skip-extensions`,
	installCmd, listCmd, infoCmd, storeCmd, newCmd)

func featureDirs(cmd *cobra.Command) []string {
	root := cli.String(cmd, "root")
	if cmd.Flags().Changed("root") {
		return []string{root}
	}

	return []string{root, path.GetHomeDirectory(".ws", "features.d")}
}

func buildVars(cmd *cobra.Command) map[string]any {
	opts, _ := cmd.Flags().GetStringToString("opt")

	vars := make(map[string]any, len(opts)+len(skippableSections))
	for key, value := range opts {
		vars[key] = value
	}

	for _, section := range skippableSections {
		if cli.Bool(cmd, section.flag) {
			vars[section.key] = "true"
		}
	}

	return vars
}

func addInstallFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringToString("opt", map[string]string{}, "Optional variables to use during installation")

	for _, section := range skippableSections {
		cmd.Flags().Bool(section.flag, false, section.desc)
	}
}

func init() {
	root, _ := config.Resolve("features", "dir")

	FeatureCmd.PersistentFlags().String("root", root, "Root directory of additional features")
	addInstallFlags(installCmd)

	installCmd.Args, infoCmd.Args, newCmd.Args = cobra.ExactArgs(1), cobra.ExactArgs(1), cobra.MaximumNArgs(1)
	listCmd.Aliases = []string{"ls"}
}
