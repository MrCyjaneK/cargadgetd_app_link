//go:build linux

// Command linkcap is the setuid-root lipstick_recorder helper for the Link plugin.
//
// ExtraFiles[0] (fd 3) is a Unix socket to the unprivileged plugin.
package main

import (
	"log"

	"github.com/mrcyjanek/cargadgetd_app_link/link"
)

func main() {
	log.SetPrefix("linkcap: ")
	log.SetFlags(0)
	if err := link.RequireSetuid(); err != nil {
		log.Fatal(err)
	}
	if err := link.BecomeRoot(); err != nil {
		log.Fatal(err)
	}
	// Lipstick bind ACL is primary gid == "privileged" (SO_PEERCRED), not uid 0.
	if err := link.SetPrivilegedGID(); err != nil {
		log.Fatal(err)
	}
	if err := link.Capture(link.CaptureFD, link.SocketPath()); err != nil {
		log.Fatal(err)
	}
}
