//go:build systemd

package supervisor

import (
	"errors"
	"net"
	"os"
	"strings"
)

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

	socket := os.Getenv("SUPERVISOR_SOCKET")
	if socket == "" {
		return errors.New("No suitable socket found")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(msg + "\n"))
	return err
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
	socket := os.Getenv("SUPERVISOR_SOCKET")
	if socket == "" {
		return errors.New("No suitable socket found")
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte("stop " + temp + "\n"))
	return err
}
