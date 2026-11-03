package clitest

import (
	"bytes"
	"strings"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"github.com/spf13/cobra"
)

func Run(cmd *cobra.Command, stdin string, args ...string) (string, error) {
	testutil.ResetCommandFlags(cmd)

	var buffer bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&buffer)
	cmd.SetErr(&buffer)
	cmd.SetArgs(args)
	err := cmd.Execute()

	return testutil.StripAnsi(buffer.String()), err
}
