package config

import (
	"fmt"
	"path/filepath"

	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/spf13/cobra"
)

type Config struct {
	SourcePath string
	OutputName string
}

var configs = map[string]Config{
	"markdownlint": {
		SourcePath: ".config/markdownlint/config",
		OutputName: ".markdownlint.json",
	},
  "ruff": {
    SourcePath: ".config/ruff/ruff.toml",
    OutputName: ".ruff.toml",
  },
  "yamllint`": {
    SourcePath: ".config/yamllint/config",
    OutputName: ".yamllint",
  },
}

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Interact with workspace related configurations",
}

var copyCmd = &cobra.Command{
	Use:   "cp",
	Short: "Copying workspace defined configurations to a project",
}

func createCommand(key string) *cobra.Command {
  config := configs[key]

  return &cobra.Command{
    Use:   key,
    Short: fmt.Sprintf("Copy the %s configuration to the project", key),
    Run: func(cmd *cobra.Command, args []string) {
      dest, _ := cmd.Flags().GetString("dest")

  		source := path.GetHomeDirectory(config.SourcePath)

      dest, _ = filepath.Abs(dest)

      fmt.Println("Source path:", source)
      fmt.Println("Absolute path:", dest + "/" + config.OutputName)
    },
  }
}

func init() {
	copyCmd.PersistentFlags().String("dest", ".", "Output directory")
	copyCmd.PersistentFlags().BoolP("force", "f", false, "Force the overwriting of an existing file")

  copyCmd.AddCommand(
    createCommand("markdownlint"),
    createCommand("ruff"),
    createCommand("yamllint"),
  )

	ConfigCmd.AddCommand(copyCmd)
}
