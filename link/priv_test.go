//go:build linux

package link

import "testing"

func TestPrivilegedGID(t *testing.T) {
	gid := PrivilegedGID()
	if gid < 1 {
		t.Fatalf("gid %d", gid)
	}
}
