//go:build linux

// Command linktouch is the setuid-root uinput helper for the Link plugin.
//
//	cargadgetd-linktouch -w 720 -h 1600   # stdin: 5-byte touch records
package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/mrcyjanek/cargadgetd_app_link/link"
)

func main() {
	log.SetPrefix("linktouch: ")
	log.SetFlags(0)
	w := flag.Int("w", 0, "panel width (ABS max+1)")
	h := flag.Int("h", 0, "panel height (ABS max+1)")
	flag.Parse()
	if *w < 1 || *h < 1 {
		log.Fatal("need -w and -h")
	}

	ruid, rgid := os.Getuid(), os.Getgid()
	if err := link.RequireSetuid(); err != nil {
		log.Fatal(err)
	}
	if err := link.BecomeRoot(); err != nil {
		log.Fatal(err)
	}

	dev, err := link.Open(*w, *h)
	if err != nil {
		log.Fatal(err)
	}
	defer dev.Close()

	if err := link.DropToReal(ruid, rgid); err != nil {
		log.Fatal(err)
	}

	var tracking int32
	down := false
	for {
		t, err := link.ReadFullTouch(os.Stdin)
		if err != nil {
			if err != io.EOF {
				log.Print(err)
			}
			return
		}
		x, y := int32(t.X), int32(t.Y)
		if int(x) >= *w || int(y) >= *h {
			continue
		}
		var ev []byte
		switch t.Action {
		case link.ActionDown:
			tracking++
			if tracking <= 0 {
				tracking = 1
			}
			ev = link.Down(x, y, tracking)
			down = true
		case link.ActionMove:
			if !down {
				tracking++
				if tracking <= 0 {
					tracking = 1
				}
				ev = link.Down(x, y, tracking)
				down = true
				break
			}
			ev = link.Move(x, y)
		case link.ActionUp:
			if !down {
				continue
			}
			ev = link.Up()
			down = false
		default:
			continue
		}
		if err := dev.Write(ev); err != nil {
			log.Fatal(err)
		}
	}
}
