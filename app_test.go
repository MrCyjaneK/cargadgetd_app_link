package main

import (
	"bytes"
	"image"
	"testing"
	"unsafe"
)

func TestPlaceholderDraw(t *testing.T) {
	a := newLink()
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	a.draw(dst)
	if dst.RGBAAt(10, 10).A == 0 {
		t.Fatal("expected opaque placeholder")
	}
}

func TestTouchMissLetterbox(t *testing.T) {
	a := newLink()
	a.layout = newLayout(800, 480, 720, 1600)
	a.onTouch(10, 10, 1) // must not panic
}

func TestEmbeddedIcon(t *testing.T) {
	loadIcon()
	if len(iconPix) < 32*32*4 {
		t.Fatalf("icon pix %d", len(iconPix))
	}
	p, w, h := iconABI()
	if p == nil || w != iconW || h != iconH {
		t.Fatalf("icon ABI %v %dx%d want %dx%d", p, w, h, iconW, iconH)
	}
	if uintptr(p) == uintptr(unsafe.Pointer(&iconPix[0])) {
		t.Fatal("icon must be C.malloc, not Go heap")
	}
	n := w * h * 4
	if !bytes.Equal(unsafe.Slice((*byte)(p), n), iconPix) {
		t.Fatal("icon pixels mismatch")
	}
}

func TestBlitYInvert(t *testing.T) {
	l := newLayout(800, 480, 4, 8)
	l.FlipY = true
	src := make([]byte, 8*16)
	// dest origin is visual top-right; FlipY reads that from buffer row 7
	off := 7*16 + 3*4
	src[off], src[off+3] = 255, 255
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	l.blit(dst, src, 16)
	c := dst.RGBAAt(l.DX, l.DY)
	if c.R != 255 {
		t.Fatalf("dest origin want red from flipped buffer, got %+v", c)
	}
}

func TestBlitRotate(t *testing.T) {
	l := newLayout(800, 480, 4, 8)
	src := make([]byte, 8*16)
	// 90° CCW: dest origin is panel top-right (3, 0)
	off := 0*16 + 3*4
	src[off], src[off+3] = 255, 255
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	l.blit(dst, src, 16)
	c := dst.RGBAAt(l.DX, l.DY)
	if c.R != 255 {
		t.Fatalf("dest origin want red, got %+v layout=%+v", c, l)
	}
}
