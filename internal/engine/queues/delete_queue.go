package queues

type Destroyable interface {
	Destroy()
}

type DeleteQueue struct {
	objects []Destroyable
}

var DefaultDeleteQueue = &DeleteQueue{}

func QDel(obj Destroyable) {
	DefaultDeleteQueue.objects = append(DefaultDeleteQueue.objects, obj)
}

func (q *DeleteQueue) Execute() {
	for _, obj := range q.objects {
		obj.Destroy()
	}
	q.objects = q.objects[:0]
}
