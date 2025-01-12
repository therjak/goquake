// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"time"

	"goquake/filesystem"
)

const (
	stereo = 2
	mono   = 1
)

type pcmSound struct {
	name          string
	numChans      uint16
	loopStart     uint32 // in samples, length/bitrate
	loopSamples   uint32
	sampleRate    uint32
	byteRate      uint32
	bytesPerFrame uint16
	bitsPerSample uint16
	data          []byte
	dataSize      uint32
}

func (p *pcmSound) Name() string {
	return p.name
}

type sound struct {
	*pcmSound
	reader *bytes.Reader
	pos    uint32
	err    error
}

func newSound(p *pcmSound) *sound {
	return &sound{
		pcmSound: p,
		reader:   bytes.NewReader(p.data),
	}
}

func (p *sound) Len() int {
	numBytes := time.Duration(p.dataSize)
	perFrame := time.Duration(p.bytesPerFrame)
	return int(numBytes / perFrame)
}

func (p *sound) Seek(newPos int) error {
	if newPos < 0 || p.Len() < newPos {
		return fmt.Errorf("seek position %d is out of range [%d,%d]", newPos, 0, p.Len())
	}
	pos := uint32(newPos) * uint32(p.bytesPerFrame)
	if _, err := p.reader.Seek(int64(pos), io.SeekStart); err != nil {
		return fmt.Errorf("seek error: %w", err)
	}
	p.pos = pos
	return nil
}

func (p *sound) Position() int {
	return int(p.pos / uint32(p.bytesPerFrame))
}

func (p *sound) Err() error {
	return p.err
}

func (p *sound) Stream(samples [][2]float64) (n int, ok bool) {
	if p.err != nil || p.pos >= p.dataSize {
		return 0, false
	}
	bytesPerFrame := int(p.bytesPerFrame)
	wantBytes := len(samples) * bytesPerFrame
	availableBytes := int(p.dataSize - p.pos)
	numBytes := min(wantBytes, availableBytes)
	d := make([]byte, numBytes)
	n, err := p.reader.Read(d)
	if err != nil && err != io.EOF {
		p.err = err
	}
	switch {
	case p.bitsPerSample == 8 && p.numChans == 1:
		for i, j := 0, 0; i <= n-bytesPerFrame; i, j = i+bytesPerFrame, j+1 {
			val := float64(d[i])/(1<<8)*2 - 1
			samples[j][0] = val
			samples[j][1] = val
		}
	case p.bitsPerSample == 8 && p.numChans == 2:
		for i, j := 0, 0; i <= n-bytesPerFrame; i, j = i+bytesPerFrame, j+1 {
			samples[j][0] = float64(d[i+0])/(1<<8)*2 - 1
			samples[j][1] = float64(d[i+1])/(1<<8)*2 - 1
		}
	case p.bitsPerSample == 16 && p.numChans == 1:
		for i, j := 0, 0; i <= n-bytesPerFrame; i, j = i+bytesPerFrame, j+1 {
			val := float64(int16(d[i+0])+int16(d[i+1])*(1<<8)) / (1 << 15)
			samples[j][0] = val
			samples[j][1] = val
		}
	case p.bitsPerSample == 16 && p.numChans == 2:
		for i, j := 0, 0; i <= n-bytesPerFrame; i, j = i+bytesPerFrame, j+1 {
			samples[j][0] = float64(int16(d[i+0])+int16(d[i+1])*(1<<8)) / (1 << 15)
			samples[j][1] = float64(int16(d[i+2])+int16(d[i+3])*(1<<8)) / (1 << 15)
		}
	}
	p.pos += uint32(n)
	return n / bytesPerFrame, true
}

type header struct {
	ID   [4]byte
	Size uint32
}

type waveHeader struct {
	ID       [4]byte // better be RIFF
	Size     uint32  // file size - 8
	RiffType [4]byte // better be WAVE
}

type chunk struct {
	header
	Data *io.SectionReader
}

// http://www.piclist.com/techref/io/serial/midi/wave.html

