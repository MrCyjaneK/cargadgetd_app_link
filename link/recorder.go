//go:build linux

package link

import (
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

const (
	wlSHMFormatARGB8888 = 0

	opDisplaySync         = 0
	opDisplayGetRegistry  = 1
	evDisplayError        = 0
	evCallbackDone        = 0
	opRegistryBind        = 0
	evRegistryGlobal      = 0
	opSHMCreatePool       = 0
	opPoolCreateBuffer    = 0
	opRecMgrCreate        = 0
	opRecorderRecordFrame = 1
	opRecorderRepaint     = 2
	evRecorderSetup       = 0
	evRecorderFrame       = 1
	evRecorderFailed      = 2
	evRecorderCancelled   = 3
)

type global struct {
	name    uint32
	iface   string
	version uint32
}

type setupEv struct {
	w, h, stride, format int32
}

const (
	recTransformNormal = 1
	recTransformYInv   = 2
)

// Capture talks lipstick_recorder and streams SHM frames to ipcFD (ExtraFiles 3).
func Capture(ipcFD int, sock string) error {
	c, err := dial(sock)
	if err != nil {
		return err
	}
	defer c.close()

	registry := c.alloc()
	if err := c.send(displayID, opDisplayGetRegistry, putU32(nil, registry), nil); err != nil {
		return err
	}
	globals, err := syncGlobals(c, registry)
	if err != nil {
		return err
	}

	var shmName, outName, mgrName uint32
	var shmVer, outVer, mgrVer uint32
	for _, g := range globals {
		switch g.iface {
		case "wl_shm":
			shmName, shmVer = g.name, g.version
		case "wl_output":
			if outName == 0 {
				outName, outVer = g.name, g.version
			}
		case "lipstick_recorder_manager":
			mgrName, mgrVer = g.name, g.version
		}
	}
	if shmName == 0 || outName == 0 || mgrName == 0 {
		return fmt.Errorf("missing globals shm=%d output=%d recorder=%d", shmName, outName, mgrName)
	}

	shm := c.alloc()
	if err := bind(c, registry, shmName, "wl_shm", minVer(shmVer, 1), shm); err != nil {
		return err
	}
	output := c.alloc()
	if err := bind(c, registry, outName, "wl_output", minVer(outVer, 2), output); err != nil {
		return err
	}
	mgr := c.alloc()
	if err := bind(c, registry, mgrName, "lipstick_recorder_manager", minVer(mgrVer, 1), mgr); err != nil {
		return err
	}

	rec := c.alloc()
	payload := putU32(putU32(nil, rec), output)
	if err := c.send(mgr, opRecMgrCreate, payload, nil); err != nil {
		return err
	}

	st, err := waitSetup(c, rec)
	if err != nil {
		return err
	}
	if st.stride < st.w*4 {
		st.stride = st.w * 4
	}
	bytes := int(st.stride) * int(st.h)
	if bytes < 1 || bytes > 64<<20 {
		return fmt.Errorf("setup: buffer %d bytes", bytes)
	}

	var memfd [2]int
	var pool [2]uint32
	var buf [2]uint32
	for i := 0; i < 2; i++ {
		fd, err := unix.MemfdCreate("linkcap", unix.MFD_CLOEXEC)
		if err != nil {
			return fmt.Errorf("memfd: %w", err)
		}
		if err := unix.Ftruncate(fd, int64(bytes)); err != nil {
			return err
		}
		memfd[i] = fd
		pool[i] = c.alloc()
		pl := putI32(putU32(nil, pool[i]), int32(bytes))
		if err := c.send(shm, opSHMCreatePool, pl, []int{fd}); err != nil {
			return fmt.Errorf("create_pool: %w", err)
		}
		buf[i] = c.alloc()
		var b []byte
		b = putU32(b, buf[i])
		b = putI32(b, 0)
		b = putI32(b, st.w)
		b = putI32(b, st.h)
		b = putI32(b, st.stride)
		// Qt wl_shm accepts ARGB8888/XRGB8888 only. Lipstick still fills GL_RGBA
		// (setup format RGBA8888 / 0x34324152) — paint as RGBA, same as CODeRUS.
		b = putU32(b, wlSHMFormatARGB8888)
		if err := c.send(pool[i], opPoolCreateBuffer, b, nil); err != nil {
			return fmt.Errorf("create_buffer: %w", err)
		}
	}
	if err := roundtrip(c); err != nil {
		return fmt.Errorf("shm buffers: %w", err)
	}
	setup := Setup{
		Width:  uint32(st.w),
		Height: uint32(st.h),
		Stride: uint32(st.stride),
		Format: uint32(st.format),
	}
	if err := SendSetup(ipcFD, setup, memfd); err != nil {
		return fmt.Errorf("ipc setup: %w", err)
	}

	slotByBuf := map[uint32]uint32{buf[0]: 0, buf[1]: 1}
	slot := 0
	var seq uint32
	if err := c.send(rec, opRecorderRecordFrame, putU32(nil, buf[slot]), nil); err != nil {
		return err
	}
	if err := c.send(rec, opRecorderRepaint, nil, nil); err != nil {
		return err
	}
	for {
		obj, opcode, payload, err := c.readMsg()
		if err != nil {
			return err
		}
		switch {
		case obj == displayID && opcode == evDisplayError:
			msg := ""
			if len(payload) >= 12 {
				msg, _ = getString(payload, 8)
			}
			if msg == "" {
				msg = fmt.Sprintf("%v", payload)
			}
			return fmt.Errorf("wayland error: %s", msg)
		case obj == rec && opcode == evRecorderFailed:
			return fmt.Errorf("recorder failed")
		case obj == rec && opcode == evRecorderCancelled:
			// retry same slot
			if err := c.send(rec, opRecorderRecordFrame, putU32(nil, buf[slot]), nil); err != nil {
				return err
			}
			if err := c.send(rec, opRecorderRepaint, nil, nil); err != nil {
				return err
			}
		case obj == rec && opcode == evRecorderFrame:
			if len(payload) < 4 {
				continue
			}
			b := getU32(payload, 0)
			s, ok := slotByBuf[b]
			if !ok {
				s = uint32(slot)
			}
			flags := uint32(0)
			tr := int32(recTransformYInv)
			if len(payload) >= 12 {
				tr = getI32(payload, 8)
			}
			if tr != recTransformNormal {
				flags = FlagYInvert
			}
			seq++
			if err := SendFrame(ipcFD, Frame{Seq: seq, Slot: s, Flags: flags}); err != nil {
				return err
			}
			slot = int(1 - s)
			if err := c.send(rec, opRecorderRecordFrame, putU32(nil, buf[slot]), nil); err != nil {
				return err
			}
			if err := c.send(rec, opRecorderRepaint, nil, nil); err != nil {
				return err
			}
		}
	}
}

func roundtrip(c *conn) error {
	cb := c.alloc()
	if err := c.send(displayID, opDisplaySync, putU32(nil, cb), nil); err != nil {
		return err
	}
	for {
		obj, opcode, payload, err := c.readMsg()
		if err != nil {
			return err
		}
		if obj == displayID && opcode == evDisplayError {
			msg := ""
			if len(payload) >= 12 {
				msg, _ = getString(payload, 8)
			}
			return fmt.Errorf("%s", msg)
		}
		if obj == cb && opcode == evCallbackDone {
			return nil
		}
	}
}

func bind(c *conn, registry, name uint32, iface string, version, id uint32) error {
	var p []byte
	p = putU32(p, name)
	p = putString(p, iface)
	p = putU32(p, version)
	p = putU32(p, id)
	return c.send(registry, opRegistryBind, p, nil)
}

func syncGlobals(c *conn, registry uint32) ([]global, error) {
	cb := c.alloc()
	if err := c.send(displayID, opDisplaySync, putU32(nil, cb), nil); err != nil {
		return nil, err
	}
	var globals []global
	for {
		obj, opcode, payload, err := c.readMsg()
		if err != nil {
			return nil, err
		}
		if obj == displayID && opcode == evDisplayError {
			return nil, fmt.Errorf("wayland error during registry")
		}
		if obj == registry && opcode == evRegistryGlobal {
			name := getU32(payload, 0)
			iface, off := getString(payload, 4)
			ver := getU32(payload, off)
			globals = append(globals, global{name: name, iface: iface, version: ver})
			continue
		}
		if obj == cb && opcode == evCallbackDone {
			return globals, nil
		}
	}
}

func waitSetup(c *conn, rec uint32) (setupEv, error) {
	for {
		obj, opcode, payload, err := c.readMsg()
		if err != nil {
			return setupEv{}, err
		}
		if obj == displayID && opcode == evDisplayError {
			msg := ""
			if len(payload) >= 12 {
				msg, _ = getString(payload, 8)
			}
			return setupEv{}, fmt.Errorf("wayland error (recorder bind?): %s", msg)
		}
		if obj == rec && opcode == evRecorderFailed {
			return setupEv{}, fmt.Errorf("recorder failed before setup")
		}
		if obj == rec && opcode == evRecorderSetup {
			if len(payload) < 16 {
				return setupEv{}, fmt.Errorf("short setup")
			}
			st := setupEv{
				w:      getI32(payload, 0),
				h:      getI32(payload, 4),
				stride: getI32(payload, 8),
				format: getI32(payload, 12),
			}
			if st.format == 0 {
				st.format = wlSHMFormatARGB8888
			}
			log.Printf("setup %dx%d stride=%d format=%d", st.w, st.h, st.stride, st.format)
			return st, nil
		}
	}
}

func minVer(have, want uint32) uint32 {
	if have == 0 || have > want {
		return want
	}
	return have
}
