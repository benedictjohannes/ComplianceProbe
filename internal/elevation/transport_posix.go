//go:build !windows

package elevation

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func GenerateSocketURI() string {
	return fmt.Sprintf("unix:///tmp/crobe-%d-%d.sock", os.Getpid(), time.Now().UnixNano())
}

func parseSocketPath(socketURI string) string {
	return strings.TrimPrefix(socketURI, "unix://")
}

func Listen(socketURI string) (net.Listener, error) {
	path := parseSocketPath(socketURI)
	_ = os.Remove(path)

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on unix socket %s: %w", path, err)
	}

	_ = os.Chmod(path, 0600)
	return l, nil
}

func Dial(socketURI string) (net.Conn, error) {
	path := parseSocketPath(socketURI)
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to dial unix socket %s: %w", path, err)
	}
	return conn, nil
}

func CleanupListener(socketURI string) {
	path := parseSocketPath(socketURI)
	_ = os.Remove(path)
}
