package battle

import (
	"math"
	"math/rand"
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
	OnLaunch   func(*Projectile, float64, float64)
}

func (a *AttackSequence) Update(dt time.Duration, soulX, soulY float64) {
	a.TimePassed += dt

	var remaining []ProjectileLauncher

	for _, launcher := range a.Projectiles {
		if a.TimePassed >= launcher.LaunchAt {
			if launcher.OnLaunch != nil {
				launcher.OnLaunch(launcher.Projectile, soulX, soulY)
			}
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

// NewHomingAttack creates an attack where projectiles spawn one at a time
// from random arena-edge positions, aim at the soul, and fly straight,
// leaving a non-damaging trail behind.
func NewHomingAttack(icon *render.AnimatedIcon, count int, spawnInterval time.Duration, speed float64, damage float64, lifetime time.Duration) *AttackSequence {
	seq := &AttackSequence{
		Projectiles: make([]ProjectileLauncher, 0, count),
	}

	for i := 0; i < count; i++ {
		x, y := randomArenaEdgePosition()

		proj := &Projectile{
			Lifetime: lifetime,
			Damage:   damage,
		}
		proj.Transform = &components.Transform{X: x, Y: y, ScaleX: 2, ScaleY: 2}
		proj.Velocity = &components.Velocity{}
		proj.Sprite = &components.Sprite{Icon: icon}
		proj.Collider = components.NewCollider(14, 8, 0, 0)
		proj.Layer = queues.LayerEntity

		var trailTimer time.Duration

		proj.Behavior = func(p *Projectile, dt time.Duration) {
			if p.Transform == nil {
				return
			}
			p.Entity.Update(dt)

			if p.Sprite != nil && p.Sprite.Icon != nil {
				p.Sprite.Icon.Update(dt)
			}

			trailTimer += dt
			if trailTimer >= 100*time.Millisecond {
				trailTimer = 0
				if p.Transform != nil && p.Sprite != nil && p.Sprite.Icon != nil {
					trail := &TrailParticle{
						X:        p.Transform.X,
						Y:        p.Transform.Y,
						ScaleX:   2,
						ScaleY:   2,
						Icon:     p.Sprite.Icon,
						Lifetime: 500 * time.Millisecond,
					}
					queues.DefaultQueue.ScheduleAt(trail, queues.LayerEntity-1)
					queues.DefaultUpdateQueue.Schedule(trail)
				}
			}
		}

		launchIdx := i
		seq.Projectiles = append(seq.Projectiles, ProjectileLauncher{
			LaunchAt:   time.Duration(i) * spawnInterval,
			Projectile: proj,
			OnLaunch: func(p *Projectile, soulX, soulY float64) {
				dx := soulX - p.Transform.X
				dy := soulY - p.Transform.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > 0 {
					p.Velocity.X = (dx / dist) * speed
					p.Velocity.Y = (dy / dist) * speed
					p.Transform.Rotation = math.Atan2(dy, dx)
				}
				_ = launchIdx
			},
		})
	}

	return seq
}

func randomArenaEdgePosition() (float64, float64) {
	angle := rand.Float64() * 2 * math.Pi
	radius := 420.0
	return 640 + math.Cos(angle)*radius, 380 + math.Sin(angle)*radius
}
