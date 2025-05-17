// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"log"
	"path/filepath"

	"goquake/math/vec"
	"goquake/snd/speaker"

	"github.com/google/uuid"
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

func (s *SndSys) startSound(start Start) {
	list, ok := s.cache[start.cache]
	if !ok {
		log.Printf("snd cache not found: %v", start.cache)
		return
	}
	if start.sfx < 0 || start.sfx >= len(list) {
		log.Printf("snd out of bounds: %v, %v", start.cache, start.sfx)
		return
	}
	pres := list[start.sfx]
	if pres == nil {
		log.Printf("snd is nil: %v", start.sfx)
		return
	}

	var ns beep.Streamer
	nss := newSound(pres)
	ns = nss
	if start.looping {
		begin := int(pres.loopStart)
		end := int(pres.loopStart + pres.loopSamples)

		var err error
		ns, err = beep.Loop2(nss, beep.LoopBetween(begin, end))
		if err != nil {
			log.Printf("%d: %v", start.sfx, err)
			return
		}
	}

	ps := &playingSound{
		masterVolume:       float64(start.volume),
		origin:             start.origin,
		entnum:             start.entityNum,
		entchannel:         start.entityChan,
		distanceMultiplier: start.attenuation / clipDistance,
		sound:              ns,
	}
	// TODO: we need to check the samplerate of the sound to match the speaker

	ps.spatialize(s.listener.ID, s.listener.Origin, s.listener.Right) // update panning
	activeSounds.add(ps)
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

func (s *SndSys) createCache(cr cacheRequest) {
	for i, s := range cr.snds {
		if i != s.ID {
			log.Printf("cache request out of order: %d, %d", i, s.ID)
			return
		}
	}

	list := make([]*pcmSound, len(cr.snds))
	for i, s := range cr.snds {
		name := filepath.Join("sound", s.Name)
		s, err := loadSFX(name)
		if err != nil {
			log.Println(err)
			continue
		}
		list[i] = s
	}
	s.cache[cr.id] = list
}

func (s *SndSys) deleteCache(id uuid.UUID) {
	delete(s.cache, id)
}

// The API

func InitSoundSystem(stop chan struct{}) *SndSys {
	if err := initSound(); err != nil {
		log.Println(err)
		return nil
	}
	s := &SndSys{
		cache:       make(map[uuid.UUID][]*pcmSound),
		shutdown:    stop,
		block:       make(chan bool),
		volume:      make(chan float32),
		stop:        make(chan Stop),
		stopAll:     make(chan bool),
		update:      make(chan listener),
		start:       make(chan Start),
		removeCache: make(chan uuid.UUID),
		addCache:    make(chan cacheRequest),
	}
	go s.run()
	return s
}

type SndSys struct {
	cache       map[uuid.UUID][]*pcmSound
	listener    listener
	shutdown    chan struct{}
	block       chan bool
	volume      chan float32
	stop        chan Stop
	stopAll     chan bool
	update      chan listener
	start       chan Start
	removeCache chan uuid.UUID
	addCache    chan cacheRequest
}

type cacheRequest struct {
	id   uuid.UUID
	snds []Sound
}

type Stop struct {
	entityNum  int
	entityChan int
}

type Start struct {
	entityNum   int
	entityChan  int
	cache       uuid.UUID
	sfx         int
	origin      vec.Vec3
	volume      float32
	attenuation float32
	looping     bool
}

type Sound struct {
	ID   int
	Name string
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
		case dc := <-s.removeCache:
			s.deleteCache(dc)
		case ac := <-s.addCache:
			s.createCache(ac)
		case start := <-s.start:
			s.startSound(start)
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
