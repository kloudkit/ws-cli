package port

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/port"
	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

const openNotice = "sign-in is off in this workspace, so every forwarded port is already open to anyone who can reach it"

func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

func PrintShared(out io.Writer, shared port.Shared) {
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

func PrintEnded(out io.Writer, port int, ended *port.Ended) {
	if ended == nil {
		return
	}

	fmt.Fprintln(out)
	styles.PrintWarning(out, fmt.Sprintf("Port %d is private again: unshared by %s", port, ended.By))
}

var shareCmd = cli.New("share <port>", "next", "Make a forwarded port public until you stop",
	"Make a port answer on its proxy-domain host without the workspace sign-in, and keep it public while this command runs. Prints the public URLs; Ctrl+C, or `ws-cli port unshare` from any terminal, makes the port private again. Needs WS_SERVER_PROXY_ALLOW_PUBLIC and WS_SERVER_PROXY_DOMAIN. The /proxy/<port>/ path always needs sign-in.",
	func(cmd *cobra.Command, args []string) error {
		number, err := port.ParsePort(args[0])
		if err != nil {
			return err
		}

		ctx, stop := SignalContext()
		defer stop()

		out := cmd.OutOrStdout()

		ended, err := port.Hold(ctx, number, func(shared port.Shared) {
			PrintShared(out, shared)
			server.PrintStop(out)
		})
		if err != nil {
			return err
		}

		PrintEnded(out, number, ended)

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

		styles.PrintSuccess(out, fmt.Sprintf("Closed %d unused %s", len(closed), plural(len(closed), "port", "ports")),
			styles.Pair{Key: "Ports", Value: strings.Join(closed, ", ")})

		return nil
	})

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}

	return many
}

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

var errNoPort = errors.New("name the port, e.g. ws-cli port share 8000")

func onePort(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errNoPort
	}

	return nil
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
	shareCmd.Args = onePort
	unshareCmd.Args = onePort
	listCmd.Args = cobra.NoArgs
	clearCmd.Args = cobra.NoArgs
}
