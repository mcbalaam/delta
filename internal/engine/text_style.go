package engine

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
}

func (s TextStyle) WithInstant(instant bool) TextStyle {
	s.Instant = instant
	return s
}

var (
	StyleNarrative = TextStyle{
		FontName:     "determination",
		StartX:       150.0,
		StartY:       700.0,
		ScaleX:       0.5,
		ScaleY:       0.5,
		FontHeight:   24.0,
		LineSpacing:  80.0,
		DefaultDelay: 0.03,
		Instant:      false,
	}

	StyleDialogue = TextStyle{
		FontName:     "greater_determination",
		StartX:       160.0,
		StartY:       340.0,
		ScaleX:       0.75,
		ScaleY:       0.75,
		FontHeight:   18.0,
		LineSpacing:  12.0,
		DefaultDelay: 0.03,
		Instant:      false,
	}
)
