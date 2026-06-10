package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/game"
	"github.com/mcbalaam/delta/internal/assets"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
)

func init() {
	assets.ProcessFonts()
}

func mustRegisterSound(sp *sound.SoundPlayer, path, name string) {
	if err := sp.RegisterNewSound(path, name); err != nil {
		log.Fatalf("sound %q: %v", name, err)
	}
}

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Delta")

	soundPlayer, err := sound.NewSoundPlayer(44000)
	if err != nil {
		log.Fatalf("sound: %v", err)
	}
	defer soundPlayer.Shutdown()

	mustRegisterSound(soundPlayer, "media/sound/snd_text.wav", "snd_text")
	mustRegisterSound(soundPlayer, "media/sound/snd_text2.wav", "snd_text2")
	mustRegisterSound(soundPlayer, "media/sound/snd_squeak.wav", "squeak")
	mustRegisterSound(soundPlayer, "media/sound/snd_select.wav", "select")
	mustRegisterSound(soundPlayer, "media/sound/snd_smallswing.wav", "smallswing")
	mustRegisterSound(soundPlayer, "media/sound/battle.wav", "battle")

	textEngine := &engine.TextEngine{
		FontsLoaded: make(map[string]render.AnimatedIcon),
	}

	g := game.NewGame(soundPlayer, textEngine)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
