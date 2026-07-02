package feature

import (
	"fmt"

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

var newCmd = &cobra.Command{
	Use:         "new [name]",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Print boilerplate for a custom feature playbook",
	Long: `Print a starter feature playbook to stdout.

Redirect it into ~/.ws/features.d/<name>.yaml, then extend it and install
with "ws-cli feature install <name>":

  ws-cli feature new redis > ~/.ws/features.d/redis.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "<feature>"
		if len(args) == 1 {
			name = args[0]
		}

		fmt.Fprintf(cmd.OutOrStdout(), scaffoldTemplate, name)

		return nil
	},
}
