package palette

func AlphaEdgeFix(w, h int32, d []byte) {
	alpha := func(p int32) byte {
		return d[p+3]
	}
	for y := int32(0); y < h; y++ {
		prev := (y - 1 + h) % h
		next := (y + 1) % h
		for x := int32(0); x < w; x++ {
			pp := (x - 1 + w) % w
			np := (x + 1) % w
			prow := prev * w
			crow := y * w
			nrow := next * w
			p := []int32{
				(pp + prow) * 4, (x + prow) * 4, (np + prow) * 4,
				(pp + crow) * 4 /*           */, (np + crow) * 4,
				(pp + nrow) * 4, (x + nrow) * 4, (np + nrow) * 4,
			}
			pixel := (x + crow) * 4
			if alpha(pixel) == 0 {
				r, g, b := int32(0), int32(0), int32(0)
				count := int32(0)
				for _, rp := range p {
					if alpha(rp) != 0 {
						r += int32(d[rp])
						g += int32(d[rp+1])
						b += int32(d[rp+2])
						count++
					}
				}
				if count != 0 {
					d[pixel] = byte(r / count)
					d[pixel+1] = byte(g / count)
					d[pixel+2] = byte(b / count)
				}
			}
		}
	}
}
