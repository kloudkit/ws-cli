package info

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/config"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
)

type Manifest struct {
	Version string `json:"version"`
	VSCode  struct {
		Version string `json:"version"`
	} `json:"vscode"`
}

func readManifest() (*Manifest, error) {
	if !internalIO.FileExists(config.DefaultManifestPath) {
		return nil, fmt.Errorf("manifest not found at %s", config.DefaultManifestPath)
	}

	data, err := os.ReadFile(config.DefaultManifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &m, nil
}

func showVersion(writer io.Writer) {
	manifest, err := readManifest()
	if err != nil {
		styles.PrintWarning(writer, fmt.Sprintf("Could not read workspace version: %v", err))
		fmt.Fprintf(writer, "%s\n", styles.Title().Render("Versions"))
		t := styles.Table().Rows(
			[]string{"ws-cli", Version},
		)
		fmt.Fprintln(writer, t.Render())
		return
	}

	fmt.Fprintf(writer, "%s\n", styles.Title().Render("Versions"))

	t := styles.Table().Rows(
		[]string{"workspace", manifest.Version},
		[]string{"ws-cli", Version},
		[]string{"VSCode", manifest.VSCode.Version},
	)

	fmt.Fprintln(writer, t.Render())
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
}

var showVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display installed workspace version",
	Run: func(cmd *cobra.Command, args []string) {
		if all, _ := cmd.Flags().GetBool("all"); all {
			showVersion(cmd.OutOrStdout())
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
		}
	},
}

func init() {
	showVersionCmd.Flags().Bool("all", false, "Show all version information")

	InfoCmd.AddCommand(showVersionCmd)
}
