//go:build linux

package link

import (
	"testing"
)

func TestSocketPathRelative(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/100000")
	t.Setenv("WAYLAND_DISPLAY", "../../display/wayland-0")
	got := SocketPath()
	want := "/run/display/wayland-0"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestSocketPathAbs(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "/run/display/wayland-0")
	if SocketPath() != "/run/display/wayland-0" {
		t.Fatal(SocketPath())
	}
}

func TestSocketPathDefault(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	if SocketPath() != "/run/display/wayland-0" {
		t.Fatal(SocketPath())
	}
}

func TestPutGetString(t *testing.T) {
	b := putString(nil, "wl_shm")
	s, off := getString(b, 0)
	if s != "wl_shm" || off != len(b) {
		t.Fatalf("%q off=%d len=%d", s, off, len(b))
	}
}

func TestMarshalAlign(t *testing.T) {
	b := putString(putU32(nil, 3), "lipstick_recorder_manager")
	b = putU32(b, 1)
	b = putU32(b, 7)
	if len(b)%4 != 0 {
		t.Fatalf("len %d", len(b))
	}
}
