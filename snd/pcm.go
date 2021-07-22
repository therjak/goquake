// SPDX-License-Identifier: GPL-2.0-or-later

package snd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"goquake/filesystem"
)

const (
	stereo = 2
	mono   = 1
)

type pcmSound struct {
	name       string
	samples    int // number of samples
	bitrate    int
	channelNum int
	loopStart  int // in samples, length/bitrate
	sampleRate int
	pcm        []byte
}

// Resample converts to 16bit stereo
func (s *pcmSound) resample() error {
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

func (s *pcmSound) resample16Mono() error {
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

func (s *pcmSound) resample8Stereo() error {
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

func (s *pcmSound) resample8Mono() error {
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

func loadSFX(filename string) (*pcmSound, error) {
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
	output := &pcmSound{name: filename}

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
