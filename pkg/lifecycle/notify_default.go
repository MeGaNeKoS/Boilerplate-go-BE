//go:build !systemd

package lifecycle

import (
	"log"

	"project-template/infrastructure/supervisor"
)

// NotifyReady performs startup tasks when systemd integration is disabled.
func NotifyReady() {
	// ensure any temporary service is terminated when running without systemd
	if err := supervisor.OnStartUp(); err != nil {
		log.Println("startup:", err)
	}
}
