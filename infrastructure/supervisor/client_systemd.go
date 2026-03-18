//go:build systemd

package supervisor

import (
	"errors"
	"net"
	"os"
	"strings"
)

// sendToSupervisor writes msg to the supervisor Unix socket.
func sendToSupervisor(msg string) error {
	socket := os.Getenv("SUPERVISOR_SOCKET")
	if socket == "" {
		return errors.New("no suitable socket found")
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	_, err = conn.Write([]byte(msg + "\n"))
	return err
}

// RequestRestart sends a restart request for SELF_SERVICE_UNIT to the
// supervisor. If TEMP_SERVICE_UNIT is set, it will be included so the
// supervisor can start it first before restarting the main service.
func RequestRestart() error {
	if strings.ToLower(os.Getenv("TEMP_SERVICE")) == "true" {
		return nil
	}
	mainUnit := os.Getenv("SELF_SERVICE_UNIT")
	if mainUnit == "" {
		return nil
	}
	msg := "restart " + mainUnit
	if tmp := os.Getenv("TEMP_SERVICE_UNIT"); tmp != "" {
		msg = msg + "," + tmp
	}
	return sendToSupervisor(msg)
}

// OnStartUp stops the temporary service specified by TEMP_SERVICE_UNIT when
// TEMP_SERVICE is unset or explicitly set to "false".
//
//go:noinline
func OnStartUp() error {
	if v := strings.ToLower(os.Getenv("TEMP_SERVICE")); v != "" && v != "false" {
		return nil
	}
	temp := os.Getenv("TEMP_SERVICE_UNIT")
	if temp == "" {
		return nil
	}
	return sendToSupervisor("stop " + temp)
}
