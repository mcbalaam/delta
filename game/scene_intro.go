package game

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine"
)

type IntroScene struct {
	camera        *engine.Camera
	elapsed       time.Duration
	duration      time.Duration
	startZoom     float64
	endZoom       float64
	updateContent func(time.Duration)
	drawContent   func(*ebiten.Image)
	onDone        func()
}

func NewIntroScene(
	updateContent func(time.Duration),
	drawContent func(*ebiten.Image),
	onDone func(),
) *IntroScene {
	return &IntroScene{
		camera:        engine.NewCamera(),
		duration:      2 * time.Second,
		startZoom:     1.2,
		endZoom:       1.0,
		updateContent: updateContent,
		drawContent:   drawContent,
		onDone:        onDone,
	}
}

func (s *IntroScene) Update(dt time.Duration) {
	s.updateContent(dt)

	s.elapsed += dt
	t := s.elapsed.Seconds() / s.duration.Seconds()
	if t >= 1.0 {
		s.camera.Zoom = s.endZoom
		if s.onDone != nil {
			s.onDone()
			s.onDone = nil
		}
		return
	}
	eased := 1 - math.Pow(1-t, 7)
	s.camera.Zoom = s.startZoom + (s.endZoom-s.startZoom)*eased
}

func (s *IntroScene) Draw(screen *ebiten.Image) {
	s.camera.Apply(screen, s.drawContent)
}
