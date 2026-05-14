package game

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
)

type Projectile struct {
	engine.RigidObject

	Damage   float64
	Fragile  bool
	Lifetime time.Duration
}

func NewProjectile(posx, posy, velx, vely, scalex, scaley, rotation float64, icon render.AnimatedIcon, width, height, xoffset, yoffset, damage float64, fragile bool, lifetime time.Duration) *Projectile {
	return &Projectile{
		RigidObject: *engine.NewRigidObject(posx, posy, velx, vely, scalex, scaley, rotation, icon, width, height, xoffset, yoffset),
		Damage:      damage,
		Fragile:     fragile,
		Lifetime:    lifetime,
	}
}

func (p *Projectile) Update(dt time.Duration) {
	p.RigidObject.Update(dt)
}

func (p *Projectile) Draw(s *ebiten.Image) {
	p.RigidObject.Draw(s)
}
