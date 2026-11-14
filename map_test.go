package main

import "testing"

func TestLayout800x480(t *testing.T) {
	l := newLayout(800, 480, 720, 1600)
	if l.DW != 800 || l.DH != 360 || l.DX != 0 || l.DY != 60 {
		t.Fatalf("layout %+v", l)
	}
	if _, _, ok := l.HUToPanel(10, 10); ok {
		t.Fatal("letterbox tap should miss")
	}
	// dest origin (0,60) = 90° CCW = panel top-right
	px, py, ok := l.HUToPanel(0, 60)
	if !ok || px != 719 || py != 0 {
		t.Fatalf("origin -> %d,%d ok=%v", px, py, ok)
	}
	// dest top-right (799, 60) ≈ panel bottom-right
	px, py, ok = l.HUToPanel(799, 60)
	if !ok || px != 719 || py < 1598 {
		t.Fatalf("top-right -> %d,%d ok=%v", px, py, ok)
	}
}

func TestLayoutInverseCenter(t *testing.T) {
	l := newLayout(800, 480, 720, 1600)
	hx, hy := 400, 240
	px, py, ok := l.HUToPanel(hx, hy)
	if !ok {
		t.Fatal("center miss")
	}
	if px < 0 || px >= 720 || py < 0 || py >= 1600 {
		t.Fatalf("out of panel %d,%d", px, py)
	}
}

func TestLayoutTiny(t *testing.T) {
	l := newLayout(0, 0, 720, 1600)
	if _, _, ok := l.HUToPanel(0, 0); ok {
		t.Fatal("empty layout")
	}
}
