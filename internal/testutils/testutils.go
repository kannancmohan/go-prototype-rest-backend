package testutils

import (
	"net"
	"strings"
)

func GetFreePort() (string, error) {
	listener, err := net.Listen("tcp", ":0") // Listen on any available port by passing port 0
	if err != nil {
		return "", err
	}
	defer listener.Close() // Make sure the listener is closed when we're done
	address := listener.Addr().String()
	port := address[strings.LastIndex(address, ":")+1:]
	return port, nil
}
