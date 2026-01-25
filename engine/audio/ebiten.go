package audio

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100

// EbitenAudioBackend implements audio playback using Ebiten
type EbitenAudioBackend struct {
	context *audio.Context
	players map[string]*playerState
	nextID  int
	mu      sync.Mutex
}

type playerState struct {
	player *audio.Player
	loop   bool
}

// NewEbitenAudioBackend creates a new Ebiten audio backend
func NewEbitenAudioBackend() *EbitenAudioBackend {
	return &EbitenAudioBackend{
		context: audio.NewContext(sampleRate),
		players: make(map[string]*playerState),
	}
}

// Play plays audio from raw data and returns a player ID
func (b *EbitenAudioBackend) Play(data []byte, format string, volume float64, loop bool) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.context == nil {
		b.context = audio.NewContext(sampleRate)
	}

	// Decode audio based on format
	var stream io.Reader
	var err error

	switch format {
	case "wav":
		stream, err = wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	case "mp3":
		stream, err = mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	case "ogg":
		stream, err = vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	default:
		// Try to detect format from data
		stream, err = b.detectAndDecode(data)
	}

	if err != nil {
		return "", fmt.Errorf("failed to decode audio: %w", err)
	}

	// Create player
	player, err := b.context.NewPlayer(stream)
	if err != nil {
		return "", fmt.Errorf("failed to create player: %w", err)
	}

	// Set volume
	player.SetVolume(volume)

	// Generate ID
	b.nextID++
	id := fmt.Sprintf("player_%d", b.nextID)

	// Store player
	b.players[id] = &playerState{
		player: player,
		loop:   loop,
	}

	// Start playing
	player.Play()

	return id, nil
}

// PlayFromClip plays an AudioClip
func (b *EbitenAudioBackend) PlayFromClip(clip *AudioClip, volume float64, loop bool) (string, error) {
	if clip == nil || len(clip.Data) == 0 {
		return "", fmt.Errorf("invalid audio clip")
	}
	return b.Play(clip.Data, clip.Format, volume, loop)
}

// Stop stops a player by ID
func (b *EbitenAudioBackend) Stop(playerID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ps, ok := b.players[playerID]; ok {
		ps.player.Pause()
		ps.player.Close()
		delete(b.players, playerID)
	}
}

// StopAll stops all players
func (b *EbitenAudioBackend) StopAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for id, ps := range b.players {
		ps.player.Pause()
		ps.player.Close()
		delete(b.players, id)
	}
}

// SetVolume sets volume for a player
func (b *EbitenAudioBackend) SetVolume(playerID string, volume float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ps, ok := b.players[playerID]; ok {
		ps.player.SetVolume(volume)
	}
}

// IsPlaying checks if a player is currently playing
func (b *EbitenAudioBackend) IsPlaying(playerID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ps, ok := b.players[playerID]; ok {
		return ps.player.IsPlaying()
	}
	return false
}

// Update should be called each frame to handle looping
func (b *EbitenAudioBackend) Update() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for id, ps := range b.players {
		if !ps.player.IsPlaying() {
			if ps.loop {
				// Rewind and play again
				ps.player.Rewind()
				ps.player.Play()
			} else {
				// Remove finished non-looping players
				ps.player.Close()
				delete(b.players, id)
			}
		}
	}
}

// detectAndDecode tries to detect audio format and decode
func (b *EbitenAudioBackend) detectAndDecode(data []byte) (io.Reader, error) {
	// Try WAV first (starts with RIFF)
	if len(data) > 4 && string(data[:4]) == "RIFF" {
		return wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	}

	// Try MP3 (starts with ID3 or 0xFF 0xFB)
	if len(data) > 3 && (string(data[:3]) == "ID3" || (data[0] == 0xFF && (data[1]&0xE0) == 0xE0)) {
		return mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	}

	// Try OGG (starts with OggS)
	if len(data) > 4 && string(data[:4]) == "OggS" {
		return vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	}

	// Default to WAV
	return wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
}

// LoadAudioClip loads an audio clip from file data
func LoadAudioClip(name string, data []byte, format string) *AudioClip {
	return &AudioClip{
		Name:   name,
		Data:   data,
		Format: format,
	}
}
