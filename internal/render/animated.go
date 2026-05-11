package render

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/mcbalaam/delta/internal/types"
)

type AnimatedIcon struct {
	CurrentState      IconState
	currentStateIndex float64
	IconStates        []IconState
}

type IconState struct {
	Name         string
	CurrentFrame Frame
	Frames       []Frame
	Mode         types.AnimationMode
}

type Frame struct {
	Image ebiten.Image
	Time  time.Duration
}
