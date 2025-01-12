// SPDX-License-Identifier: GPL-2.0-or-later

package snd

type citem interface {
	Name() string
}

type cache[T citem] struct {
	elements []T
}

func (c *cache[T]) Get(i int) T {
	if i < 0 || i >= len(c.elements) {
		var zero T
		return zero
	}
	return c.elements[i]
}

func (c *cache[T]) Has(n string) (int, bool) {
	for i, s := range c.elements {
		if s.Name() == n {
			return i, true
		}
	}
	return -1, false
}

func (c *cache[T]) Add(s T) int {
	r := len(c.elements)
	c.elements = append(c.elements, s)
	return r
}
