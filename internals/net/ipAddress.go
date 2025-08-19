package net

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func hexToIPv4(hex string) (net.IP, error) {
	ip := make(net.IP, 4)

	_, err := fmt.Sscanf(hex, "%02x%02x%02x%02x", &ip[3], &ip[2], &ip[1], &ip[0])

	if err != nil {
		return nil, err
	}

	return ip, nil
}

func GetNodeIP() (string, error) {
	data, err := os.ReadFile("/proc/net/route")

	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)

		if len(fields) >= 3 && fields[1] == "00000000" {
			ip, err := hexToIPv4(fields[2])

			if err != nil {
				return "", err
			}

			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("default gateway not found")
}

func GetInternalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		var ip net.IP

		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			ip = ipnet.IP.To4()

			if ip != nil {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("internal IP not found")
}
