package testutils

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
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

func WaitForPort(port string, timeout time.Duration) error {
	start := time.Now()
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", port))
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("port %s did not become ready within %v", port, timeout)
		}
		time.Sleep(100 * time.Millisecond) // Polling interval
	}
}

func WaitForServerLog(reader io.Reader, logMessage string, timeout time.Duration) error {
	scanner := bufio.NewScanner(reader)
	start := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, logMessage) {
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("server did not log '%s' within %v", logMessage, timeout)
		}
	}
	return scanner.Err()
}
