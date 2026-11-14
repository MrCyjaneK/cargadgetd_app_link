package main

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include "cargadgetd/cg_app.h"
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"image"
	"unsafe"
)

var (
	app      = newLink()
	linkName = C.CString("Link")
	iconC    *C.uint8_t
)

func ensureIconC() {
	if iconC != nil {
		return
	}
	loadIcon()
	if len(iconPix) == 0 {
		return
	}
	n := C.size_t(len(iconPix))
	iconC = (*C.uint8_t)(C.malloc(n))
	if iconC == nil {
		return
	}
	C.memcpy(unsafe.Pointer(iconC), unsafe.Pointer(&iconPix[0]), n)
}

//export cargadgetd_app_link_icon
func cargadgetd_app_link_icon(w, h *C.int) *C.uint8_t {
	ensureIconC()
	if w != nil {
		*w = C.int(iconW)
	}
	if h != nil {
		*h = C.int(iconH)
	}
	return iconC
}

func iconABI() (unsafe.Pointer, int, int) {
	var w, h C.int
	return unsafe.Pointer(cargadgetd_app_link_icon(&w, &h)), int(w), int(h)
}

//export cargadgetd_app_link_init
func cargadgetd_app_link_init(_ *C.struct_cargadgetd_host) {}

//export cargadgetd_app_link_draw
func cargadgetd_app_link_draw(rgba *C.uint32_t, w, h, stride C.int) {
	if rgba == nil || w < 1 || h < 1 || stride < w*4 {
		return
	}
	pix := unsafe.Slice((*uint8)(unsafe.Pointer(rgba)), int(h)*int(stride))
	dst := &image.RGBA{
		Pix:    pix,
		Stride: int(stride),
		Rect:   image.Rect(0, 0, int(w), int(h)),
	}
	app.draw(dst)
}

//export cargadgetd_app_link_touch
func cargadgetd_app_link_touch(x, y, action C.int) {
	app.onTouch(int(x), int(y), int(action))
}

//export cargadgetd_app_link_pause
func cargadgetd_app_link_pause() { app.pause() }

//export cargadgetd_app_link_resume
func cargadgetd_app_link_resume() { app.resume() }

//export cargadgetd_app_link_stop
func cargadgetd_app_link_stop() { app.shutdown() }

//export cargadgetd_app_link_name
func cargadgetd_app_link_name() *C.char { return linkName }

func main() {}
