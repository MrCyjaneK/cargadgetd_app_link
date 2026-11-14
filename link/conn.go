//go:build linux

package link

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const displayID = 1

type conn struct {
	fd   int
	next uint32
	rbuf []byte
}

func dial(path string) (*conn, error) {
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	if err := unix.Connect(fd, &unix.SockaddrUnix{Name: path}); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("connect %s: %w", path, err)
	}
	return &conn{fd: fd, next: 2}, nil
}

func (c *conn) close() { _ = unix.Close(c.fd) }

func (c *conn) alloc() uint32 {
	id := c.next
	c.next++
	return id
}

func (c *conn) send(obj uint32, opcode uint16, payload []byte, fds []int) error {
	size := 8 + len(payload)
	if size%4 != 0 {
		return fmt.Errorf("wayland: unaligned %d", size)
	}
	hdr := make([]byte, size)
	binary.LittleEndian.PutUint32(hdr[0:4], obj)
	binary.LittleEndian.PutUint16(hdr[4:6], opcode)
	binary.LittleEndian.PutUint16(hdr[6:8], uint16(size))
	copy(hdr[8:], payload)
	var oob []byte
	if len(fds) > 0 {
		oob = unix.UnixRights(fds...)
	}
	return unix.Sendmsg(c.fd, hdr, oob, nil, 0)
}

func (c *conn) readMsg() (obj uint32, opcode uint16, payload []byte, err error) {
	for len(c.rbuf) < 8 {
		if err = c.fill(); err != nil {
			return
		}
	}
	obj = binary.LittleEndian.Uint32(c.rbuf[0:4])
	opcode = binary.LittleEndian.Uint16(c.rbuf[4:6])
	size := int(binary.LittleEndian.Uint16(c.rbuf[6:8]))
	if size < 8 || size%4 != 0 {
		err = fmt.Errorf("wayland: bad size %d", size)
		return
	}
	for len(c.rbuf) < size {
		if err = c.fill(); err != nil {
			return
		}
	}
	payload = append([]byte(nil), c.rbuf[8:size]...)
	c.rbuf = c.rbuf[size:]
	return
}

func (c *conn) fill() error {
	buf := make([]byte, 8192)
	n, err := unix.Read(c.fd, buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("wayland: eof")
	}
	c.rbuf = append(c.rbuf, buf[:n]...)
	return nil
}

func putU32(b []byte, v uint32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	return append(b, tmp[:]...)
}

func putI32(b []byte, v int32) []byte {
	return putU32(b, uint32(v))
}

func putString(b []byte, s string) []byte {
	n := len(s) + 1
	pad := (n + 3) &^ 3
	buf := make([]byte, 4+pad)
	binary.LittleEndian.PutUint32(buf, uint32(n))
	copy(buf[4:], s)
	return append(b, buf...)
}

func getU32(b []byte, off int) uint32 {
	return binary.LittleEndian.Uint32(b[off : off+4])
}

func getI32(b []byte, off int) int32 {
	return int32(getU32(b, off))
}

func getString(b []byte, off int) (string, int) {
	n := int(getU32(b, off))
	off += 4
	if n < 1 || off+n > len(b) {
		return "", off
	}
	s := string(b[off : off+n-1])
	pad := (n + 3) &^ 3
	return s, off + pad
}

// SocketPath is the compositor socket. No argv paths.
func SocketPath() string {
	d := os.Getenv("WAYLAND_DISPLAY")
	if d == "" {
		return "/run/display/wayland-0"
	}
	if filepath.IsAbs(d) {
		return d
	}
	rt := os.Getenv("XDG_RUNTIME_DIR")
	if rt == "" {
		return "/run/display/wayland-0"
	}
	return filepath.Clean(filepath.Join(rt, d))
}
