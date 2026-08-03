//go:build windows

package elevation

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
)

func GenerateSocketURI() string {
	return fmt.Sprintf("npipe://./pipe/crobe-%d-%d", os.Getpid(), time.Now().UnixNano())
}

func parsePipeName(socketURI string) string {
	name := strings.TrimPrefix(socketURI, "npipe://")
	name = strings.ReplaceAll(name, "/", "\\")
	if strings.HasPrefix(name, ".\\pipe\\") {
		name = "\\\\" + name
	}
	if !strings.HasPrefix(name, "\\\\.\\pipe\\") {
		name = "\\\\.\\pipe\\" + strings.TrimPrefix(name, "\\")
	}
	return name
}

func Listen(socketURI string) (net.Listener, error) {
	pipeName := parsePipeName(socketURI)
	l, err := winio.ListenPipe(pipeName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on windows named pipe %s: %w", pipeName, err)
	}
	return l, nil
}

func Dial(socketURI string) (net.Conn, error) {
	pipeName := parsePipeName(socketURI)
	timeout := 5 * time.Second
	conn, err := winio.DialPipe(pipeName, &timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial windows named pipe %s: %w", pipeName, err)
	}
	return conn, nil
}

func CleanupListener(socketURI string) {
	// Windows named pipes are automatically cleaned up when all handles are closed.
}
