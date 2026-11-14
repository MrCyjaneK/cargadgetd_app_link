//go:build linux

package link

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// SendSetup writes the header and two memfds on a Unix socket (linkcap → plugin).
func SendSetup(fd int, s Setup, memfd [2]int) error {
	oob := unix.UnixRights(memfd[0], memfd[1])
	return unix.Sendmsg(fd, s.Marshal(), oob, nil, 0)
}

// RecvSetup reads the header and two memfds. Caller owns and must close the fds.
func RecvSetup(fd int) (Setup, [2]int, error) {
	var none [2]int
	buf := make([]byte, SetupSize)
	oob := make([]byte, unix.CmsgSpace(8*2))
	n, oobn, _, _, err := unix.Recvmsg(fd, buf, oob, 0)
	if err != nil {
		return Setup{}, none, err
	}
	if n < SetupSize {
		if n == 0 {
			return Setup{}, none, fmt.Errorf("setup: helper closed (no header)")
		}
		return Setup{}, none, fmt.Errorf("setup: short payload %d", n)
	}
	s, err := ParseSetup(buf[:n])
	if err != nil {
		return Setup{}, none, err
	}
	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return Setup{}, none, err
	}
	var fds []int
	for _, m := range msgs {
		got, err := unix.ParseUnixRights(&m)
		if err != nil {
			return Setup{}, none, err
		}
		fds = append(fds, got...)
	}
	if len(fds) != 2 {
		for _, f := range fds {
			_ = unix.Close(f)
		}
		return Setup{}, none, fmt.Errorf("setup: got %d fds, want 2", len(fds))
	}
	return s, [2]int{fds[0], fds[1]}, nil
}

// SendFrame writes one frame notification.
func SendFrame(fd int, f Frame) error {
	_, err := unix.Write(fd, f.Marshal())
	return err
}

// RecvFrame reads one frame notification.
func RecvFrame(fd int) (Frame, error) {
	var b [FrameSize]byte
	n := 0
	for n < FrameSize {
		got, err := unix.Read(fd, b[n:])
		if err != nil {
			return Frame{}, err
		}
		if got == 0 {
			return Frame{}, fmt.Errorf("frame: eof")
		}
		n += got
	}
	return ParseFrame(b[:])
}
