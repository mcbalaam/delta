package battle

import (
	"time"

	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
)

type Projectile struct {
	engine.RigidObject

	Damage   float64
	Fragile  bool
	Lifetime time.Duration
	Elapsed  time.Duration
	Behavior ProjectileBehavior
	Death    ProjectileDeath
}

func NewProjectile(posx, posy, velx, vely, scalex, scaley, rotation float64, icon *render.AnimatedIcon, width, height, xoffset, yoffset, damage float64, fragile bool, lifetime time.Duration) *Projectile {
	proj := &Projectile{
		RigidObject: *engine.NewRigidObject(posx, posy, velx, vely, scalex, scaley, rotation, icon, width, height, xoffset, yoffset),
		Damage:      damage,
		Fragile:     fragile,
		Lifetime:    lifetime,
	}
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
		p.RigidObject.Update(dt)
	}

	if p.Elapsed >= p.Lifetime {
		if p.Death != nil {
			p.Death(p, dt)
		}
	}
}
