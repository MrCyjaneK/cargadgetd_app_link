//go:build linux

package link

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	uiDevCreate  = 0x5501
	uiDevDestroy = 0x5502
	uiSetEvbit   = 0x40045564
	uiSetKeybit  = 0x40045565
	uiSetAbsbit  = 0x40045567
	uiSetPropbit = 0x4004556e

	inputPropDirect = 0x01
	absCnt          = 64
	uinputNameSize  = 80
)

type uinputUserDev struct {
	Name         [uinputNameSize]byte
	Bus          uint16
	Vendor       uint16
	Product      uint16
	Version      uint16
	FFEffectsMax uint32
	Absmax       [absCnt]int32
	Absmin       [absCnt]int32
	Absfuzz      [absCnt]int32
	Absflat      [absCnt]int32
}

// Device is an open uinput virtual digitizer.
type Device struct {
	fd int
}

// Open creates a direct-touch type-B MT device covering [0,w) × [0,h).
func Open(w, h int) (*Device, error) {
	if w < 1 || h < 1 || w > 4096 || h > 4096 {
		return nil, fmt.Errorf("uinput: bad abs %dx%d", w, h)
	}
	fd, err := unix.Open("/dev/uinput", unix.O_WRONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/uinput: %w", err)
	}
	set := func(req, v int) error {
		if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(v)); errno != 0 {
			return errno
		}
		return nil
	}
	if err := set(uiSetEvbit, evKey); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("EV_KEY: %w", err)
	}
	if err := set(uiSetEvbit, evAbs); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("EV_ABS: %w", err)
	}
	if err := set(uiSetEvbit, evSyn); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("EV_SYN: %w", err)
	}
	if err := set(uiSetKeybit, btnTouch); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("BTN_TOUCH: %w", err)
	}
	for _, a := range []int{absX, absY, absMTSlot, absMTPositionX, absMTPositionY, absMTTrackingID} {
		if err := set(uiSetAbsbit, a); err != nil {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("ABS 0x%x: %w", a, err)
		}
	}
	_ = set(uiSetPropbit, inputPropDirect)

	var u uinputUserDev
	copy(u.Name[:], []byte("cargadgetd-link"))
	u.Bus = 0x06 // BUS_VIRTUAL
	u.Vendor = 0x0001
	u.Product = 0x0001
	u.Version = 1
	amax := int32(w - 1)
	hmax := int32(h - 1)
	u.Absmax[absX] = amax
	u.Absmax[absY] = hmax
	u.Absmax[absMTSlot] = 9
	u.Absmax[absMTPositionX] = amax
	u.Absmax[absMTPositionY] = hmax
	u.Absmax[absMTTrackingID] = 65535

	raw := unsafe.Slice((*byte)(unsafe.Pointer(&u)), int(unsafe.Sizeof(u)))
	if _, err := unix.Write(fd, raw); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("uinput_user_dev: %w", err)
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uiDevCreate, 0); errno != 0 {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("UI_DEV_CREATE: %v", errno)
	}
	return &Device{fd: fd}, nil
}

func (d *Device) Write(p []byte) error {
	if d == nil || d.fd < 0 {
		return os.ErrClosed
	}
	_, err := unix.Write(d.fd, p)
	return err
}

func (d *Device) Close() error {
	if d == nil || d.fd < 0 {
		return nil
	}
	_, _, _ = unix.Syscall(unix.SYS_IOCTL, uintptr(d.fd), uiDevDestroy, 0)
	err := unix.Close(d.fd)
	d.fd = -1
	return err
}
