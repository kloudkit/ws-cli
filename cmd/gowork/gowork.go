package gowork

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var GoworkCmd = &cobra.Command{
	Use:     "gowork",
	Aliases: []string{"clip"},
	Short:   "Interact with the native clipboard",
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create go.work file/Update go version in go.work file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/workspace/go.work"
		goVersion := runtime.Version()
		version := strings.Replace(goVersion, "go", "", -1)

		// Read go.work
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error reading file: %w", err)
		}

		// If file doesn't exist — create it
		if os.IsNotExist(err) {
			content := fmt.Sprintf(`go %s

toolchain %s

use ()
`, version, goVersion)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("error creating file: %w", err)
			}
			fmt.Println("go.work file created with version:", goVersion)
			return nil
		}

		// update Go version
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "go ") {
				lines[i] = "go " + version
			}
			if strings.HasPrefix(line, "toolchain ") {
				lines[i] = "toolchain " + goVersion
			}
		}
		newContent := strings.Join(lines, "\n")

		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("error updating file: %w", err)
		}
		fmt.Println("go.work file updated to version:", goVersion)
		return nil
	},
}

var addProjectCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Add a project to go.work",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pathToAdd := args[0]
		filePath := "/workspace/go.work"
		goVersion := runtime.Version()
		version := strings.Replace(goVersion, "go", "", -1)

		data, err := os.ReadFile(filePath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error reading go.work: %w", err)
		}

		var lines []string
		if os.IsNotExist(err) {
			// create new file with project
			lines = []string{
				"go " + version,
				"",
				"toolchain " + goVersion,
				"",
				"use (",
				fmt.Sprintf("\t%s", pathToAdd),
				")",
			}
		} else {
			lines = strings.Split(string(data), "\n")

			// locate use block
			useStart := -1
			useEnd := -1
			for i, line := range lines {
				if strings.HasPrefix(line, "use (") {
					useStart = i
				}
				if useStart != -1 && strings.HasPrefix(line, ")") {
					useEnd = i
					break
				}
			}

			if useStart != -1 && useEnd != -1 {
				// check if already exists
				for i := useStart + 1; i < useEnd; i++ {
					if strings.TrimSpace(lines[i]) == pathToAdd {
						fmt.Println("Project already exists in go.work:", pathToAdd)
						return nil
					}
				}
				// add before closing parenthesis
				lines = append(lines[:useEnd], append([]string{fmt.Sprintf("\t%s", pathToAdd)}, lines[useEnd:]...)...)
			} else {
				// add new use block
				lines = append(lines, "use (", fmt.Sprintf("\t%s", pathToAdd), ")")
			}
		}

		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("error writing to go.work: %w", err)
		}

		fmt.Println("Project added to go.work:", pathToAdd)
		return nil
	},
}

func init() {
	GoworkCmd.AddCommand(createCmd)
	GoworkCmd.AddCommand(addProjectCmd)
}
