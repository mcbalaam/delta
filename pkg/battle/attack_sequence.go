package battle

import (
	"time"

	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
)

type AttackSequence struct {
	Projectiles       []ProjectileLauncher
	ActiveProjectiles []*Projectile
	TimePassed        time.Duration
}

type ProjectileLauncher struct {
	LaunchAt   time.Duration
	Projectile *Projectile
}

func (a *AttackSequence) Update(dt time.Duration) {
	a.TimePassed += dt

	var remaining []ProjectileLauncher

	for _, launcher := range a.Projectiles {
		if a.TimePassed >= launcher.LaunchAt {
			if launcher.Projectile.Layer == 0 {
				launcher.Projectile.Layer = queues.LayerEntity
			}
			queues.DefaultQueue.ScheduleAt(launcher.Projectile, launcher.Projectile.Layer)
			queues.DefaultUpdateQueue.Schedule(launcher.Projectile)
			a.ActiveProjectiles = append(a.ActiveProjectiles, launcher.Projectile)
		} else {
			remaining = append(remaining, launcher)
		}
	}

	a.Projectiles = remaining
}

// NewBasicAttack creates a simple attack with projectiles flying right-to-left.
// icon is the sprite for each projectile; count is how many to spawn;
// duration is the total attack time. Projectiles are staggered evenly.
func NewBasicAttack(icon *render.AnimatedIcon, count int, duration time.Duration, damage float64) *AttackSequence {
	seq := &AttackSequence{
		Projectiles: make([]ProjectileLauncher, 0, count),
	}

	stagger := duration / time.Duration(count)
	startY := 150.0
	spacing := 400.0 / float64(count)

	for i := 0; i < count; i++ {
		y := startY + float64(i)*spacing

		proj := &Projectile{
			Lifetime: duration,
			Damage:   damage / float64(count),
		}
		proj.Transform = &components.Transform{X: 1250, Y: y, ScaleX: 2, ScaleY: 2}
		proj.Velocity = &components.Velocity{X: -120, Y: 0}
		proj.Sprite = &components.Sprite{Icon: icon}
		proj.Collider = components.NewCollider(14, 8, 0, 0)
		proj.Layer = queues.LayerEntity

		seq.Projectiles = append(seq.Projectiles, ProjectileLauncher{
			LaunchAt:   time.Duration(i) * stagger,
			Projectile: proj,
		})
	}

	return seq
}
