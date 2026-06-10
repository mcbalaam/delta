package engine

import (
	"log"

	"github.com/mcbalaam/delta/internal/sound"
)

type DialogueSession struct {
	engine        *TextEngine
	lines         []string
	currentIndex  int
	style         TextStyle
	soundPlayer   *sound.SoundPlayer
	currentDisp   *TextDisplay
	OnAllComplete func()
}

func NewDialogueSession(te *TextEngine, lines []string, style TextStyle, sp *sound.SoundPlayer) *DialogueSession {
	return &DialogueSession{
		engine:      te,
		lines:       lines,
		style:       style,
		soundPlayer: sp,
	}
}

func (s *DialogueSession) Start() {
	s.currentIndex = 0
	s.next()
}

func (s *DialogueSession) next() {
	if s.currentIndex >= len(s.lines) {
		if s.OnAllComplete != nil {
			s.OnAllComplete()
		}
		return
	}

	if s.currentDisp != nil {
		s.currentDisp.Destroy()
	}

	line := s.lines[s.currentIndex]
	s.currentIndex++

	disp, err := s.engine.DisplayText(s.style, line, s.soundPlayer, func() {
		s.next()
	})
	if err != nil {
		log.Printf("Dialogue error: %v", err)
		s.next()
		return
	}

	s.currentDisp = disp
}

func (s *DialogueSession) ForceComplete() {
	if s.currentDisp != nil {
		s.currentDisp.ForceComplete()
	}
	if s.currentIndex < len(s.lines) {
		s.currentIndex = len(s.lines)
		if s.OnAllComplete != nil {
			s.OnAllComplete()
		}
	}
}

func (s *DialogueSession) Destroy() {
	if s.currentDisp != nil {
		s.currentDisp.Destroy()
		s.currentDisp = nil
	}
}
