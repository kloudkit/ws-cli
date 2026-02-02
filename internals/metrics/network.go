package metrics

import (
	"strconv"
	"strings"
)

func GetNetworkStats() (*NetworkStats, error) {
	stats := &NetworkStats{}

	lineNum := 0
	err := processFileLines("/proc/self/net/dev", func(line string) {
		lineNum++
		if lineNum <= 2 {
			return
		}

		ifaceStats := parseNetDevLine(line)
		if ifaceStats != nil {
			stats.ReceiveBytesTotal += ifaceStats.ReceiveBytesTotal
			stats.TransmitBytesTotal += ifaceStats.TransmitBytesTotal
			stats.ReceivePacketsTotal += ifaceStats.ReceivePacketsTotal
			stats.TransmitPacketsTotal += ifaceStats.TransmitPacketsTotal
			stats.ReceiveErrorsTotal += ifaceStats.ReceiveErrorsTotal
			stats.TransmitErrorsTotal += ifaceStats.TransmitErrorsTotal
		}
	})

	return stats, err
}

func parseNetDevLine(line string) *NetworkStats {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil
	}

	fields := strings.Fields(parts[1])
	if len(fields) < 16 {
		return nil
	}

	stats := &NetworkStats{}

	stats.ReceiveBytesTotal = atoi(fields[0])
	stats.ReceivePacketsTotal = atoi(fields[1])
	stats.ReceiveErrorsTotal = atoi(fields[2])
	stats.TransmitBytesTotal = atoi(fields[8])
	stats.TransmitPacketsTotal = atoi(fields[9])
	stats.TransmitErrorsTotal = atoi(fields[10])

	return stats
}

const (
	tcpEstablished = 1
	tcpListen      = 10
)

func GetSocketStats() (*SocketStats, error) {
	stats := &SocketStats{}

	tcpEstablished, tcpListen := parseTCPSockets("/proc/self/net/tcp")
	stats.TCPEstablished = tcpEstablished
	stats.TCPListen = tcpListen

	tcp6Established, tcp6Listen := parseTCPSockets("/proc/self/net/tcp6")
	stats.TCPEstablished += tcp6Established
	stats.TCPListen += tcp6Listen

	stats.UDP = countUDPSockets("/proc/self/net/udp")
	stats.UDP += countUDPSockets("/proc/self/net/udp6")

	return stats, nil
}

func parseTCPSockets(path string) (established, listen uint64) {
	lineNum := 0
	_ = processFileLines(path, func(line string) {
		lineNum++
		if lineNum == 1 {
			return
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			return
		}

		state, err := strconv.ParseUint(fields[3], 16, 8)
		if err != nil {
			return
		}

		switch state {
		case tcpEstablished:
			established++
		case tcpListen:
			listen++
		}
	})

	return established, listen
}

func countUDPSockets(path string) uint64 {
	lineNum := 0
	var count uint64
	_ = processFileLines(path, func(line string) {
		lineNum++
		if lineNum == 1 {
			return
		}
		count++
	})

	return count
}
