// SPDX-License-Identifier: GPL-2.0-or-later

package snd

var (
	soundPrecache cache
)

type cache []*pcmSound

func (c *cache) Get(i int) *pcmSound {
	if i < 0 || i >= len(*c) {
		return nil
	}
	return (*c)[i]
}

func (c *cache) Has(n string) (int, bool) {
	for i, s := range *c {
		if s.name == n {
			return i, true
		}
	}
	return -1, false
}

func (c *cache) Add(s *pcmSound) int {
	r := len(*c)
	*c = append(*c, s)
	return r
}
