package main

import (
	"project-template/cmd"
	"project-template/pkg/lifecycle"
)

func main() {
	lifecycle.Run(cmd.Start)
}
