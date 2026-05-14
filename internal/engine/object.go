package engine

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/render"
)

type Object struct {
	PosX     float64
	PosY     float64
	VelX     float64
	VelY     float64
	Rotation float64
	ScaleX   float64
	ScaleY   float64
	Icon     render.AnimatedIcon
}

func (o *Object) Draw(s *ebiten.Image) {
	o.Icon.Draw(s, o.PosX, o.PosY, o.ScaleX, o.ScaleY, o.Rotation)
}

func (o *Object) Update(dt time.Duration) {
	seconds := dt.Seconds()
	o.PosX += o.VelX * seconds
	o.PosY += o.VelY * seconds
	o.Icon.Update(dt)
}

func NewObject(posx, posy, velx, vely, scalex, scaley, rotation float64, icon render.AnimatedIcon) *Object {
	object := &Object{
		PosX:     posx,
		PosY:     posy,
		VelX:     velx,
		VelY:     vely,
		ScaleX:   scalex,
		ScaleY:   scaley,
		Rotation: rotation,
		Icon:     icon,
	}

	return object
}
