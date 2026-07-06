package battle

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
)

// TrailParticle is a non-damaging visual trail left behind projectiles.
type TrailParticle struct {
	X        float64
	Y        float64
	ScaleX   float64
	ScaleY   float64
	Icon     *render.AnimatedIcon
	Elapsed  time.Duration
	Lifetime time.Duration
}

func (tp *TrailParticle) Update(dt time.Duration) {
	tp.Elapsed += dt
	if tp.Elapsed >= tp.Lifetime {
		queues.QDel(tp)
	}
}

func (tp *TrailParticle) Draw(s *ebiten.Image) {
	if tp.Icon == nil {
		return
	}
	alpha := 1.0 - tp.Elapsed.Seconds()/tp.Lifetime.Seconds()
	if alpha < 0 {
		alpha = 0
	}
	tp.Icon.DrawWithColorScale(s, tp.X, tp.Y, tp.ScaleX, tp.ScaleY, 0, 1, 1, 1, alpha)
}

func (tp *TrailParticle) Destroy() {
	queues.DefaultQueue.Unschedule(tp)
	queues.DefaultUpdateQueue.Unschedule(tp)
}

type Projectile struct {
	engine.Entity

	Damage   float64
	Fragile  bool
	Lifetime time.Duration
	Elapsed  time.Duration
	Behavior ProjectileBehavior
	Death    ProjectileDeath
}

func NewProjectile(posx, posy, velx, vely, scalex, scaley, rotation float64, icon *render.AnimatedIcon, width, height, xoffset, yoffset, damage float64, fragile bool, lifetime time.Duration) *Projectile {
	proj := &Projectile{
		Damage:   damage,
		Fragile:  fragile,
		Lifetime: lifetime,
	}
	proj.Transform = &components.Transform{X: posx, Y: posy, ScaleX: scalex, ScaleY: scaley, Rotation: rotation}
	proj.Velocity = &components.Velocity{X: velx, Y: vely}
	proj.Sprite = &components.Sprite{Icon: icon}
	proj.Collider = components.NewCollider(width, height, xoffset, yoffset)
	return proj
}

// A behaviour other than moving in a straight line.
type ProjectileBehavior func(p *Projectile, dt time.Duration)

// Called when the projectile's `Lifetime` runs out. Plays a fancy animation, launches more projectiles, whatever you need before the projectile is destroyed.
type ProjectileDeath func(p *Projectile, dt time.Duration)

func (p *Projectile) Update(dt time.Duration) {
	p.Elapsed += dt

	if p.Behavior != nil {
		p.Behavior(p, dt)
	} else {
		p.Entity.Update(dt)
	}

	if p.Elapsed >= p.Lifetime {
		if p.Death != nil {
			p.Death(p, dt)
		}
		queues.QDel(p)
	}
}

func (p *Projectile) Draw(s *ebiten.Image) {
	if p.Sprite == nil || p.Sprite.Icon == nil {
		return
	}
	engine.DrawSpriteOnCollider(s, p.Sprite.Icon, p.Transform, p.Collider)
}
