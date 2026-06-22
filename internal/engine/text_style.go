package engine

import "image/color"

type TextStyle struct {
	FontName     string
	StartX       float64
	StartY       float64
	ScaleX       float64
	ScaleY       float64
	FontHeight   float64
	LineSpacing  float64
	DefaultDelay float64
	Instant      bool
	CharSpacing  float64
	Color        color.Color
}

func (s TextStyle) WithInstant(instant bool) TextStyle {
	s.Instant = instant
	return s
}

var (
	StyleNarrative = TextStyle{
		FontName:     "determination",
		StartX:       60.0,
		StartY:       732.0,
		ScaleX:       0.5,
		ScaleY:       0.5,
		FontHeight:   24.0,
		LineSpacing:  90.0,
		DefaultDelay: 0.03,
		Instant:      false,
		CharSpacing:  2.0,
	}

	StyleDialogue = TextStyle{
		FontName:     "greater-determination-sb",
		StartX:       160.0,
		StartY:       340.0,
		ScaleX:       0.75,
		ScaleY:       0.75,
		FontHeight:   18.0,
		LineSpacing:  12.0,
		DefaultDelay: 0.03,
		Instant:      false,
		CharSpacing:  2.0,
	}

	StyleBubble = TextStyle{
		FontName:     "greater-determination-sb",
		StartX:       690.0,
		StartY:       230.0,
		ScaleX:       0.2,
		ScaleY:       0.2,
		FontHeight:   18.0,
		LineSpacing:  128.0,
		DefaultDelay: 0.03,
		Instant:      false,
		CharSpacing:  2.0,
		Color:        color.RGBA{0, 0, 0, 255},
	}
)
