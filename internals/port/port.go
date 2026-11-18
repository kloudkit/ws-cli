package port

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	wsnet "github.com/kloudkit/ws-cli/internals/net"
)

var ProcNetFiles = []string{"/proc/net/tcp", "/proc/net/tcp6"}

const listenState = "0A"

type Listener struct {
	Port      int
	Addresses []string
}

type Source struct {
	Kind string `json:"kind"`
	Pid  int    `json:"pid,omitempty"`
}

func (s Source) String() string {
	if s.Kind == "cli" {
		return fmt.Sprintf("ws-cli pid %d", s.Pid)
	}

	return "the editor"
}

type Share struct {
	Port   int      `json:"port"`
	Access string   `json:"access"`
	Source Source   `json:"source"`
	Since  string   `json:"since"`
	URLs   []string `json:"urls"`
}

type Listing struct {
	Revision      int     `json:"revision"`
	Enabled       bool    `json:"enabled"`
	LoginRequired bool    `json:"loginRequired"`
	Shares        []Share `json:"shares"`
}

type Event struct {
	Share
	Event         string `json:"event"`
	LoginRequired bool   `json:"loginRequired"`
	By            string `json:"by"`
}

type TabPort struct {
	Port      int    `json:"port"`
	Privacy   string `json:"privacy"`
	Forwarded bool   `json:"forwarded"`
	Running   *bool  `json:"running"`
}

type Cleared struct {
	Closed []int `json:"closed"`
}

func Self() Source {
	return Source{Kind: "cli", Pid: os.Getpid()}
}

func ParsePort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q: expected a number from 1 to 65535", raw)
	}

	return port, nil
}

func Listeners(files ...string) ([]Listener, error) {
	if len(files) == 0 {
		files = ProcNetFiles
	}

	byPort := map[int][]string{}
	read := 0

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		read++

		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 4 || fields[3] != listenState {
				continue
			}

			address, port, ok := decodeAddress(fields[1])
			if ok && !slices.Contains(byPort[port], address) {
				byPort[port] = append(byPort[port], address)
			}
		}
	}

	if read == 0 {
		return nil, errors.New("cannot read the listening ports from /proc/net/tcp or /proc/net/tcp6")
	}

	listeners := make([]Listener, 0, len(byPort))
	for port, addresses := range byPort {
		slices.Sort(addresses)
		listeners = append(listeners, Listener{Port: port, Addresses: addresses})
	}

	slices.SortFunc(listeners, func(a, b Listener) int { return a.Port - b.Port })

	return listeners, nil
}

func decodeAddress(raw string) (string, int, bool) {
	hostHex, portHex, found := strings.Cut(raw, ":")
	if !found {
		return "", 0, false
	}

	port, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return "", 0, false
	}

	bytes, err := hex.DecodeString(hostHex)
	if err != nil || (len(bytes) != net.IPv4len && len(bytes) != net.IPv6len) {
		return "", 0, false
	}

	for word := 0; word < len(bytes); word += 4 {
		slices.Reverse(bytes[word : word+4])
	}

	return net.IP(bytes).String(), int(port), true
}

func List() (*Listing, error) {
	body, err := wsnet.SessionCall(http.MethodGet, "/ports", nil)
	if err != nil {
		return nil, err
	}

	var listing Listing
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("error reading the public ports: %w", err)
	}

	return &listing, nil
}

func Unshare(port int) error {
	_, err := wsnet.SessionCall(http.MethodPost, "/ports/unshare", map[string]any{"port": port, "source": Self()})
	return err
}

func Hold(ctx context.Context, port int, onShared func(Event)) (string, error) {
	body, err := wsnet.SessionStream("/ports/share", map[string]any{"port": port, "access": "public", "source": Self()})
	if err != nil {
		return "", err
	}
	defer body.Close()

	stop := context.AfterFunc(ctx, func() { body.Close() })
	defer stop()

	decoder := json.NewDecoder(body)
	shared := false
	for {
		var next Event
		if err := decoder.Decode(&next); err != nil {
			switch {
			case ctx.Err() != nil:
				return "", nil
			case shared:
				return "the workspace server", nil
			case errors.Is(err, io.EOF):
				return "", fmt.Errorf("the workspace server ended the share of port %d before confirming it", port)
			default:
				return "", fmt.Errorf("lost the workspace server while sharing port %d: %w", port, err)
			}
		}

		switch next.Event {
		case "shared":
			shared = true
			onShared(next)
		case "unshared":
			return next.By, nil
		}
	}
}

func Tab() ([]TabPort, error) {
	body, err := wsnet.SendEnvelope("", map[string]any{"type": "portList"})
	if err != nil {
		return nil, err
	}

	var ports []TabPort
	if err := json.Unmarshal(body, &ports); err != nil {
		return nil, fmt.Errorf("error reading the editor's Ports tab: %w", err)
	}

	return ports, nil
}

func Clear() (*Cleared, error) {
	body, err := wsnet.SendEnvelope("", map[string]any{"type": "portClear"})
	if err != nil {
		return nil, err
	}

	var cleared Cleared
	if err := json.Unmarshal(body, &cleared); err != nil {
		return nil, fmt.Errorf("error reading the editor's answer: %w", err)
	}

	return &cleared, nil
}
