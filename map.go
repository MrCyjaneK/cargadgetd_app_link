package main

import "image"

// Layout turns the portrait framebuffer 90° CCW into the landscape HU.
// Landscape lock draws the phone UI 90° CW inside that buffer, so CCW
// comes out upright and stays that way.
type Layout struct {
	PanelW, PanelH int
	DX, DY, DW, DH int
	FlipY          bool
}

func newLayout(huW, huH, panelW, panelH int) Layout {
	l := Layout{PanelW: panelW, PanelH: panelH}
	if huW < 1 || huH < 1 || panelW < 1 || panelH < 1 {
		return l
	}
	rotW, rotH := panelH, panelW // 90° CCW
	if rotW*huH < rotH*huW {
		l.DH = huH
		l.DW = rotW * huH / rotH
	} else {
		l.DW = huW
		l.DH = rotH * huW / rotW
	}
	if l.DW < 1 {
		l.DW = 1
	}
	if l.DH < 1 {
		l.DH = 1
	}
	l.DX = (huW - l.DW) / 2
	l.DY = (huH - l.DH) / 2
	return l
}

func (l Layout) inDest(hx, hy int) bool {
	return hx >= l.DX && hx < l.DX+l.DW && hy >= l.DY && hy < l.DY+l.DH
}

// HUToPanel inverts 90° CCW + letterbox. ok is false for letterbox taps.
func (l Layout) HUToPanel(hx, hy int) (px, py int, ok bool) {
	if l.DW < 1 || l.DH < 1 || !l.inDest(hx, hy) {
		return 0, 0, false
	}
	rx := (hx - l.DX) * l.PanelH / l.DW
	ry := (hy - l.DY) * l.PanelW / l.DH
	px = l.PanelW - 1 - ry
	py = rx
	if px < 0 || py < 0 || px >= l.PanelW || py >= l.PanelH {
		return 0, 0, false
	}
	return px, py, true
}

func (l Layout) blit(dst *image.RGBA, src []byte, stride int) {
	if dst == nil || len(src) == 0 || stride < 4 || l.DW < 1 || l.DH < 1 {
		return
	}
	b := dst.Bounds()
	for hy := l.DY; hy < l.DY+l.DH; hy++ {
		if hy < b.Min.Y || hy >= b.Max.Y {
			continue
		}
		for hx := l.DX; hx < l.DX+l.DW; hx++ {
			if hx < b.Min.X || hx >= b.Max.X {
				continue
			}
			px, py, ok := l.HUToPanel(hx, hy)
			if !ok {
				continue
			}
			sy := py
			if l.FlipY {
				sy = l.PanelH - 1 - py
			}
			off := sy*stride + px*4
			if off < 0 || off+4 > len(src) {
				continue
			}
			di := dst.PixOffset(hx, hy)
			copy(dst.Pix[di:di+4], src[off:off+4])
		}
	}
}
