package engine

import (
	"time"
)

type UpdateQueue struct {
	ObjectsQueue []Updateable
}

type Updateable interface {
	Update(dt time.Duration)
}

var DefaultUpdateQueue = &UpdateQueue{}

func (d *UpdateQueue) Schedule(o Updateable) {
	d.ObjectsQueue = append(d.ObjectsQueue, o)
}

func (d *UpdateQueue) Execute(dt time.Duration) {
	for i := 0; i < len(d.ObjectsQueue); i++ {
		obj := d.ObjectsQueue[i]
		obj.Update(dt)
	}
}
