// SPDX-License-Identifier: GPL-2.0-or-later

package snd

// This uses sdl2 mixer. This is not sufficient for the kind of stuff quake does.
// probably should do this with something along
// github.com/gopxl/beep/v2
// github.com/ebitengine/oto/v3

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"goquake/math/vec"

	// github.com/ebitengine/oto/v3

	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	clipDistance       = 1000.0
	desiredSampleRate  = 11025
	desiredBitdepth    = 16
	desiredAudioFormat = uint16(sdl.AUDIO_S16SYS)
	desiredChannelNum  = 2
)

var (
	mustSampleRate  = desiredSampleRate
	mustChannelNum  = desiredChannelNum
	mustAudioFormat = uint16(desiredAudioFormat)
	activeSounds    = newASounds()
)

type playingSound struct {
	channel    int // playing on channel
	entchannel int // entchannel
	// TODO:
	// entchannel. 0 willingly overrides, 1-7 always overrides
	// 0 auto
	// 1 weapon
	// 2 voice
	// 3 item
	// 4 body
	// 8 no phys add
	entnum             int // entnum
	distanceMultiplier float32
	masterVolume       float32
	origin             vec.Vec3
	done               bool // if done it must no longer be updated
	right              uint8
	left               uint8
	startTime          time.Time
	sound              *pcmSound
}

// beep interface:
// Len() int
// Position() int
// Seek(p int) error
// Close() error
// Error() error
// Stream(samples [][2]float64) (n int, ok bool)
//

/*
type Player struct {
	player *oto.Player
}

// NewPlayer(sampleRate,channelNum,bytesPerSample,bufferSize) (*Player, error)
// player.Close() error
// player.Write(data []byte) (int,error)

func NewPlayer() (*Player, error) {
	bufferSize := func() int {
		if desiredSampleRate <= 11025 {
			return 256
		} else if desiredSampleRate <= 22050 {
			return 512
		} else if desiredSampleRate <= 44100 {
			return 1024
		} else if desiredSampleRate <= 56000 {
			return 2048 // for 48 kHz
		}
		return 4096 // for 96 kHz
	}()

	p, err := oto.NewPlayer(desiredSampleRate, desiredChannelNum, desiredBitdepth, bufferSize)
	if err != nil {
		return nil, err
	}
	return &Player{p}, nil
}

func (p *Player) SetPanning(channel int, left float32, right float32) {
	// Panning for channel 'channel'
}

func (p *Player) SetVolume(v float32) {
	// Volume over all channels
}

func (p *Player) Play(s *Sound, entity int, origin vec.Vec3) int {
	// Returns playing channel id
	// is the input playingSound correct?
	return 0
}

func (p *Player) UpdateListenerPos(listener int, listenerPos, listenerRight vec.Vec3) {
}

func (p *Player) Close() error {
	return p.Close()
}
*/

func (s *playingSound) spatialize(listener int, listenerPos, listenerRight vec.Vec3) {
	if listener == s.entnum {
		s.right = uint8(s.masterVolume)
		s.left = uint8(s.masterVolume)
	} else {
		v := vec.Sub(s.origin, listenerPos)
		dist := v.Length() * s.distanceMultiplier
		v = v.Normalize()
		dot := vec.Dot(listenerRight, v)
		dist = 1.0 - dist
		lscale := (1.0 - dot) * dist
		rscale := (1.0 + dot) * dist
		l := s.masterVolume * lscale
		if l < 0 {
			l = 0
		} else if l > 254 {
			l = 254
		}
		s.left = uint8(l)
		r := s.masterVolume * rscale
		if r < 0 {
			r = 0
		} else if r > 254 {
			r = 254
		}
		s.right = uint8(r)
	}
}

type aSounds struct {
	sounds map[int]*playingSound
}

func newASounds() *aSounds {
	return &aSounds{make(map[int]*playingSound)}
}

func (a *aSounds) soundCleanup(channel int) {
	delete(a.sounds, channel)
}

func (a *aSounds) add(p *playingSound) {
	a.sounds[p.channel] = p
}

func (a *aSounds) stop(entnum, entchannel int) {
	for _, s := range a.sounds {
		if s.entnum == int(entnum) && s.entchannel == int(entchannel) {
			mix.HaltChannel(s.channel)
		}
	}
}

func (a *aSounds) stopAll() {
	mix.HaltChannel(-1)
}

func (a *aSounds) update(listener int, listenerOrigin, listenerRight vec.Vec3) {
	for _, s := range a.sounds {
		s.spatialize(listener, listenerOrigin, listenerRight)
		if err := mix.SetPanning(s.channel, s.left, s.right); err != nil {
			log.Println(err)
		}
	}
	// TODO(therjak): start sounds which became audible
	//                stop sounds which are unaudible
	// ambientsounds to ambient_levels
}

func soundCleanup(channel int) {
	activeSounds.soundCleanup(channel)
}

func initSound() error {
	if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
		return err
	}

	chunkSize := func() int {
		if desiredSampleRate <= 11025 {
			return 256
		} else if desiredSampleRate <= 22050 {
			return 512
		} else if desiredSampleRate <= 44100 {
			return 1024
		} else if desiredSampleRate <= 56000 {
			return 2048 /* for 48 kHz */
		}
		return 4096 /* for 96 kHz */
	}()

	if err := mix.OpenAudio(desiredSampleRate, desiredAudioFormat, desiredChannelNum, chunkSize); err != nil {
		return err
	}
	frequency, format, channels, _ /*open*/, err := mix.QuerySpec()
	if err != nil {
		return err
	}
	mustSampleRate = frequency
	mustChannelNum = channels
	mustAudioFormat = format
	// TODO: I guess the chunkSize should be updated if anything changed

	// TODO: until there is correct conversion code reject everything if not the desired format
	if mustSampleRate != desiredSampleRate ||
		mustChannelNum != desiredChannelNum ||
		mustAudioFormat != desiredAudioFormat {
		log.Println(err)
		return fmt.Errorf("Wrong samplerate")
	}

	mix.AllocateChannels(128) // Observed are maps with more than 64 sounds
	mix.ChannelFinished(soundCleanup)
	return nil
}

