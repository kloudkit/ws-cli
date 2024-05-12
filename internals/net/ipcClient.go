package net

import (
	"context"
	"github.com/kloudkit/ws-cli/internals/path"
	nativeNet "net"
	"net/http"
	"time"
)

func GetIPCClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (nativeNet.Conn, error) {
				return nativeNet.Dial("unix", path.GetIPCSocket())
			},
		},
		Timeout: 3 * time.Second,
	}
}
