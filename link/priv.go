//go:build linux

package link

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// RequireSetuid fatals the process unless the binary is running with euid 0.
func RequireSetuid() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("needs setuid root (euid=%d)", os.Geteuid())
	}
	return nil
}

// BecomeRoot sets ruid/euid/gid to 0. Same reason as gadgetctl: some kernel
// checks look at the real uid.
func BecomeRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("euid=%d", os.Geteuid())
	}
	if os.Getuid() == 0 {
		return nil
	}
	if err := syscall.Setgid(0); err != nil {
		return fmt.Errorf("setgid: %w", err)
	}
	if err := syscall.Setuid(0); err != nil {
		return fmt.Errorf("setuid: %w", err)
	}
	return nil
}

// PrivilegedGID is lipstick's recorder ACL: it compares SO_PEERCRED.gid
// (primary/effective gid) to group "privileged", not supplementary groups
// and not uid 0.
func PrivilegedGID() int {
	g, err := user.LookupGroup("privileged")
	if err == nil {
		if id, err := strconv.Atoi(g.Gid); err == nil && id > 0 {
			return id
		}
	}
	return 995
}

// SetPrivilegedGID sets the primary gid to "privileged" so lipstick will
// allow binding lipstick_recorder_manager. Must run while euid is 0.
func SetPrivilegedGID() error {
	gid := PrivilegedGID()
	if err := unix.Setresgid(gid, gid, gid); err != nil {
		return fmt.Errorf("setresgid privileged %d: %w", gid, err)
	}
	return nil
}

// DropToReal switches back to the invoking user after opening privileged fds.
func DropToReal(ruid, rgid int) error {
	if ruid == 0 {
		return nil
	}
	if err := unix.Setresgid(rgid, rgid, rgid); err != nil {
		return fmt.Errorf("setresgid: %w", err)
	}
	if err := unix.Setresuid(ruid, ruid, ruid); err != nil {
		return fmt.Errorf("setresuid: %w", err)
	}
	return nil
}
