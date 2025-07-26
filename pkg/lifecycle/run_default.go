package lifecycle

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Run executes the provided start function in a goroutine and gracefully
// shuts down the application when a termination signal is received or when
// the start function indicates completion via the provided channel.
func Run(startFn func(chan struct{})) {
	doneChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)

	go startFn(doneChan)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Println("Received signal:", sig.String())
		CloseServerListener()
		if err := ShutdownServer(); err != nil {
			log.Println("Error shutting down server:", err)
		}
	case <-doneChan:
		log.Println("cmd finished execution (e.g., version flag).")
	}

	log.Println("Application stopped gracefully.")
}
