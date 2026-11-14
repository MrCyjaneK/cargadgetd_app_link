//go:build !linux

package main

type capture struct{}

func (c *capture) start() {}
func (c *capture) stop()  {}
func (c *capture) snapshot() ([]byte, int, int, int, bool, bool) {
	return nil, 0, 0, 0, false, false
}
func (c *capture) size() (int, int, bool) { return 0, 0, false }

type touchOut struct{}

func (t *touchOut) start(w, h int)        {}
func (t *touchOut) send(action, x, y int) {}
func (t *touchOut) stop()                 {}
