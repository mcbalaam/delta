package battle

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
)

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
