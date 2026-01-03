// SPDX-License-Identifier: GPL-2.0-or-later

package rand

const (
	noise1 = 0xB5297A4D
	noise2 = 0x68E31DA4
	noise3 = 0x1B56C4E9
)

type Generator struct {
	idx  uint32
	seed uint32
}

func New(seed uint32) Generator {
	return Generator{idx: 0, seed: seed}
}

func noise(p uint32, s uint32) uint32 {
	m := p
	m *= noise1
	m += s
	m ^= (m >> 8)
	m *= noise2
	m ^= (m << 8)
	m *= noise3
	m ^= (m >> 8)
	return m
}

func (g *Generator) rand() uint32 {
	g.idx++
	return noise(g.idx, g.seed)
}

func (g *Generator) NewSeed(s uint32) {
	g.seed = s
}

func (g *Generator) Uint32n(n uint32) uint32 {
	return g.rand() % n
}

func (g *Generator) Intn(n int) int {
	return int(g.Uint32n(uint32(n)))
}

func (g *Generator) Float32() float32 {
	return float32(g.Uint32n(1<<26)) / (1 << 26)
}
