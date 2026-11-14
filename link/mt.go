//go:build linux

package link

import (
	"encoding/binary"
)

const (
	evSyn = 0
	evKey = 1
	evAbs = 3

	synReport = 0

	btnTouch = 0x14a

	absX            = 0x00
	absY            = 0x01
	absMTSlot       = 0x2f
	absMTPositionX  = 0x35
	absMTPositionY  = 0x36
	absMTTrackingID = 0x39

	inputEventSize = 24
)

// AppendEV appends one 64-bit input_event (timeval zeroed).
func AppendEV(dst []byte, typ, code uint16, value int32) []byte {
	var ev [inputEventSize]byte
	binary.LittleEndian.PutUint16(ev[16:18], typ)
	binary.LittleEndian.PutUint16(ev[18:20], code)
	binary.LittleEndian.PutUint32(ev[20:24], uint32(value))
	return append(dst, ev[:]...)
}

func appendSYN(dst []byte) []byte {
	return AppendEV(dst, evSyn, synReport, 0)
}

// Down is type-B MT press on slot 0.
func Down(x, y, tracking int32) []byte {
	var b []byte
	b = AppendEV(b, evAbs, absMTSlot, 0)
	b = AppendEV(b, evAbs, absMTTrackingID, tracking)
	b = AppendEV(b, evAbs, absMTPositionX, x)
	b = AppendEV(b, evAbs, absMTPositionY, y)
	b = AppendEV(b, evAbs, absX, x)
	b = AppendEV(b, evAbs, absY, y)
	b = AppendEV(b, evKey, btnTouch, 1)
	return appendSYN(b)
}

// Move is type-B MT motion on slot 0.
func Move(x, y int32) []byte {
	var b []byte
	b = AppendEV(b, evAbs, absMTSlot, 0)
	b = AppendEV(b, evAbs, absMTPositionX, x)
	b = AppendEV(b, evAbs, absMTPositionY, y)
	b = AppendEV(b, evAbs, absX, x)
	b = AppendEV(b, evAbs, absY, y)
	return appendSYN(b)
}

// Up is type-B MT release on slot 0.
func Up() []byte {
	var b []byte
	b = AppendEV(b, evAbs, absMTSlot, 0)
	b = AppendEV(b, evAbs, absMTTrackingID, -1)
	b = AppendEV(b, evKey, btnTouch, 0)
	return appendSYN(b)
}
