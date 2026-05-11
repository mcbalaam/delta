package types

type AnimationMode int

const (
	ModeOnce     AnimationMode = iota
	ModeLoop     AnimationMode = iota
	ModePingPong AnimationMode = iota
)

var animationModeToString = [...]string{
	ModeOnce:     "once",
	ModeLoop:     "loop",
	ModePingPong: "pingpong",
}
