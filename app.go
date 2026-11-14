package main

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type linkApp struct {
	layout Layout
	cap    *capture
	touch  *touchOut
}

func newLink() *linkApp {
	return &linkApp{cap: &capture{}, touch: &touchOut{}}
}

func (a *linkApp) draw(dst *image.RGBA) {
	b := dst.Bounds()
	draw.Draw(dst, b, &image.Uniform{C: color.RGBA{R: 0x18, G: 0x1c, B: 0x24, A: 0xff}}, image.Point{}, draw.Src)
	src, stride, pw, ph, flipY, ok := a.cap.snapshot()
	if !ok {
		drawLabel(dst, "Link", (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2, 2, color.RGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff})
		return
	}
	a.layout = newLayout(b.Dx(), b.Dy(), pw, ph)
	a.layout.FlipY = flipY
	a.layout.blit(dst, src, stride)
}

func (a *linkApp) onTouch(x, y, action int) {
	px, py, ok := a.layout.HUToPanel(x, y)
	if !ok {
		return
	}
	a.touch.send(action, px, py)
}

func (a *linkApp) resume() {
	lockPhoneLandscape()
	a.cap.start()
	pw, ph, ok := a.cap.size()
	if ok {
		a.touch.start(pw, ph)
	}
}

func (a *linkApp) pause() { a.shutdown() }
func (a *linkApp) shutdown() {
	unlockPhoneLandscape()
	a.touch.stop()
	a.cap.stop()
}

func drawLabel(dst *image.RGBA, s string, cx, cy, scale int, col color.Color) {
	if scale < 1 {
		scale = 1
	}
	face := basicfont.Face7x13
	advance := font.MeasureString(face, s).Ceil() * scale
	height := face.Height * scale
	x := cx - advance/2
	y := cy - height/2
	tmp := image.NewRGBA(image.Rect(0, 0, advance/scale+2, face.Height+2))
	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(1, face.Ascent),
	}
	d.DrawString(s)
	for py := 0; py < tmp.Bounds().Dy(); py++ {
		for px := 0; px < tmp.Bounds().Dx(); px++ {
			_, _, _, a := tmp.At(px, py).RGBA()
			if a == 0 {
				continue
			}
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.Set(x+px*scale+sx, y+py*scale+sy, col)
				}
			}
		}
	}
}
