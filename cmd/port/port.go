package port

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/port"
	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

const openNotice = "sign-in is off in this workspace, so every forwarded port is already open to anyone who can reach it"

func PrintShared(out io.Writer, shared port.Event) {
	styles.PrintTitle(out, "Public Port")

	pairs := []styles.Pair{{Key: "Port", Value: strconv.Itoa(shared.Port)}, {Key: "Access", Value: shared.Access}}
	for _, url := range shared.URLs {
		pairs = append(pairs, styles.Pair{Key: "URL", Value: url})
	}

	styles.PrintKeyValues(out, pairs...)

	if !shared.LoginRequired {
		fmt.Fprintln(out)
		styles.PrintWarning(out, openNotice)
	}
}

func PrintEnded(out io.Writer, port int, by string) {
	if by == "" {
		return
	}

	fmt.Fprintln(out)
	styles.PrintWarning(out, fmt.Sprintf("Port %d is private again: unshared by %s", port, by))
}

var shareCmd = cli.New("share <port>", "next", "Make a forwarded port public until you stop",
	"Make a port answer on its proxy-domain host without the workspace sign-in, and keep it public while this command runs. Prints the public URLs; Ctrl+C, or `ws-cli port unshare` from any terminal, makes the port private again. Needs WS_SERVER_PROXY_ALLOW_PUBLIC and WS_SERVER_PROXY_DOMAIN. The /proxy/<port>/ path always needs sign-in.",
	func(cmd *cobra.Command, args []string) error {
		number, err := port.ParsePort(args[0])
		if err != nil {
			return err
		}

		ctx, stop := cli.SignalContext()
		defer stop()

		out := cmd.OutOrStdout()

		by, err := port.Hold(ctx, number, func(shared port.Event) {
			PrintShared(out, shared)
			server.PrintStop(out)
		})
		if err != nil {
			return err
		}

		PrintEnded(out, number, by)

		return nil
	})

var unshareCmd = cli.New("unshare <port>", "next", "Make a public port private",
	"End the share of a public port, whoever holds it: a `ws-cli port share` in another terminal exits, and a share set in the editor's Ports view goes back to private.",
	func(cmd *cobra.Command, args []string) error {
		number, err := port.ParsePort(args[0])
		if err != nil {
			return err
		}

		if err := port.Unshare(number); err != nil {
			return err
		}

		styles.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Port %d is private", number))

		return nil
	})

var listCmd = cli.MCP(cli.New("list", "next", "List the listening ports and which are public",
	"List the TCP ports listening in the workspace, whether each one is public or private, its public URLs, and its state in the editor's Ports tab.",
	func(cmd *cobra.Command, args []string) error {
		listeners, err := port.Listeners()
		if err != nil {
			return err
		}

		listing, err := port.List()
		if err != nil {
			return err
		}

		tab, tabErr := port.Tab()

		renderList(cmd.OutOrStdout(), listeners, listing, tab, tabErr)

		return nil
	}), cli.MCPRead)

var clearCmd = cli.New("clear", "next", "Close the unused ports in the editor's Ports tab",
	"Close the forwarded ports in the editor's Ports tab whose process has exited. A port with a live process stays.",
	func(cmd *cobra.Command, args []string) error {
		cleared, err := port.Clear()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(cleared.Closed) == 0 {
			styles.PrintEmpty(out, "No unused ports to close")
			return nil
		}

		closed := make([]string, len(cleared.Closed))
		for i, number := range cleared.Closed {
			closed[i] = strconv.Itoa(number)
		}

		styles.PrintSuccess(out, "Closed the unused ports", styles.Pair{Key: "Ports", Value: strings.Join(closed, ", ")})

		return nil
	})

func renderList(out io.Writer, listeners []port.Listener, listing *port.Listing, tab []port.TabPort, tabErr error) {
	shares := map[int]port.Share{}
	for _, share := range listing.Shares {
		shares[share.Port] = share
	}

	tabs := map[int]port.TabPort{}
	for _, entry := range tab {
		tabs[entry.Port] = entry
	}

	type row struct {
		port      int
		addresses string
	}

	rows := make([]row, 0, len(listeners))
	seen := map[int]bool{}
	for _, listener := range listeners {
		rows = append(rows, row{listener.Port, strings.Join(listener.Addresses, " ")})
		seen[listener.Port] = true
	}

	for _, share := range listing.Shares {
		if !seen[share.Port] {
			rows = append(rows, row{share.Port, "not listening"})
		}
	}

	table := styles.NewTable("Port", "Address", "Visibility", "URL", "Ports Tab").Title("Ports")

	for _, r := range rows {
		visibility, url := "private", ""
		if share, ok := shares[r.port]; ok {
			visibility = fmt.Sprintf("%s (%s)", share.Access, share.Source)
			url = strings.Join(share.URLs, " ")
		}

		table.Rows([]string{strconv.Itoa(r.port), r.addresses, visibility, url, tabState(tabs, r.port, tabErr)})
	}

	table.Print(out)

	if !listing.Enabled {
		fmt.Fprintln(out)
		styles.PrintMuted(out, "Public ports are off: set WS_SERVER_PROXY_ALLOW_PUBLIC to allow them")
	} else if !listing.LoginRequired {
		fmt.Fprintln(out)
		styles.PrintWarning(out, openNotice)
	}

	if tabErr != nil {
		fmt.Fprintln(out)
		styles.PrintMuted(out, "Ports tab unknown: "+tabErr.Error())
	}
}

func tabState(tabs map[int]port.TabPort, number int, tabErr error) string {
	if tabErr != nil {
		return "unknown"
	}

	entry, ok := tabs[number]
	if !ok {
		return "-"
	}

	state := "detected"
	if entry.Forwarded {
		state = "forwarded, " + entry.Privacy
	}

	if entry.Running != nil && !*entry.Running {
		state += ", unused"
	}

	return state
}

var PortCmd = cli.Group("port", "next", "Share and list the workspace's ports",
	"Make a forwarded port public on its proxy-domain host without the workspace sign-in, list the listening ports and which are public, and tidy the editor's Ports tab.",
	`# Share a dev server until Ctrl+C
ws-cli port share 8000

# End a share from any terminal
ws-cli port unshare 8000

# See what is listening and what is public
ws-cli port list

# Close the unused ports in the editor's Ports tab
ws-cli port clear`,
	shareCmd, unshareCmd, listCmd, clearCmd)

func init() {
	shareCmd.Args = cobra.ExactArgs(1)
	unshareCmd.Args = cobra.ExactArgs(1)
	listCmd.Args = cobra.NoArgs
	clearCmd.Args = cobra.NoArgs
}
