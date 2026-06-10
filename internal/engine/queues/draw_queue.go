package queues

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	LayerBackground = -200
	LayerArena      = -100
	LayerEntity     = 0
	LayerUI         = 100
	LayerText       = 200
	LayerOverlay    = 300
)

type Drawable interface {
	Draw(s *ebiten.Image)
}

type drawEntry struct {
	layer int
	obj   Drawable
}

type DrawQueue struct {
	entries []drawEntry
	dirty   bool
}

var DefaultQueue = &DrawQueue{}

func (d *DrawQueue) Schedule(o Drawable) {
	d.ScheduleAt(o, 0)
}

func (d *DrawQueue) ScheduleAt(o Drawable, layer int) {
	d.entries = append(d.entries, drawEntry{layer: layer, obj: o})
	d.dirty = true
}

func (d *DrawQueue) Unschedule(o Drawable) {
	for i, e := range d.entries {
		if e.obj == o {
			d.entries = append(d.entries[:i], d.entries[i+1:]...)
			return
		}
	}
}

func (d *DrawQueue) Execute(s *ebiten.Image) {
	if d.dirty {
		sort.SliceStable(d.entries, func(i, j int) bool {
			return d.entries[i].layer < d.entries[j].layer
		})
		d.dirty = false
	}
	for _, e := range d.entries {
		e.obj.Draw(s)
	}
}
