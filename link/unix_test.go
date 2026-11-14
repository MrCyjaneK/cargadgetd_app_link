//go:build linux

package link

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestSetupFdsRoundTrip(t *testing.T) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])

	m0, err := unix.MemfdCreate("linkipc-t0", unix.MFD_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(m0)
	m1, err := unix.MemfdCreate("linkipc-t1", unix.MFD_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(m1)
	if err := unix.Ftruncate(m0, 4096); err != nil {
		t.Fatal(err)
	}
	if err := unix.Ftruncate(m1, 4096); err != nil {
		t.Fatal(err)
	}

	want := Setup{Width: 16, Height: 8, Stride: 64, Format: 0}
	errc := make(chan error, 1)
	go func() {
		errc <- SendSetup(fds[1], want, [2]int{m0, m1})
	}()
	got, gfds, err := RecvSetup(fds[0])
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(gfds[0])
	defer unix.Close(gfds[1])
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("setup %+v want %+v", got, want)
	}

	if err := SendFrame(fds[1], Frame{Seq: 3, Slot: 1, Flags: FlagYInvert}); err != nil {
		t.Fatal(err)
	}
	fr, err := RecvFrame(fds[0])
	if err != nil {
		t.Fatal(err)
	}
	if fr.Seq != 3 || fr.Slot != 1 || !fr.YInvert() {
		t.Fatalf("frame %+v", fr)
	}
}
