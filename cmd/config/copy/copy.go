package copy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/spf13/cobra"
)

type config struct {
	SourcePath string
	OutputName string
}

var configs = map[string]config{
	"markdownlint": {
		SourcePath: ".config/markdownlint/config",
		OutputName: ".markdownlint.json",
	},
	"ruff": {
		SourcePath: ".config/ruff/ruff.toml",
		OutputName: ".ruff.toml",
	},
	"yamllint": {
		SourcePath: ".config/yamllint/config",
		OutputName: ".yamllint",
	},
}

var CopyCmd = &cobra.Command{
	Use:   "cp",
	Short: "Copying workspace defined configurations to a project",
}

func copy(source, dest string) error {
	stats, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !stats.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", source)
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)

	return err
}

func createCommand(key string) *cobra.Command {
	config := configs[key]

	return &cobra.Command{
		Use:   key,
		Short: fmt.Sprintf("Copy %s configuration to the project", key),
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, _ := cmd.Flags().GetString("dest")
			force, _ := cmd.Flags().GetBool("force")

			source := path.GetHomeDirectory(config.SourcePath)

			dest, _ = filepath.Abs(dest)
			dest = path.AppendSegments(dest, config.OutputName)

			if !path.CanOverride(dest, force) {
				return fmt.Errorf("the file [%s] already exists", dest)
			}

			if err := copy(source, dest); err != nil {
				return fmt.Errorf("the file [%s] could not be written", dest)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Copied [%s] to [%s]\n", source, dest)

			return nil
		},
	}
}

func init() {
	CopyCmd.PersistentFlags().String("dest", ".", "Output directory")
	CopyCmd.PersistentFlags().BoolP("force", "f", false, "Force the overwriting of an existing file")

	CopyCmd.AddCommand(
		createCommand("markdownlint"),
		createCommand("ruff"),
		createCommand("yamllint"),
	)
}
