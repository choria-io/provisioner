package main

//go:generate go run agent/gen.go

import (
	"github.com/choria-io/provisioning-agent/cmd"
)

func main() {
	cmd.Run()
}
