package styles

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const forceColorEnv = "WS__INTERNAL_FORCE_COLOR"

func colorEnv() []string {
	environ := os.Environ()

	if os.Getenv(forceColorEnv) != "" && os.Getenv("NO_COLOR") == "" {
		environ = append(environ, "CLICOLOR_FORCE=1", "COLORTERM=truecolor")
	}

	return environ
}

func NewWriter(w io.Writer) io.Writer {
	return colorprofile.NewWriter(w, colorEnv())
}

func WrapOutput(cmd *cobra.Command) {
	if cmd.OutOrStdout() == os.Stdout {
		cmd.SetOut(NewWriter(os.Stdout))
	}

	if cmd.ErrOrStderr() == os.Stderr {
		cmd.SetErr(NewWriter(os.Stderr))
	}
}

func TerminalWidth() int {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return 0
	}

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}

	return width
}
