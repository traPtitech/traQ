package main

import (
	"log"

	"github.com/traPtitech/traQ/cmd"
)

var (
	version  = "UNKNOWN"
	revision = "UNKNOWN"
)

func main() {
	cmd.Version = version
	cmd.Revision = revision
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
