// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"log"
	"path/filepath"
	"time"

	"goquake/math"
	"goquake/math/vec"
	"goquake/snd/speaker"

	"github.com/gopxl/beep/v2"
)

const (
	clipDistance      = 1000.0
	desiredSampleRate = 11025
	desiredBitdepth   = 16
	desiredChannelNum = 2
)

var (
	mustSampleRate = desiredSampleRate
	mustChannelNum = desiredChannelNum
	activeSounds   = newASounds()
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
	right              float64
	left               float64
	startTime          time.Time
	sound              *sound
	paused             bool
}

func (s *playingSound) spatialize(listener int, listenerPos, listenerRight vec.Vec3) {
	if listener == s.entnum {
		s.right = 1
		s.left = 1
	} else {
		v := vec.Sub(s.origin, listenerPos)
		dist := v.Length() * s.distanceMultiplier
		v = v.Normalize()
		dot := vec.Dot(listenerRight, v)
		dist = 1.0 - dist
		lscale := (1.0 - dot) * dist
		rscale := (1.0 + dot) * dist
		s.left = math.Clamp(0, float64(lscale), 1)
		s.right = math.Clamp(0, float64(rscale), 1)
	}
}

func (s *playingSound) Stream(samples [][2]float64) (int, bool) {
	if s.sound == nil {
		return 0, false
	}
	if s.paused {
		clear(samples)
		return len(samples), true
	}
	n, ok := s.sound.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= s.left
		samples[i][1] *= s.right
	}
	return n, ok
}

func (s *playingSound) Err() error {
	if s.sound == nil {
		return nil
	}
	return s.sound.Err()
}

type aSounds struct {
	sounds map[int]*playingSound
}

func newASounds() *aSounds {
	return &aSounds{make(map[int]*playingSound)}
}

func (a *aSounds) add(p *playingSound) {
	a.sounds[p.channel] = p
}

func (a *aSounds) stop(entnum, entchannel int) {
	for id, s := range a.sounds {
		if s.entnum == entnum && s.entchannel == entchannel {
			s.paused = true
			s.sound = nil
		}
		delete(a.sounds, id)
	}
}

func (a *aSounds) update(listener int, listenerOrigin, listenerRight vec.Vec3) {
	for _, s := range a.sounds {
		s.spatialize(listener, listenerOrigin, listenerRight)
	}
	// TODO(therjak): start sounds which became audible
	//                stop sounds which are unaudible
	// ambientsounds to ambient_levels
}

func chunkSize() int {
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
}

func initSound() error {
	sr := beep.SampleRate(desiredSampleRate)
	speaker.Init(sr, chunkSize())

	return nil
}

func (s *SndSys) shutdown() {
	speaker.Close()
}

func (s *SndSys) start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3,
	fvol float32, attenuation float32, looping bool) {
	pres := s.cache.Get(sfx)
	if pres == nil {
		log.Printf("asked found sound out of range %v", sfx)
		return
	}

	ps := &playingSound{
		masterVolume:       fvol,
		origin:             sndOrigin,
		entnum:             entnum,
		entchannel:         entchannel,
		distanceMultiplier: attenuation / clipDistance,
		startTime:          time.Now(),
		channel:            -1,
		sound:              newSound(pres),
	}
	// TODO: we need to check the samplerate of the sound to match the speeker
	// TODO: Looping

	ps.spatialize(s.listener.ID, s.listener.Origin, s.listener.Right) // update panning
	speaker.Play(ps)

	activeSounds.add(ps)
	// TODO: how/when to remove sounds from activeSounds?
}

func (s *SndSys) stop(entnum, entchannel int) {
	// why does the server know which channel to stop
	activeSounds.stop(entnum, entchannel)
}

func (s *SndSys) stopAll() {
	speaker.Clear()
	activeSounds = newASounds()
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
	speaker.Suspend()
}

// gets called when window gains focus
func (s *SndSys) unblock() {
	speaker.Resume()
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
	return s.cache.Add(sfx)
}

func (s *SndSys) setVolume(v float32) {
	speaker.SetVolume(float64(v))
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
	cache    cache[*pcmSound]
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
