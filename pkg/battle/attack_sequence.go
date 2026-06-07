package battle

import (
	"time"

	"github.com/mcbalaam/delta/internal/engine"
)

type AttackSequence struct {
	Projectiles []ProjectileLauncher
	TimePassed  time.Duration
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
			engine.DefaultQueue.Schedule(launcher.Projectile)
			engine.DefaultUpdateQueue.Schedule(launcher.Projectile)
		} else {
			remaining = append(remaining, launcher)
		}
	}

	a.Projectiles = remaining
}
