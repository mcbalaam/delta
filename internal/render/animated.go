package render

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/mcbalaam/delta/internal/types"
)

type Frame struct {
	Image *ebiten.Image
	Time  time.Duration
}

type IconState struct {
	Name         string
	CurrentFrame int
	Frames       []Frame
	Mode         types.AnimationMode
	dir          int // +1 or -1
	elapsed      time.Duration
}

type AnimatedIcon struct {
	CurrentState *IconState
	IconStates   map[string]*IconState
}

func NewAnimatedIconFromPath(path string, stateKey string) (*AnimatedIcon, error) {
	if err := DefaultManager.CacheIconStates(path); err != nil {
		return nil, fmt.Errorf("cache icon states: %w", err)
	}

	st, err := DefaultManager.GetIconState(stateKey)
	if err != nil {
		return nil, fmt.Errorf("get icon state %q: %w", stateKey, err)
	}

	st.CurrentFrame = 0
	st.elapsed = 0
	st.dir = 1

	iconStates := map[string]*IconState{
		st.Name: &st,
	}

	return &AnimatedIcon{
		CurrentState: &st,
		IconStates:   iconStates,
	}, nil
}

func (a *AnimatedIcon) Update(dt time.Duration) {
	s := a.CurrentState
	if s == nil || len(s.Frames) == 0 {
		return
	}
	s.elapsed += dt
	if s.CurrentFrame < 0 {
		s.CurrentFrame = 0
	}
	for s.Frames[s.CurrentFrame].Time > 0 && s.elapsed >= s.Frames[s.CurrentFrame].Time {
		s.elapsed -= s.Frames[s.CurrentFrame].Time
		switch s.Mode {
		case types.AnimationModeLoop:
			s.CurrentFrame++
			if s.CurrentFrame >= len(s.Frames) {
				s.CurrentFrame = 0
			}
		case types.AnimationModeOnce:
			if s.CurrentFrame < len(s.Frames)-1 {
				s.CurrentFrame++
			} else {
				s.elapsed = 0
				return
			}
		case types.AnimationModePingPong:
			if s.dir == 0 {
				s.dir = 1
			}
			s.CurrentFrame += s.dir
			if s.CurrentFrame >= len(s.Frames) {
				s.CurrentFrame = len(s.Frames) - 2
				s.dir = -1
			}
			if s.CurrentFrame < 0 {
				s.CurrentFrame = 1
				s.dir = 1
			}
		default:
			s.CurrentFrame++
			if s.CurrentFrame >= len(s.Frames) {
				s.CurrentFrame = 0
			}
		}
	}
}

func (a *AnimatedIcon) Draw(screen *ebiten.Image, x, y float64) {
	s := a.CurrentState
	if s == nil || len(s.Frames) == 0 {
		return
	}
	frame := s.Frames[s.CurrentFrame]
	if frame.Image == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(frame.Image, op)
}

func (a *AnimatedIcon) SetIconState(state string) error {
	ns, ok := a.IconStates[state]
	if !ok {
		return fmt.Errorf("state %q not found", state)
	}

	ns.CurrentFrame = 0
	ns.elapsed = 0
	ns.dir = 1
	a.CurrentState = ns
	return nil
}
