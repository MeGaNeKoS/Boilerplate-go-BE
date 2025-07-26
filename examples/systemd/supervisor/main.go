package main

import (
	"log"
	"os"

	"project-template/examples/systemd/supervisor/connection"
)

func run() error {
	socket := os.Getenv("SUPERVISOR_SOCKET")
	return connection.Listen(socket)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
