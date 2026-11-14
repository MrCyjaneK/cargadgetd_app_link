package link

import (
	"bytes"
	"testing"
)

func TestSetupRoundTrip(t *testing.T) {
	s := Setup{Width: 720, Height: 1600, Stride: 2880, Format: 0}
	got, err := ParseSetup(s.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if got != s {
		t.Fatalf("got %+v want %+v", got, s)
	}
	if s.Bytes() != 2880*1600 {
		t.Fatalf("bytes %d", s.Bytes())
	}
}

func TestSetupRejects(t *testing.T) {
	if _, err := ParseSetup([]byte{1, 2, 3}); err == nil {
		t.Fatal("short")
	}
	b := Setup{Width: 720, Height: 1600, Stride: 2880}.Marshal()
	b[0] = 0
	if _, err := ParseSetup(b); err == nil {
		t.Fatal("magic")
	}
	bad := Setup{Width: 0, Height: 10, Stride: 40}.Marshal()
	if _, err := ParseSetup(bad); err == nil {
		t.Fatal("zero width")
	}
}

func TestFrameRoundTrip(t *testing.T) {
	f := Frame{Seq: 9, Slot: 1, Flags: FlagYInvert}
	got, err := ParseFrame(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if got != f {
		t.Fatalf("%+v", got)
	}
	if !got.YInvert() {
		t.Fatal("y invert")
	}
	if _, err := ParseFrame(Frame{Slot: 2}.Marshal()); err == nil {
		t.Fatal("slot 2")
	}
}

func TestTouchRoundTrip(t *testing.T) {
	want := Touch{Action: ActionMove, X: 719, Y: 1599}
	got, err := ParseTouch(want.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%+v", got)
	}
	got, err = ReadFullTouch(bytes.NewReader(want.Marshal()))
	if err != nil || got != want {
		t.Fatalf("read %v %v", got, err)
	}
	if _, err := ParseTouch([]byte{9, 0, 0, 0, 0}); err == nil {
		t.Fatal("bad action")
	}
}
