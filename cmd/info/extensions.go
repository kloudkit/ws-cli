package info

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
	"io"
	"os/exec"
	"strings"
)

func fetchExtensions() [][]string {
	out, _ := exec.Command("code", "--list-extensions", "--show-versions").Output()

	var extensions [][]string
	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "@")

		if len(parts) == 2 {
			extensions = append(extensions, []string{parts[0], parts[1]})
		}
	}

	return extensions
}

func showExtensions(writer io.Writer) {
	extensions := fetchExtensions()

	fmt.Fprintf(writer, "%s\n", styles.TitleWithCount("Extensions", len(extensions)))

	fmt.Fprintln(writer)

	t := styles.Table("Name", "Version").
		Rows(extensions...)

	fmt.Fprintln(writer, t.Render())
}

var extensionsCmd = &cobra.Command{
	Use:   "extensions",
	Short: "Display installed extensions",
	RunE: func(cmd *cobra.Command, args []string) error {
		showExtensions(cmd.OutOrStdout())
		return nil
	},
}

func init() {
	InfoCmd.AddCommand(extensionsCmd)
}
