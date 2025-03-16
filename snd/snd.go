// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"log"
	"path/filepath"

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
	masterVolume       float64
	origin             vec.Vec3
	done               bool // if done it must no longer be updated
	right              float64
	left               float64
	sound              beep.Streamer
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
		samples[i][0] *= s.left * s.masterVolume
		samples[i][1] *= s.right * s.masterVolume
	}
	return n, ok
}

func (s *playingSound) Err() error {
	if s.sound == nil {
		return nil
	}
	return s.sound.Err()
}

type channel [8]*playingSound

type aSounds struct {
	// entchannel 0-7
	// 0 is ambient
	ambient []*playingSound
	sounds  map[int]channel
	local   *playingSound
}

func newASounds() *aSounds {
	return &aSounds{
		ambient: make([]*playingSound, 0),
		sounds:  make(map[int]channel),
	}
}

func (a *aSounds) add(entnum, entchannel int, p *playingSound) {
	if entchannel < 0 {
		a.local = p
		return
	}
	if entnum == 0 {
		a.ambient = append(a.ambient, p)
		return
	}
	c, ok := a.sounds[entnum]
	if !ok {
		c = channel{}
	}
	c[entchannel] = p
	a.sounds[entnum] = c
}

func (a *aSounds) stop(entnum, entchannel int) {
	c, ok := a.sounds[entnum]
	if !ok {
		return
	}
	ps := c[entchannel]
	if ps != nil {
		ps.paused = true
		ps.sound = nil
	}
	c[entchannel] = nil
	a.sounds[entnum] = c
}

func (a *aSounds) update(listener int, listenerOrigin, listenerRight vec.Vec3) {
	for _, c := range a.sounds {
		for i := range c {
			s := c[i]
			if s != nil {
				s.spatialize(listener, listenerOrigin, listenerRight)
			}
		}
	}
	for _, a := range a.ambient {
		a.spatialize(listener, listenerOrigin, listenerRight)
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

	var ns beep.Streamer
	nss := newSound(pres)
	ns = nss
	if looping {
		begin := int(pres.loopStart)
		end := int(pres.loopStart + pres.loopSamples)

		var err error
		ns, err = beep.Loop2(nss, beep.LoopBetween(begin, end))
		if err != nil {
			log.Printf("%d: %v", sfx, err)
			return
		}
	}

	ps := &playingSound{
		masterVolume:       float64(fvol),
		origin:             sndOrigin,
		entnum:             entnum,
		entchannel:         entchannel,
		distanceMultiplier: attenuation / clipDistance,
		sound:              ns,
	}
	// TODO: we need to check the samplerate of the sound to match the speeker

	ps.spatialize(s.listener.ID, s.listener.Origin, s.listener.Right) // update panning
	activeSounds.add(entnum, entchannel, ps)
	speaker.Play(ps)

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
