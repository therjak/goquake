// SPDX-License-Identifier: GPL-2.0-or-later

package image

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"os"

	"github.com/therjak/goquake/filesystem"
)

// Write expects RGBA 8bit data
func Write(name string, data []byte, width, height int) error {
	if len(data) < width*height*4 {
		return fmt.Errorf("Tried to write an image but there is not enough data")
	}
	r := image.Rect(0, 0, width, height)
	img := &image.NRGBA{
		Pix:    data,
		Stride: 4 * width,
		Rect:   r,
	}

	f, err := os.Create(name)
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func Load(name string) (*image.NRGBA, error) {
	// Have not seen an image loaded, wait with further implementation
	// until we found an image we could not load.
	if _, err := filesystem.Stat(name + ".tga"); err == nil {
		i, err := loadTGA(name + ".tga")
		if err != nil {
			log.Printf("Failed to load %v.tga, %v", name, err)
		} else {
			log.Printf("Succeeded in loading %v.tga", name)
		}
		return i, err
	}
	if _, err := filesystem.Stat(name + ".pcx"); err == nil {
		i, err := loadPCX(name + ".pcx")
		if err != nil {
			log.Printf("Faild to load %v.pcx, %v", name, err)
		} else {
			log.Printf("Succeeded in loading %v.pcx", name)
		}
		return i, err
	}
	return nil, fmt.Errorf("Image %v not found", name)
}

type tgaHeader struct {
	IDLength       uint8
	ColormapType   uint8
	ImageType      uint8
	ColormapIndex  uint16
	ColormapLength uint16
	ColormapSize   uint8
	XOrigin        uint16
	YOrigin        uint16
	Width          uint16
	Height         uint16
	PixelSize      uint8
	Attributes     uint8
}

func loadTGA(name string) (*image.NRGBA, error) {
	f, err := filesystem.GetFile(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var header tgaHeader
	if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("Invalid tga header in %v: %v", name, err)
	}
	if header.ImageType != 2 && header.ImageType != 10 {
		return nil, fmt.Errorf("TGA %v is not a type 2 or type 10", name)
	}
	if header.ColormapType != 0 || (header.PixelSize != 32 && header.PixelSize != 24) {
		return nil, fmt.Errorf("TGA %v is not 24bit or 32bit", name)
	}

	width, height := int(header.Width), int(header.Height)
	bounds := image.Rect(0, 0, width, height)
	nrgba := image.NewNRGBA(bounds)

	if header.IDLength != 0 {
		// skip Image ID
		f.Seek(int64(header.IDLength), io.SeekCurrent)
	}

	// ColormapType is 0 so no color map data. Next is image data.

	if header.ImageType == 2 {
		// Uncompressed RGB image
		if header.PixelSize == 24 {
			// RGB
			cr := make([]uint8, width*3)
			for y := 0; y < height; y++ {
				n, err := f.Read(cr)
				if err != nil {
					return nil, fmt.Errorf("Failed to read: %v", err)
				}
				if n != len(cr) {
					return nil, fmt.Errorf("Not enough pixels")
				}
				for x := 0; x < width; x++ {
					p := x + width*y
					nrgba.Pix[p*4+0] = cr[p*3+0]
					nrgba.Pix[p*4+1] = cr[p*3+1]
					nrgba.Pix[p*4+2] = cr[p*3+2]
					nrgba.Pix[p*4+3] = 255
				}
			}
		} else /* header.PixelSize == 32 */ {
			// RGBA
			// TODO: Optimize by just copy?
			cr := make([]uint8, width*4)
			for y := 0; y < height; y++ {
				n, err := f.Read(cr)
				if err != nil {
					return nil, fmt.Errorf("Failed to read: %v", err)
				}
				if n != len(cr) {
					return nil, fmt.Errorf("Not enough pixels")
				}
				for x := 0; x < width; x++ {
					p := x + width*y
					nrgba.Pix[p*4+0] = cr[p*4+0]
					nrgba.Pix[p*4+1] = cr[p*4+1]
					nrgba.Pix[p*4+2] = cr[p*4+2]
					nrgba.Pix[p*4+3] = cr[p*4+3]
				}
			}
		}
	} else if header.ImageType == 10 {
		// Runlength encoded RGB image
		return nil, fmt.Errorf("Not implemented")
	}

	return nrgba, nil
}

func loadPCX(name string) (*image.NRGBA, error) {
	return nil, fmt.Errorf("Not implemented")
}
