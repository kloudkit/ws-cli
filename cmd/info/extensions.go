package info

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"os/exec"
)

func fetchExtensions() string {
	out, _ := exec.Command("code", "--list-extensions", "--show-versions").Output()

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(out))
	skipHeader := true

	for scanner.Scan() {
		if skipHeader {
			skipHeader = false
			continue
		}

		buf.WriteString("\t\t ")
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
	}

	return buf.String()
}

var extensionsCmd = &cobra.Command{
	Use:   "extensions",
	Short: "Display installed extensions",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStdout(), fetchExtensions())
	},
}

func init() {
	InfoCmd.AddCommand(extensionsCmd)
}
