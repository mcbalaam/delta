package engine

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Camera struct {
	X, Y float64
	Zoom float64
}

func NewCamera() *Camera {
	return &Camera{Zoom: 1.0}
}

func (c *Camera) Apply(screen *ebiten.Image, draw func(*ebiten.Image)) {
	if c.Zoom == 1.0 && c.X == 0 && c.Y == 0 {
		draw(screen)
		return
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	off := ebiten.NewImage(w, h)
	draw(off)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(c.Zoom, c.Zoom)
	op.GeoM.Translate(float64(w)/2+c.X, float64(h)/2+c.Y)
	screen.DrawImage(off, op)
}
