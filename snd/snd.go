package snd

// This uses sdl2 mixer. This is not sufficient for the kind of stuff quake does.
// probably should do this with something along
// github.com/faiface/beep
// github.com/hajimehoshi/oto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"path/filepath"
	"quake/commandline"
	"quake/cvar"
	"quake/cvars"
	"quake/filesystem"
	"quake/math/vec"
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
	stereo             = 2
	mono               = 1
	desiredChannelNum  = stereo
)

var (
	soundFlag       = true
	mustSampleRate  = desiredSampleRate
	mustChannelNum  = desiredChannelNum
	mustAudioFormat = uint16(desiredAudioFormat)
)

type playingSound struct {
	channel            int // playing on channel
	entchannel         int // entchannel
	entnum             int // entnum
	distanceMultiplier float32
	masterVolume       float32
	origin             vec.Vec3
	done               bool // if done it must no longer be updated
	right              uint8
	left               uint8
	startTime          time.Time
	sound              *Sound
}

/*
type Player struct {
	player *oto.Player
}

// NewPlayer(sampleRate,channelNum,bytesPerSample,bufferSize) (*Player, error)
// player.Close() error
// player.Write(data []byte) (int,error)
// player.SetUnderrunCallback(f func()) (something like func() { log.Printf("You are slow") } )

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

func Init() {
	log.Printf("Sound is %v", commandline.Sound())
	if cvars.NoSound.Value() != 0 || !commandline.Sound() {
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
	mix.CloseAudio()
}

func Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3,
	fvol float32, attenuation float32, looping bool) {
	if cvars.NoSound.Value() != 0 || !soundFlag {
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
	schan, err := s.data.Play(ps.channel, loop)
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
	// why does the server know which channel to stop
	activeSounds.stop(entnum, entchannel)
}

func StopAll() {
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
	if cvars.NoSound.Value() != 0 || !soundFlag {
		return
	}
	mix.Volume(-1, 0)
}

// gets called when window gains focus
func Unblock() {
	onVolumeChange(nil)
}

type Sound struct {
	data       *mix.Chunk
	name       string
	samples    int // number of samples
	bitrate    int
	channelNum int
	loopStart  int // in samples, length/bitrate
	sampleRate int
	pcm        []byte
}

// Resample converts to 16bit stereo
func (s *Sound) Resample() error {
	// TODO: this should convert to mustAudioFormat, mustSampleRate and mustChannelNum
	// for now convert to 16 bit stereo (desired format)
	// TODO: convert to 11025 sampleRate
	if s.sampleRate != mustSampleRate {
		return fmt.Errorf("Not desired sample rate. %v, %v", s.sampleRate, s.name)
	}
	if s.bitrate == 16 && s.channelNum == 2 {
		return nil
	}
	if s.bitrate == 16 && s.channelNum == 1 {
		return s.resample16Mono()
	}
	if s.bitrate == 8 && s.channelNum == 2 {
		return s.resample8Stereo()
	}
	if s.bitrate == 8 && s.channelNum == 1 {
		return s.resample8Mono()
	}
	return fmt.Errorf("Unsupported sound format: %v", s.name)
}

func (s *Sound) resample16Mono() error {
	newPCM := make([]byte, len(s.pcm)*2)
	for i := 0; i < len(s.pcm); i += 2 {
		newPCM[i*2] = s.pcm[i]
		newPCM[i*2+1] = s.pcm[i+1]
		newPCM[i*2+2] = s.pcm[i]
		newPCM[i*2+3] = s.pcm[i+1]
	}
	s.pcm = newPCM
	s.channelNum = 2
	return nil
}

func (s *Sound) resample8Stereo() error {
	newPCM := make([]byte, len(s.pcm)*2)
	for i := 0; i < len(s.pcm); i++ {
		v := (int16(s.pcm[i]) - 128)
		newPCM[i*2] = 0
		newPCM[i*2+1] = byte(v)
	}
	s.pcm = newPCM
	s.bitrate = 16
	return nil
}

func (s *Sound) resample8Mono() error {
	newPCM := make([]byte, len(s.pcm)*4)
	for i := 0; i < len(s.pcm); i++ {
		v := (int16(s.pcm[i]) - 128)
		newPCM[i*4] = 0
		newPCM[i*4+1] = byte(v)
		newPCM[i*4+2] = 0
		newPCM[i*4+3] = byte(v)
	}
	s.pcm = newPCM
	s.channelNum = 2
	s.bitrate = 16
	return nil
}

func (s *Sound) sdlLoad() error {
	l := s.samples * (s.bitrate / 8) * s.channelNum
	if l > len(s.pcm) {
		log.Printf("Bad sdlLoad")
		return fmt.Errorf("Bad sdlLoad")
	}
	c, err := mix.QuickLoadRAW(&s.pcm[0], uint32(l))
	if err != nil {
		return err
	}
	s.data = c
	return nil
}

var (
	soundPrecache []*Sound
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
	if err := s.sdlLoad(); err != nil {
		log.Println(err)
		return -1
	}
	r := len(soundPrecache)
	soundPrecache = append(soundPrecache, s)
	return r
}

type waveHeader struct {
	ID       [4]byte // better be RIFF
	Size     uint32  // file size - 8
	RiffType [4]byte // better be WAVE
}

type chunk struct {
	ID   [4]byte
	Size int
	Body []byte
}

func loadSFX(filename string) (*Sound, error) {
	mem, err := filesystem.GetFileContents(filename)
	if err != nil {
		return nil, fmt.Errorf("Could not load file %v: %v", filename, err)
	}
	// 12 would be enough for the header, but the file would contain no data
	// we need 12 + 8 for the next chunk
	if len(mem) < 20 {
		return nil, fmt.Errorf("file with length < 20, %v", filename)
	}

	header := getHeader(mem)
	if header.ID != [4]byte{'R', 'I', 'F', 'F'} ||
		header.RiffType != [4]byte{'W', 'A', 'V', 'E'} {
		return nil, fmt.Errorf("file is not a RIFF wave file")
	}
	if int(header.Size) != len(mem)-8 {
		log.Println("wave file length in header seems off")
	}

	chunks := []chunk{}
	chunkSize := 12 // last read chunk has size of...
	chunkMem := mem
	for len(chunkMem) >= chunkSize+8 { // we have at least one chunk left to read
		chunkMem = chunkMem[chunkSize:] // we know we have at least 8 byte available
		var id [4]byte
		copy(id[:], chunkMem)
		// size of the chunk without id (4byte) and length (4byte) info will come now
		size := int(binary.LittleEndian.Uint32(chunkMem[4:]))
		chunkSize = size + 8
		if chunkSize > len(chunkMem) {
			//			fmt.Printf("Got broken chunk in file %v, %v, %v, %v, %v\n",
			//				filename, chunkSize, len(chunkMem), string(id[:]), string(chunkMem[8:12]))
			continue
		}
		chunks = append(chunks, chunk{
			ID:   id,
			Size: size,
			Body: chunkMem[8:chunkSize],
		})
		if chunkSize%2 != 0 {
			// spec says chunks are WORD aligned (2,4,6,8,10,...) with 0 padding but 'size' does not include padding.
			chunkSize = chunkSize + 1 // (size + 1 ) &^ 1
		}
	}
	output := &Sound{name: filename}

	gotFMT := 0
	for _, c := range chunks {
		if c.ID == [4]byte{'f', 'm', 't', ' '} {
			if c.Size < 16 {
				return nil, fmt.Errorf("Got broken fmt chunk, %v", filename)
			}
			f := readFMT(c)
			if f.CompressionCode != 0x0001 {
				return nil, fmt.Errorf("Invalid sound format: %v, %v", f.CompressionCode, filename)
			}
			if f.ChannelNum != mono && f.ChannelNum != stereo {
				return nil, fmt.Errorf("Invalid number of sound channels: %v, %v", f.ChannelNum, filename)
			}
			output.channelNum = int(f.ChannelNum)
			if f.SignificantBitsPerSample != 8 && f.SignificantBitsPerSample != 16 {
				return nil, fmt.Errorf("Invalid sound bitrate: %v", f.SignificantBitsPerSample)
			}
			output.bitrate = int(f.SignificantBitsPerSample)
			output.sampleRate = int(f.SampleRate)
			gotFMT += 1
		}
	}
	if gotFMT != 1 {
		return nil, fmt.Errorf("Invalid number of fmt blocks: %v", gotFMT)
	}

	for _, c := range chunks {
		if c.ID == [4]byte{'d', 'a', 't', 'a'} {
			output.samples = c.Size / (output.bitrate / 8)
			output.pcm = c.Body
			break // We only support a single data chunk, so reading the first is enough
		}
	}

	output.loopStart = -1
	for i, c := range chunks {
		if c.ID == [4]byte{'c', 'u', 'e', ' '} {
			// https://sites.google.com/site/musicgapi/technical-documents/wav-file-format#cue
			// off 0x00: 4byte 'cue '
			// off 0x04: 4byte chunk data size
			// off 0x08: 4byte num cue points
			// off 0x0c: list of points
			// each cue points:
			// off 0x00: 4byte ID
			// off 0x04: 4byte Position
			// off 0x08: 4byte Data chunk ID
			// off 0x0c: 4byte Chunk start
			// off 0x10: 4byte Block start
			// off 0x14: 4byte Sample Offset

			// It would be correct to first check if the number of cue points is at least 1
			// but this check was not done in the original, so only check for the length.
			const sampleOffsetOffset = 0x14 + 4
			if c.Size < sampleOffsetOffset+4 {
				// offset Sample Offset + length +  length of num cue points size
				return nil, fmt.Errorf("Got faulty cue chunk")
			}
			loopStart := binary.LittleEndian.Uint32(c.Body[sampleOffsetOffset:])
			if int(loopStart) >= output.samples {
				// check here as well, we might not get a LIST chunk
				return nil, fmt.Errorf("Got loop start beyond the sound end")
			}
			output.loopStart = int(loopStart)
			if output.loopStart < 0 {
				log.Printf("negative loop start")
			}
			if len(chunks) >= i+1 {
				next := chunks[i+1]
				if next.ID == [4]byte{'L', 'I', 'S', 'T'} {
					// off 0x00: 4byte 'LIST'
					// off 0x04: 4byte chunk data size
					// off 0x08: 4byte list type id, probably 'adtl' but not directly checked
					// off 0x0c: data, depending on list type id
					// adtl data:
					// off 0x00: 4byte sub chunk id
					// off 0x04: 4byte size
					// off 0x08: 4byte id of the relevant cue point
					// off 0x0c: 4byte sample length
					// off 0x10: 4byte purpose id
					// purpose id = (chunk start + 28) should be 'mark'
					// sample length = (chunk start + 24) is wanted nr as Uint32

					// the first 8 bytes are still part of the chunk metadata, so just add
					// the list type id length to adtl data
					const purposeOffset = 0x10 + 4
					if next.Size >= purposeOffset+4 {
						if string(next.Body[purposeOffset:][:4]) == "mark" {
							const sampleLengthOffset = 0x0c + 4
							loopSamples := binary.LittleEndian.Uint32(next.Body[sampleLengthOffset:][:4])
							samples := int(loopStart + loopSamples) // sample length + cue point sample offset
							if samples > output.samples {
								return nil, fmt.Errorf("Sound %v has bad loop length, samples: %v, loop: %v", filename, output.samples, samples)
							}
							output.samples = samples
						}
					}
				}
			}
			// There should be only one 'cue ' chunk, at least the orig does not read more
			break
		}
	}
	return output, nil
}

func getHeader(data []byte) waveHeader {
	header := waveHeader{}
	copy(header.ID[:], data)
	header.Size = binary.LittleEndian.Uint32(data[4:])
	copy(header.RiffType[:], data[8:])
	return header
}

type waveFmt struct {
	/*
		   			0x00	4	Chunk ID	"fmt " (0x666D7420)
						0x04	4	Chunk Data Size	16 + extra format bytes
		   			0x08	2	Compression code	1 - 65,535
		   			0x0a	2	Number of channels	1 - 65,535
		   			0x0c	4	Sample rate	1 - 0xFFFFFFFF
		   			0x10	4	Average bytes per second	1 - 0xFFFFFFFF
		   			0x14	2	Block align	1 - 65,535
		   			0x16	2	Significant bits per sample	2 - 65,535
		   			0x18	2	Extra format bytes	0 - 65,535
		   			0x1a	  Extra format bytes *
	*/
	// The first two are already read as part of the chunk
	// ID                       [4]byte // better be fmt
	// Size                     uint32  // 16 + extra format
	CompressionCode          uint16 // better be PCM (0x0001)
	ChannelNum               uint16 // expect 1 or 2
	SampleRate               uint32
	AvgBytePSec              uint32
	BlockAlign               uint16
	SignificantBitsPerSample uint16
}

func readFMT(c chunk) waveFmt {
	f := waveFmt{}
	br := bytes.NewReader(c.Body)
	err := binary.Read(br, binary.LittleEndian, &f)
	if err != nil {
		log.Printf("could not read fmt chunk")
	}
	return f
}

func onVolumeChange(_ *cvar.Cvar) {
	if cvars.Volume == nil || !soundFlag {
		return
	}
	v := cvars.Volume.Value()
	if v > 1 {
		cvars.Volume.SetByString("1")
		// this will cause recursion so exit early
		return
	}
	if v < 0 {
		cvars.Volume.SetByString("0")
		// this will cause recursion so exit early
		return
	}
	// this needs some init to work,
	// can only be called between mix.OpenAudio and mix.CloseAudio
	mix.Volume(-1, int(v*mix.MAX_VOLUME))
}

func init() {
	// was called sfxvolume
	cvars.Volume.SetCallback(onVolumeChange)
}
