package engine

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/systems"
)

type Entity struct {
	Transform *components.Transform
	Velocity  *components.Velocity
	Sprite    *components.Sprite
	Collider  *components.Collider

	Layer int
}

func (e *Entity) Update(dt time.Duration) {
	if e.Velocity != nil && e.Transform != nil {
		seconds := dt.Seconds()
		e.Transform.X += e.Velocity.X * seconds
		e.Transform.Y += e.Velocity.Y * seconds
	}
	if e.Sprite != nil {
		e.Sprite.Update(dt)
	}
	if e.Collider != nil && e.Transform != nil {
		e.Collider.UpdateWorldVerts(e.Transform)
	}
}

func (e *Entity) Draw(s *ebiten.Image) {
	if e.Sprite == nil || e.Transform == nil {
		return
	}
	e.Sprite.Icon.Draw(s, e.Transform.X, e.Transform.Y,
		e.Transform.ScaleX, e.Transform.ScaleY, e.Transform.Rotation)
}

func (e *Entity) Destroy() {
	queues.DefaultQueue.Unschedule(e)
	queues.DefaultUpdateQueue.Unschedule(e)
	e.Transform = nil
	e.Velocity = nil
	e.Sprite = nil
	e.Collider = nil
}

func (e *Entity) CheckCollisionsWithList(others []*Entity, signalName string) {
	for _, other := range others {
		if e.Collider != nil && other.Collider != nil {
			if e.Collider.CollidesWith(other.Collider) {
				systems.MasterSignalBus.Emit(signalName, e, other)
			}
		}
	}
}

func DrawSpriteOnCollider(screen *ebiten.Image, icon *render.AnimatedIcon, transform *components.Transform, collider *components.Collider) {
	if icon == nil || transform == nil || collider == nil {
		return
	}
	offsetVec := components.Vec{X: collider.OffsetX, Y: collider.OffsetY}
	transformedOffset := components.TransformPoint(offsetVec, transform.ScaleX, transform.ScaleY, transform.Rotation, 0, 0)
	centerX := transform.X - (collider.Width*transform.ScaleX)/2
	centerY := transform.Y - (collider.Height*transform.ScaleY)/2
	icon.Draw(screen, centerX+transformedOffset.X, centerY+transformedOffset.Y,
		transform.ScaleX, transform.ScaleY, transform.Rotation)
}
