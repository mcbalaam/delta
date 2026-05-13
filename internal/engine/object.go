package engine

import (
	"time"

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

func (o *Object) Update(dt time.Duration) {
	o.Icon.Update(dt)
}
