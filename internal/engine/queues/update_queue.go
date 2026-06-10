package queues

import "time"

type Updateable interface {
	Update(dt time.Duration)
}

type UpdateQueue struct {
	objects []Updateable
}

var DefaultUpdateQueue = &UpdateQueue{}

func (q *UpdateQueue) Schedule(o Updateable) {
	q.objects = append(q.objects, o)
}

func (q *UpdateQueue) Unschedule(o Updateable) {
	for i, obj := range q.objects {
		if obj == o {
			q.objects = append(q.objects[:i], q.objects[i+1:]...)
			return
		}
	}
}

func (q *UpdateQueue) Execute(dt time.Duration) {
	for _, obj := range q.objects {
		obj.Update(dt)
	}
}
