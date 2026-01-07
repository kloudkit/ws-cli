package secrets

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

type cmdContext struct {
	cmd       *cobra.Command
	masterKey string
	force     bool
	dryRun    bool
	raw       bool
}

func newContext(cmd *cobra.Command) *cmdContext {
	return &cmdContext{
		cmd:       cmd,
		masterKey: getString(cmd, "master"),
		force:     getBool(cmd, "force"),
		dryRun:    getBool(cmd, "dry-run"),
		raw:       getBool(cmd, "raw"),
	}
}

func (c *cmdContext) resolveMasterKey() ([]byte, error) {
	return internalSecrets.ResolveMasterKey(c.masterKey)
}

func (c *cmdContext) print(msg string) {
	if c.raw {
		fmt.Fprintln(c.cmd.OutOrStdout(), msg)
	} else {
		fmt.Fprintln(c.cmd.OutOrStdout(), styles.Code().Render(msg))
	}
}

func (c *cmdContext) success(msg string) {
	if !c.raw {
		fmt.Fprintln(c.cmd.OutOrStdout(), styles.Success().Render(msg))
	}
}

func (c *cmdContext) dryRunMsg(msg string) {
	fmt.Fprintln(c.cmd.OutOrStdout(), styles.Warning().Render(msg))
}

func getString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func getBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func getInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}
