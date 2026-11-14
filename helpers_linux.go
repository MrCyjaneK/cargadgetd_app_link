//go:build linux

package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/mrcyjanek/cargadgetd_app_link/link"
	"golang.org/x/sys/unix"
)

const (
	linkcapBin    = "cargadgetd-linkcap"
	linktouchBin  = "cargadgetd-linktouch"
	defaultBinDir = "/usr/share/cargadgetd/bin"
)

type capture struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	fd      int
	setup   link.Setup
	maps    [2][]byte
	memfd   [2]int
	slot    uint32
	flipY   bool
	ready   bool
	stopped bool
}

func (c *capture) start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != nil || c.stopped {
		return
	}
	path := helperPath(linkcapBin)
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		log.Printf("link: socketpair: %v", err)
		return
	}
	child := os.NewFile(uintptr(pair[1]), "linkcap-child")
	cmd := exec.Command(path)
	cmd.ExtraFiles = []*os.File{child}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = unix.Close(pair[0])
		_ = unix.Close(pair[1])
		_ = child.Close()
		log.Printf("link: start %s: %v", path, err)
		return
	}
	_ = child.Close()
	_ = unix.Close(pair[1])
	st, fds, err := link.RecvSetup(pair[0])
	if err != nil {
		_ = unix.Close(pair[0])
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		log.Printf("link: capture setup: %v", err)
		return
	}
	unix.CloseOnExec(fds[0])
	unix.CloseOnExec(fds[1])
	n := st.Bytes()
	var maps [2][]byte
	for i := 0; i < 2; i++ {
		maps[i], err = unix.Mmap(fds[i], 0, n, unix.PROT_READ, unix.MAP_SHARED)
		if err != nil {
			log.Printf("link: mmap %d: %v", i, err)
			_ = unix.Close(fds[0])
			_ = unix.Close(fds[1])
			_ = unix.Close(pair[0])
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return
		}
	}
	c.cmd = cmd
	c.fd = pair[0]
	c.setup = st
	c.maps = maps
	c.memfd = fds
	c.ready = true
	go c.readFrames()
}

func (c *capture) readFrames() {
	c.mu.Lock()
	fd := c.fd
	c.mu.Unlock()
	for {
		fr, err := link.RecvFrame(fd)
		if err != nil {
			return
		}
		c.mu.Lock()
		c.slot = fr.Slot
		c.flipY = fr.YInvert()
		c.mu.Unlock()
	}
}

func (c *capture) snapshot() (src []byte, stride, w, h int, flipY, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready || c.maps[c.slot] == nil {
		return nil, 0, 0, 0, false, false
	}
	src = c.maps[c.slot]
	stride = int(c.setup.Stride)
	w = int(c.setup.Width)
	h = int(c.setup.Height)
	return src, stride, w, h, c.flipY, true
}

func (c *capture) size() (w, h int, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready {
		return 0, 0, false
	}
	return int(c.setup.Width), int(c.setup.Height), true
}

func (c *capture) stop() {
	c.mu.Lock()
	cmd := c.cmd
	fd := c.fd
	maps := c.maps
	memfd := c.memfd
	c.cmd = nil
	c.fd = 0
	c.maps = [2][]byte{}
	c.ready = false
	c.stopped = true
	c.mu.Unlock()
	if fd > 0 {
		_ = unix.Close(fd)
	}
	for i := 0; i < 2; i++ {
		if maps[i] != nil {
			_ = unix.Munmap(maps[i])
		}
		if memfd[i] > 0 {
			_ = unix.Close(memfd[i])
		}
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_, _ = cmd.Process.Wait()
	}
	c.mu.Lock()
	c.stopped = false
	c.mu.Unlock()
}

type touchOut struct {
	mu  sync.Mutex
	cmd *exec.Cmd
	w   io.WriteCloser
	pw  int
	ph  int
}

func (t *touchOut) start(w, h int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd != nil || w < 1 || h < 1 {
		return
	}
	path := helperPath(linktouchBin)
	pr, pw, err := os.Pipe()
	if err != nil {
		log.Printf("link: pipe: %v", err)
		return
	}
	cmd := exec.Command(path, "-w", strconv.Itoa(w), "-h", strconv.Itoa(h))
	cmd.Stdin = pr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		log.Printf("link: start %s: %v", path, err)
		return
	}
	_ = pr.Close()
	t.cmd = cmd
	t.w = pw
	t.pw, t.ph = w, h
}

func (t *touchOut) send(action, x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.w == nil {
		return
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if t.pw > 0 && x >= t.pw {
		x = t.pw - 1
	}
	if t.ph > 0 && y >= t.ph {
		y = t.ph - 1
	}
	rec := link.Touch{Action: uint8(action), X: uint16(x), Y: uint16(y)}
	_, _ = t.w.Write(rec.Marshal())
}

func (t *touchOut) stop() {
	t.mu.Lock()
	cmd := t.cmd
	w := t.w
	t.cmd = nil
	t.w = nil
	t.mu.Unlock()
	if w != nil {
		_ = w.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_, _ = cmd.Process.Wait()
	}
}

func helperPath(name string) string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return filepath.Join(defaultBinDir, name)
}