func loadSFX(filename string) (sound *pcmSound, err error) {
	file, err := filesystem.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Could not load file %v: %v", filename, err)
	}
	defer file.Close()

	wh := waveHeader{} // 12 byte
	if err := binary.Read(file, binary.LittleEndian, &wh); err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}

	if wh.ID != [4]byte{'R', 'I', 'F', 'F'} ||
		wh.RiffType != [4]byte{'W', 'A', 'V', 'E'} {
		return nil, fmt.Errorf("file is not a RIFF wave file")
	}

	chunks := []*chunk{}
	nextChunkStart := int64(12)
	for { // we have at least one chunk left to read
		c := &chunk{}
		if err := binary.Read(file, binary.LittleEndian, &c.header); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read chunk header: %v", err)
		}
		nextChunkStart += 8
		size := int64(c.Size)
		if size%2 != 0 {
			// spec says chunks are WORD aligned (2,4,6,8,10,...) with 0 padding
			// but 'size' does not include padding.
			size = size + 1
		}
		c.Data = io.NewSectionReader(file, nextChunkStart, size)
		nextChunkStart, err = file.Seek(size, os.SEEK_CUR)
		if err != nil {
			return nil, fmt.Errorf("Seek error: %v", err)
		}
		chunks = append(chunks, c)
	}
	output := &pcmSound{
		name: filename,
	}

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
			if f.BitsPerSample != 8 && f.BitsPerSample != 16 {
				return nil, fmt.Errorf("Invalid sound bitrate: %v", f.BitsPerSample)
			}
			output.numChans = f.ChannelNum
			output.sampleRate = f.SampleRate
			output.byteRate = f.ByteRate
			output.bytesPerFrame = f.BytesPerFrame
			output.bitsPerSample = f.BitsPerSample
			gotFMT += 1
		}
	}
	if gotFMT != 1 {
		return nil, fmt.Errorf("Invalid number of fmt blocks: %v", gotFMT)
	}
	output.loopStart = math.MaxUint32

	cueIdx := -1
	for idx, c := range chunks {
		id := string(c.ID[:])
		switch id {
		default:
			log.Printf("unknown chunk: %q", string(c.ID[:]))
		case "fmt ":
			// already parsed
		case "data":
			output.dataSize = c.Size
			data, err := io.ReadAll(c.Data)
			if err != nil {
				return nil, fmt.Errorf("Failed to read pcm: %w", err)
			}
			output.data = data

		case "cue ":
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
			cueIdx = idx
			var numCuePoints uint32
			if err := binary.Read(c.Data, binary.LittleEndian, &numCuePoints); err != nil {
				return nil, fmt.Errorf("Invalid CuePoints: %v", err)
			}
			var cuePoint struct {
				ID           uint32
				Pos          uint32
				DataChunkID  [4]byte
				ChunkStart   uint32
				BlockStart   uint32
				SampleOffset uint32
			}
			if numCuePoints != 1 {
				log.Printf("NumCuePoints != 1")
			}
			for i := uint32(0); i < numCuePoints; i++ {
				if err := binary.Read(c.Data, binary.LittleEndian, &cuePoint); err != nil {
					return nil, fmt.Errorf("Invalid CuePoint: %v", err)
				}
				output.loopStart = cuePoint.SampleOffset
				break
			}
		case "LIST":
			if cueIdx+1 != idx {
				// the original code expects this 'LIST' to follow the 'cue ' to be
				// a valid 'mark' entry for the loopSample number
				continue
			}
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
			var listType [4]byte
			if err := binary.Read(c.Data, binary.LittleEndian, &listType); err != nil {
				return nil, fmt.Errorf("Invalid CuePoint: %v", err)
			}
			t := string(listType[:])
			switch t {
			default:
				log.Printf("Wave file with LIST Type: %q", t)
			case "adtl":
				var adtlHeader header
				if err := binary.Read(c.Data, binary.LittleEndian, &adtlHeader); err != nil {
					return nil, fmt.Errorf("Invalid adtlHeader: %v", err)
				}
				if string(adtlHeader.ID[:]) != "ltxt" {
					log.Printf("invalid adtl type %q", string(adtlHeader.ID[:]))
					break
				}
				var adtl struct {
					CuePointID   [4]byte
					SampleLength uint32
					PurposeID    [4]byte
					/*
						A full ltxt would also have
						County uint16
						Language uint16
						Dialect uint16
						CodePage uint16
						Text []byte
						but we do not care about this data
					*/
				}
				if err := binary.Read(c.Data, binary.LittleEndian, &adtl); err != nil {
					return nil, fmt.Errorf("Invalid adtl: %v", err)
				}
				if string(adtl.PurposeID[:]) == "mark" {
					output.loopSamples = adtl.SampleLength
				}
			}
		}
	}
	return output, nil
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
	CompressionCode uint16 // better be PCM (0x0001)
	ChannelNum      uint16 // expect 1 or 2
	SampleRate      uint32
	ByteRate        uint32
	BytesPerFrame   uint16
	BitsPerSample   uint16
}

func readFMT(c *chunk) waveFmt {
	f := waveFmt{}
	err := binary.Read(c.Data, binary.LittleEndian, &f)
	if err != nil {
		log.Printf("could not read fmt chunk")
	}
	return f
}
