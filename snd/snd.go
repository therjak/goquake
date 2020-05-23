package snd

// This uses sdl2 mixer. This is not sufficient for the kind of stuff quake does.
// probably should do this with something along
// github.com/faiface/beep
// github.com/hajimehoshi/oto

import (
	"fmt"
	"github.com/therjak/goquake/math/vec"
	"log"
	"path/filepath"
	"time"

	// "github.com/hajimehoshi/oto"

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
	soundFlag       = true
	mustSampleRate  = desiredSampleRate
	mustChannelNum  = desiredChannelNum
	mustAudioFormat = uint16(desiredAudioFormat)
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

var (
	activeSounds = newASounds()
)

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
}

func soundCleanup(channel int) {
	activeSounds.soundCleanup(channel)
}

func Init(active bool) {
	if !active {
		soundFlag = false
		return
	}
	if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
		log.Println(err)
		soundFlag = false
		return
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
		log.Println(err)
		soundFlag = false
		return
	}
	frequency, format, channels, _ /*open*/, err := mix.QuerySpec()
	if err != nil {
		log.Println(err)
		soundFlag = false
		return
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
		soundFlag = false
		return
	}

	mix.AllocateChannels(128) // Observed are maps with more than 64 sounds
	mix.ChannelFinished(soundCleanup)
}

func Shutdown() {
	if !soundFlag {
		return
	}
	mix.CloseAudio()
}

func Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3,
	fvol float32, attenuation float32, looping bool) {
	if !soundFlag {
		return
	}
	if sfx < 0 || sfx >= len(soundPrecache) {
		log.Printf("asked found sound out of range %v", sfx)
		return
	}
	s := soundPrecache[sfx]

	// TODO(therjak): how to remove this allocation?
	ps := &playingSound{
		masterVolume:       fvol * mix.MAX_VOLUME,
		origin:             sndOrigin,
		entnum:             entnum,
		entchannel:         entchannel,
		distanceMultiplier: attenuation / clipDistance,
		startTime:          time.Now(),
		channel:            -1,
		sound:              s,
	}

	ps.spatialize(listener.ID, listener.Origin, listener.Right) // update panning
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
	chunk, err := newSDLSound(s)
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

func Stop(entnum, entchannel int) {
	if !soundFlag {
		return
	}
	// why does the server know which channel to stop
	activeSounds.stop(entnum, entchannel)
}

func StopAll() {
	if !soundFlag {
		return
	}
	activeSounds.stopAll()
}

var (
	listener = Listener{}
)

type Listener struct {
	Origin vec.Vec3
	Right  vec.Vec3
	ID     int
}

func Update(l Listener) {
	// update the direction and distance to all sound sources
	listener = l
	activeSounds.update(l.ID, l.Origin, l.Right)
}

// gets called when window looses focus
func Block() {
	if !soundFlag {
		return
	}
	mix.Volume(-1, 0)
}

// gets called when window gains focus
func Unblock() {
	if !soundFlag {
		return
	}
	mix.Volume(-1, int(volume*mix.MAX_VOLUME))
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

var (
	soundPrecache []*pcmSound
)

func PrecacheSound(n string) int {
	if soundFlag == false {
		return -1
	}
	name := filepath.Join("sound", n)
	for i, s := range soundPrecache {
		if s.name == name {
			return i
		}
	}
	s, err := loadSFX(name)
	if err != nil {
		log.Println(err)
		return -1
	}
	if err := s.Resample(); err != nil {
		log.Println(err)
		return -1
	}
	r := len(soundPrecache)
	soundPrecache = append(soundPrecache, s)
	return r
}

var volume float32

func SetVolume(v float32) {
	if !soundFlag {
		return
	}
	volume = v
	// this needs some init to work,
	// can only be called between mix.OpenAudio and mix.CloseAudio
	mix.Volume(-1, int(volume*mix.MAX_VOLUME))
}
