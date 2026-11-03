package net

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
)

func GetNodeIP() (string, error) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return "", err
	}

	return defaultGateway(string(data))
}

func defaultGateway(routes string) (string, error) {
	for line := range strings.Lines(routes) {
		if fields := strings.Fields(line); len(fields) >= 3 && fields[1] == "00000000" {
			gw, err := strconv.ParseUint(fields[2], 16, 32)
			if err != nil {
				return "", err
			}

			return net.IPv4(byte(gw), byte(gw>>8), byte(gw>>16), byte(gw>>24)).String(), nil
		}
	}

	return "", errors.New("default gateway not found")
}

func GetInternalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.To4().String(), nil
		}
	}

	return "", errors.New("internal IP not found")
}
