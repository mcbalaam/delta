package sound

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type SoundPlayer struct {
	Sounds map[string]*Sound
	mu     sync.Mutex
}

type Sound struct {
	Path   string
	Format beep.Format
}

func NewSoundPlayer() *SoundPlayer {
	return &SoundPlayer{Sounds: make(map[string]*Sound)}
}

func InitSpeaker(format beep.Format) {
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
}

func (s *SoundPlayer) RegisterNewSound(path string, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	streamer.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Sounds[name] != nil {
		return fmt.Errorf("sound exists: %s", name)
	}

	s.Sounds[name] = &Sound{
		Path:   path,
		Format: format,
	}
	return nil
}

func (s *SoundPlayer) PlaySound(name string) error {
	s.mu.Lock()
	sound, ok := s.Sounds[name]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no sound: %s", name)
	}
	return sound.Play()
}

func (s *Sound) Play() error {
	f, err := os.Open(s.Path)
	if err != nil {
		return err
	}
	streamer, _, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return err
	}
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		streamer.Close()
		f.Close()
	})))
	return nil
}
