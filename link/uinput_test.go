//go:build linux

package link

import (
	"os"
	"testing"
)

func TestOpenUinput(t *testing.T) {
	if _, err := os.Stat("/dev/uinput"); err != nil {
		t.Skip("/dev/uinput missing")
	}
	fd, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("/dev/uinput: %v", err)
	}
	_ = fd.Close()
	d, err := Open(720, 1600)
	if err != nil {
		t.Skipf("UI_DEV_CREATE: %v", err)
	}
	defer d.Close()
	if err := d.Write(Down(10, 20, 1)); err != nil {
		t.Fatal(err)
	}
	if err := d.Write(Up()); err != nil {
		t.Fatal(err)
	}
}
