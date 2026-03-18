//go:build systemd

package lifecycle

import (
	"log"
	"os"

	"project-template/infrastructure/supervisor"

	"github.com/coreos/go-systemd/v22/daemon"
)

// NotifyReady informs systemd that the service finished starting and
// is ready to accept connections.
func NotifyReady() {
	// stop any temporary service before notifying systemd
	if err := supervisor.OnStartUp(); err != nil {
		log.Println("startup:", err)
	}
	if os.Getenv("NOTIFY_SOCKET") != "" {
		if _, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
			log.Printf("systemd notify failed: %v", err)
		}
	}
}
