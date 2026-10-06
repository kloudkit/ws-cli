package metrics

import (
	"strconv"
	"strings"
)

func GetNetworkStats() (*NetworkStats, error) {
	stats := &NetworkStats{}

	err := processFileLines("/proc/self/net/dev", 2, func(line string) {
		_, counters, found := strings.Cut(line, ":")
		fields := strings.Fields(counters)
		if !found || len(fields) < 16 {
			return
		}

		stats.ReceiveBytesTotal += atoi(fields[0])
		stats.ReceivePacketsTotal += atoi(fields[1])
		stats.ReceiveErrorsTotal += atoi(fields[2])
		stats.TransmitBytesTotal += atoi(fields[8])
		stats.TransmitPacketsTotal += atoi(fields[9])
		stats.TransmitErrorsTotal += atoi(fields[10])
	})

	return stats, err
}

const (
	tcpEstablished = 1
	tcpListen      = 10
)

func GetSocketStats() (*SocketStats, error) {
	established, listen := parseTCPSockets("/proc/self/net/tcp")
	established6, listen6 := parseTCPSockets("/proc/self/net/tcp6")

	return &SocketStats{
		TCPEstablished: established + established6,
		TCPListen:      listen + listen6,
		UDP:            countUDPSockets("/proc/self/net/udp") + countUDPSockets("/proc/self/net/udp6"),
	}, nil
}

func parseTCPSockets(path string) (established, listen uint64) {
	_ = processFileLines(path, 1, func(line string) {
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
	var count uint64
	_ = processFileLines(path, 1, func(string) { count++ })

	return count
}
