// SPDX-License-Identifier: GPL-2.0-or-later
package model

import (
	"github.com/therjak/goquake/math/vec"
)

const (
	EntityEffectBrightField = 1 << iota
	EntityEffectMuzzleFlash // 2
	EntityEffectBrightLight // 4
	EntityEffectDimLight    // 8
)

const (
	EntityEffectRocket  = 1 << iota
	EntityEffectGrenade // 2
	EntityEffectGib     // 4
	EntityEffectRotate  // 8
	EntityEffectTracer  // 16
	EntityEffectZomGib  // 32
	EntityEffectTracer2 // 64
	EntityEffectTracer3 // 128
)

type ModType int

const (
	ModBrush ModType = iota
	ModSprite
	ModAlias
)

const (
	MAX_MODELS = 2048
)

type QModel struct {
	name    string  // alias + sprite + brush
	modType ModType // alias + sprite + brush

	flags int // alias
	// Cache // alias + sprite
	// vboindexofs // alias
	// vboxyzofs // alias
	// vbostofs // alias
	// meshindexesvbo // alias
	// meshvbo // alias

	mins vec.Vec3 // sprite + alias + brush
	maxs vec.Vec3 // sprite + alias + brush
	// rmins // alias + brush
	// rmaxs // alias + brush
	// ymins // alias + brush
	// ymaxs // alias + brush

	FrameCount int // numframes, alias + sprite + brush
	SyncType   int // alias + sprite
}

func (q *QModel) Mins() vec.Vec3 {
	return q.mins
}
func (q *QModel) Maxs() vec.Vec3 {
	return q.maxs
}
func (q *QModel) Type() ModType {
	return q.modType
}
func (q *QModel) Name() string {
	return q.name
}
func (q *QModel) Flags() int {
	return q.flags
}

func (q *QModel) SetMins(m vec.Vec3) {
	q.mins = m
}
func (q *QModel) SetMaxs(m vec.Vec3) {
	q.maxs = m
}
func (q *QModel) SetType(t ModType) {
	q.modType = t
}
func (q *QModel) SetName(n string) {
	q.name = n
}
func (q *QModel) SetFlags(f int) {
	q.flags = f
}

type Model interface {
	Name() string
	Type() ModType
	Mins() vec.Vec3
	Maxs() vec.Vec3
	Flags() int
}
