//go:build linux

package link

import (
	"encoding/binary"
	"testing"
)

func TestDownMoveUp(t *testing.T) {
	d := Down(10, 20, 1)
	if len(d)%24 != 0 || len(d) < 24*6 {
		t.Fatalf("down len %d", len(d))
	}
	// last event SYN_REPORT
	if binary.LittleEndian.Uint16(d[len(d)-8:len(d)-6]) != evSyn {
		t.Fatal("down missing SYN")
	}
	m := Move(11, 21)
	if len(m)%24 != 0 {
		t.Fatalf("move len %d", len(m))
	}
	u := Up()
	if len(u)%24 != 0 {
		t.Fatalf("up len %d", len(u))
	}
	// tracking id -1
	var sawNeg bool
	for i := 0; i+24 <= len(u); i += 24 {
		code := binary.LittleEndian.Uint16(u[i+18 : i+20])
		val := int32(binary.LittleEndian.Uint32(u[i+20 : i+24]))
		if code == absMTTrackingID && val == -1 {
			sawNeg = true
		}
	}
	if !sawNeg {
		t.Fatal("up missing tracking -1")
	}
}

func TestAppendEVLayout(t *testing.T) {
	b := AppendEV(nil, evAbs, absMTPositionX, 720)
	if len(b) != 24 {
		t.Fatalf("len %d", len(b))
	}
	if binary.LittleEndian.Uint16(b[16:18]) != evAbs {
		t.Fatal("type")
	}
	if binary.LittleEndian.Uint16(b[18:20]) != absMTPositionX {
		t.Fatal("code")
	}
	if int32(binary.LittleEndian.Uint32(b[20:24])) != 720 {
		t.Fatal("value")
	}
}
