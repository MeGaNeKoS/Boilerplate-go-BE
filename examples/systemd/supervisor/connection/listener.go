package connection

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Listen starts a Unix domain socket listener and blocks serving restart
// requests. The provided socket path may be empty to use the default.
func Listen(socket string) error {
	if socket == "" {
		return errors.New("no socket specified")
	}
	if err := os.RemoveAll(socket); err != nil {
		return fmt.Errorf("failed to remove old socket: %w", err)
	}
	l, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}

	// *** Set permissions so group can write to the socket ***
	if err := os.Chmod(socket, 0660); err != nil {
		log.Printf("Failed to chmod socket: %v", err)
	}

	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			log.Printf("Failed to close listener: %v", err)
		}
	}(l)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handle(conn)
	}
}

func handle(c net.Conn) {
	defer func(c net.Conn) {
		if err := c.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}(c)

	r := bufio.NewReader(c)
	line, err := r.ReadString('\n')
	if err != nil {
		log.Printf("read error: %v", err)
		return
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	fields := strings.Fields(line)
	cmdName := fields[0]
	arg := strings.TrimSpace(strings.TrimPrefix(line, cmdName))

	switch cmdName {
	case "stop":
		svc := strings.TrimSpace(arg)
		if svc == "" {
			return
		}
		if err := exec.Command("systemctl", "stop", svc).Run(); err != nil {
			log.Printf("systemctl stop %s failed: %v", svc, err)
		}
	case "restart":
		fields := strings.Fields(arg)
		if len(fields) == 0 {
			return
		}
		parts := strings.Split(fields[0], ",")
		mainSvc := strings.TrimSpace(parts[0])
		if mainSvc == "" {
			return
		}

		timeout := 0
		if len(fields) > 1 {
			if v, err := strconv.Atoi(fields[1]); err == nil {
				timeout = v
			}
		}
		if timeout == 0 {
			if env := os.Getenv("TEMP_SERVICE_ACTIVE_TIMEOUT"); env != "" {
				if v, err := strconv.Atoi(env); err == nil {
					timeout = v
				}
			}
		}
		if timeout == 0 {
			timeout = 30
		}

		if len(parts) > 1 {
			tmp := strings.TrimSpace(parts[1])
			if tmp != "" {
				if err := exec.Command("systemctl", "start", tmp).Run(); err != nil {
					log.Printf("systemctl start %s failed: %v", tmp, err)
				} else {
					deadline := time.Now().Add(time.Duration(timeout) * time.Second)
					for {
						out, err := exec.Command("systemctl", "is-active", tmp).Output()
						if err == nil && strings.TrimSpace(string(out)) == "active" {
							break
						}
						if time.Now().After(deadline) {
							log.Printf("temporary service not active: %v %s", err, strings.TrimSpace(string(out)))
							break
						}
						time.Sleep(time.Second)
					}
				}
			}
		}
		if err := exec.Command("systemctl", "restart", mainSvc).Run(); err != nil {
			log.Printf("systemctl restart %s failed: %v", mainSvc, err)
		}
	}
}
