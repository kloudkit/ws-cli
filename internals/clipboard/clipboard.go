package clipboard

import (
	"fmt"
	"io"

	"github.com/kloudkit/ws-cli/internals/net"
)

func Paste(writer io.Writer) error {
	client := net.GetIPCClient()

	resp, err := client.Get("http://localhost/clipboard")
	if err != nil {
		return fmt.Errorf("error retrieving from workspace socket: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("error outputting clipboard data: %v", err)
	}

	return nil
}