func (s *SndSys) shutdown() {
	mix.CloseAudio()
}

func (s *SndSys) start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3,
	fvol float32, attenuation float32, looping bool) {
	pres := s.cache.Get(sfx)
	if pres == nil {
		log.Printf("asked found sound out of range %v", sfx)
		return
	}

	// TODO(therjak): how to remove this allocation?
	ps := &playingSound{
		masterVolume:       fvol * mix.MAX_VOLUME,
		origin:             sndOrigin,
		entnum:             entnum,
		entchannel:         entchannel,
		distanceMultiplier: attenuation / clipDistance,
		startTime:          time.Now(),
		channel:            -1,
		sound:              pres,
	}

	ps.spatialize(s.listener.ID, s.listener.Origin, s.listener.Right) // update panning
	if ps.left != 0 || ps.right != 0 {
		// ignore this sound
	}

	loop := func() int {
		if !looping {
			return 0 // do not loop
		}
		// BUG: This produces loops but ignores that the 2. loop
		// should start at loopstart.
		return -1 // loop infinite
	}()
	chunk, err := newSDLSound(pres)
	if err != nil {
		log.Println(err)
		return
	}
	schan, err := chunk.Play(ps.channel, loop)
	if err != nil {
		log.Printf("Playing Channels are %v", mix.Playing(-1))
		log.Println(err)
		return
	}
	ps.channel = schan
	if err := mix.SetPanning(ps.channel, ps.left, ps.right); err != nil {
		log.Println(err)
	}
	activeSounds.add(ps)
}

func (s *SndSys) stop(entnum, entchannel int) {
	// why does the server know which channel to stop
	activeSounds.stop(entnum, entchannel)
}

func (s *SndSys) stopAll() {
	activeSounds.stopAll()
}

type listener struct {
	Origin vec.Vec3
	Right  vec.Vec3
	ID     int
}

func (s *SndSys) update(id int, origin vec.Vec3, right vec.Vec3) {
	// update the direction and distance to all sound sources
	s.listener = listener{
		Origin: origin,
		Right:  right,
		ID:     id,
	}
	activeSounds.update(id, origin, right)
}

// gets called when window looses focus
func (s *SndSys) block() {
	mix.Volume(-1, 0)
}

// gets called when window gains focus
func (s *SndSys) unblock() {
	mix.Volume(-1, int(s.volume*mix.MAX_VOLUME))
}

func newSDLSound(s *pcmSound) (*mix.Chunk, error) {
	if s.sampleRate != mustSampleRate {
		return nil, fmt.Errorf("Not desired sample rate. %v, %v", s.sampleRate, s.name)
	}
	l := s.samples * (s.bitrate / 8) * s.channelNum
	if l > len(s.pcm) {
		log.Printf("Bad sdlLoad")
		return nil, fmt.Errorf("Bad sdlLoad")
	}
	return mix.QuickLoadRAW(&s.pcm[0], uint32(l))
}

func (s *SndSys) precacheSound(n string) int {
	name := filepath.Join("sound", n)
	if i, ok := s.cache.Has(name); ok {
		return i
	}
	sfx, err := loadSFX(name)
	if err != nil {
		log.Println(err)
		return -1
	}
	if err := sfx.resample(); err != nil {
		log.Println(err)
		return -1
	}
	return s.cache.Add(sfx)
}

func (s *SndSys) setVolume(v float32) {
	s.volume = v
	// this needs some init to work,
	// can only be called between mix.OpenAudio and mix.CloseAudio
	mix.Volume(-1, int(s.volume*mix.MAX_VOLUME))
}

// The API

func InitSoundSystem(active bool) *SndSys {
	if !active {
		return nil
	}
	if err := initSound(); err != nil {
		log.Println(err)
		return nil
	}
	return &SndSys{}
}

type SndSys struct {
	cache    cache
	volume   float32
	listener listener
}

func (s *SndSys) Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3, fvol float32, attenuation float32, looping bool) {
	if s == nil {
		return
	}
	s.start(entnum, entchannel, sfx, sndOrigin, fvol, attenuation, looping)
}

func (s *SndSys) Stop(entnum, entchannel int) {
	if s == nil {
		return
	}
	s.stop(entnum, entchannel)
}
func (s *SndSys) StopAll() {
	if s == nil {
		return
	}
	s.stopAll()
}
func (s *SndSys) PrecacheSound(n string) int {
	if s == nil {
		return -1
	}
	return s.precacheSound(n)
}
func (s *SndSys) Update(id int, origin vec.Vec3, right vec.Vec3) {
	if s == nil {
		return
	}
	s.update(id, origin, right)
}
func (s *SndSys) Shutdown() {
	if s == nil {
		return
	}
	s.shutdown()
}
func (s *SndSys) Unblock() {
	if s == nil {
		return
	}
	s.unblock()
}
func (s *SndSys) Block() {
	if s == nil {
		return
	}
	s.block()
}
func (s *SndSys) SetVolume(v float32) {
	if s == nil {
		return
	}
	s.setVolume(v)
}
