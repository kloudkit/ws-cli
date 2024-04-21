package net

import (
	"context"
	nativeNet "net"
	"net/http"
	"time"
)

func GetIPCClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (nativeNet.Conn, error) {
				return nativeNet.Dial("unix", "/tmp/workspace-ipc.sock")
			},
		},
		Timeout: 3 * time.Second,
	}
}
