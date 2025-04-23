// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"log"
	"path/filepath"

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

const Local = -1

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

func (s *SndSys) startSound(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3,
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
	// TODO: we need to check the samplerate of the sound to match the speaker

	ps.spatialize(s.listener.ID, s.listener.Origin, s.listener.Right) // update panning
	activeSounds.add(entnum, entchannel, ps)
	speaker.Play(ps)

	// TODO: how/when to remove sounds from activeSounds?
}

func (s *SndSys) stopSound(entnum, entchannel int) {
	// why does the server know which channel to stop
	activeSounds.stop(entnum, entchannel)
}

func (s *SndSys) stopAllSound() {
	speaker.Clear()
	activeSounds = newASounds()
}

type listener struct {
	Origin vec.Vec3
	Right  vec.Vec3
	ID     int
}

func (s *SndSys) updateListener(l listener) {
	// update the direction and distance to all sound sources
	s.listener = l
	activeSounds.update(s.listener.ID, s.listener.Origin, s.listener.Right)
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

// The API

func InitSoundSystem(stop chan struct{}) *SndSys {
	if err := initSound(); err != nil {
		log.Println(err)
		return nil
	}
	s := &SndSys{
		shutdown: stop,
		block:    make(chan bool),
		volume:   make(chan float32),
		stop:     make(chan Stop),
		stopAll:  make(chan bool),
		update:   make(chan listener),
		start:    make(chan Start),
	}
	go s.run()
	return s
}

type SndSys struct {
	cache    cache[*pcmSound]
	listener listener
	shutdown chan struct{}
	block    chan bool
	volume   chan float32
	stop     chan Stop
	stopAll  chan bool
	update   chan listener
	start    chan Start
}

type Stop struct {
	entityNum  int
	entityChan int
}

type Start struct {
	entityNum   int
	entityChan  int
	sfx         int
	origin      vec.Vec3
	volume      float32
	attenuation float32
	looping     bool
}

func (s *SndSys) run() {
	for {
		select {
		case <-s.shutdown:
			speaker.Close()
			return
		case b := <-s.block:
			if b {
				speaker.Suspend()
			} else {
				speaker.Resume()
			}
		case v := <-s.volume:
			speaker.SetVolume(float64(v))
		case stop := <-s.stop:
			s.stopSound(stop.entityNum, stop.entityChan)
		case <-s.stopAll:
			s.stopAllSound()
		case l := <-s.update:
			s.updateListener(l)
		case start := <-s.start:
			s.startSound(start.entityNum, start.entityChan, start.sfx, start.origin, start.volume, start.attenuation, start.looping)
		}
	}
}

func (s *SndSys) Stop(entnum, entchannel int) {
	if s == nil {
		return
	}
	s.stop <- Stop{
		entityNum:  entnum,
		entityChan: entchannel,
	}
}
func (s *SndSys) StopAll() {
	if s == nil {
		return
	}
	s.stopAll <- true
}

func (s *SndSys) Update(id int, origin vec.Vec3, right vec.Vec3) {
	if s == nil {
		return
	}
	s.update <- listener{
		ID:     id,
		Origin: origin,
		Right:  right,
	}
}

// This should not exist but overall shutdown is to broken
func (s *SndSys) Shutdown() {
	if s == nil {
		return
	}
	speaker.Close()
}
func (s *SndSys) Unblock() {
	if s == nil {
		return
	}
	s.block <- false
}
func (s *SndSys) Block() {
	if s == nil {
		return
	}
	s.block <- true
}
func (s *SndSys) SetVolume(v float32) {
	if s == nil {
		return
	}
	s.volume <- v
}
