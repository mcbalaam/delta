package engine

import "github.com/hajimehoshi/ebiten/v2"

type DrawQueue struct {
	ObjectsQueue []Drawable
}

type Drawable interface {
	Draw(s *ebiten.Image)
}

var DefaultQueue = &DrawQueue{}

func (d *DrawQueue) Schedule(o Drawable) {
	d.ObjectsQueue = append(d.ObjectsQueue, o)
}

func (d *DrawQueue) Execute(s *ebiten.Image) {
	for i := 0; i < len(d.ObjectsQueue); i++ {
		obj := d.ObjectsQueue[i]
		obj.Draw(s)
	}
}
