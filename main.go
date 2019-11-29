package main

import (
	"github.com/traPtitech/traQ/cmd"
	"log"
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
