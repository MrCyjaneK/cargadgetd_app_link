// Package link is the Link plugin helpers: IPC, setuid, lipstick capture, uinput.
package link

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// Magic is ASCII "CGLC".
	Magic uint32 = 0x43474C43
	// Version of the capture setup header.
	Version uint32 = 1
	// CaptureFD is ExtraFiles[0] in the linkcap child (starts at 3).
	CaptureFD = 3
	// SetupSize is the capture setup header length (no fds).
	SetupSize = 24
	// FrameSize is the capture frame notification length.
	FrameSize = 12
	// FlagYInvert is lipstick transform_y_inverted (GL read is bottom-up).
	FlagYInvert uint32 = 1
	// TouchSize is one stdin touch record.
	TouchSize = 5

	ActionUp   = 0
	ActionDown = 1
	ActionMove = 2
)

// Setup is sent once from linkcap with two memfds (SCM_RIGHTS).
type Setup struct {
	Width  uint32
	Height uint32
	Stride uint32
	Format uint32
}

// Frame is a “pixels ready in slot” notification.
type Frame struct {
	Seq   uint32
	Slot  uint32
	Flags uint32
}

// Touch is one HU-mapped panel tap in native panel pixels.
type Touch struct {
	Action uint8
	X      uint16
	Y      uint16
}

func (s Setup) Marshal() []byte {
	b := make([]byte, SetupSize)
	binary.LittleEndian.PutUint32(b[0:4], Magic)
	binary.LittleEndian.PutUint32(b[4:8], Version)
	binary.LittleEndian.PutUint32(b[8:12], s.Width)
	binary.LittleEndian.PutUint32(b[12:16], s.Height)
	binary.LittleEndian.PutUint32(b[16:20], s.Stride)
	binary.LittleEndian.PutUint32(b[20:24], s.Format)
	return b
}

func ParseSetup(b []byte) (Setup, error) {
	if len(b) < SetupSize {
		return Setup{}, fmt.Errorf("setup: short header %d", len(b))
	}
	if binary.LittleEndian.Uint32(b[0:4]) != Magic {
		return Setup{}, fmt.Errorf("setup: bad magic")
	}
	if binary.LittleEndian.Uint32(b[4:8]) != Version {
		return Setup{}, fmt.Errorf("setup: unsupported version %d", binary.LittleEndian.Uint32(b[4:8]))
	}
	s := Setup{
		Width:  binary.LittleEndian.Uint32(b[8:12]),
		Height: binary.LittleEndian.Uint32(b[12:16]),
		Stride: binary.LittleEndian.Uint32(b[16:20]),
		Format: binary.LittleEndian.Uint32(b[20:24]),
	}
	if s.Width < 1 || s.Height < 1 || s.Width > 4096 || s.Height > 4096 {
		return Setup{}, fmt.Errorf("setup: bad size %dx%d", s.Width, s.Height)
	}
	if s.Stride < s.Width*4 || s.Stride > 4096*8 {
		return Setup{}, fmt.Errorf("setup: bad stride %d", s.Stride)
	}
	return s, nil
}

func (s Setup) Bytes() int {
	return int(s.Stride) * int(s.Height)
}

func (f Frame) Marshal() []byte {
	b := make([]byte, FrameSize)
	binary.LittleEndian.PutUint32(b[0:4], f.Seq)
	binary.LittleEndian.PutUint32(b[4:8], f.Slot)
	binary.LittleEndian.PutUint32(b[8:12], f.Flags)
	return b
}

func ParseFrame(b []byte) (Frame, error) {
	if len(b) < FrameSize {
		return Frame{}, fmt.Errorf("frame: short %d", len(b))
	}
	f := Frame{
		Seq:   binary.LittleEndian.Uint32(b[0:4]),
		Slot:  binary.LittleEndian.Uint32(b[4:8]),
		Flags: binary.LittleEndian.Uint32(b[8:12]),
	}
	if f.Slot > 1 {
		return Frame{}, fmt.Errorf("frame: bad slot %d", f.Slot)
	}
	return f, nil
}

func (f Frame) YInvert() bool { return f.Flags&FlagYInvert != 0 }

func (t Touch) Marshal() []byte {
	b := make([]byte, TouchSize)
	b[0] = t.Action
	binary.LittleEndian.PutUint16(b[1:3], t.X)
	binary.LittleEndian.PutUint16(b[3:5], t.Y)
	return b
}

func ParseTouch(b []byte) (Touch, error) {
	if len(b) < TouchSize {
		return Touch{}, fmt.Errorf("touch: short %d", len(b))
	}
	t := Touch{
		Action: b[0],
		X:      binary.LittleEndian.Uint16(b[1:3]),
		Y:      binary.LittleEndian.Uint16(b[3:5]),
	}
	if t.Action > ActionMove {
		return Touch{}, fmt.Errorf("touch: bad action %d", t.Action)
	}
	return t, nil
}

// ReadFullTouch reads one record from r.
func ReadFullTouch(r io.Reader) (Touch, error) {
	var b [TouchSize]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Touch{}, err
	}
	return ParseTouch(b[:])
}
