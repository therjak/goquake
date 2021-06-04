// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//#ifndef ENTITIES_H
//#define ENTITIES_H
//#include <stdio.h>
//#include "q_stdinc.h"
//#include "gl_model.h"
//#include "render.h"
//extern entity_t *cl_entities;
//extern entity_t cl_viewent;
//extern entity_t cl_temp_entities[256];
//typedef entity_t* entityPtr;
//typedef qmodel_t* modelPtr;
//inline entity_t* getCLEntity(int i) { return &cl_entities[i]; }
//extern entity_t cl_static_entities[512];
//inline entity_t* getStaticEntity(int i) { return &cl_static_entities[i]; }
//void CL_ParseStaticC(entity_t* e, int modelindex);
//void R_DrawBrushModel(entity_t* e);
//#endif
import "C"

import (
	"unsafe"

	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/model"
	"github.com/therjak/goquake/texture"
)

const (
	lerpMoveStep   = 1 << iota // this is a MOVETYPE_STEP entity, enable movement lerp
	lerpResetAnim              // disable anim lerping until next anim frame
	lerpResetAnim2             // set his and the previous flag to disable anim lerping for two anim frames
	lerpResetMove              // disable movement lerping until next origin/angles change
	lerpFinish                 // use lerpfinish time from server update instead of assuming interval of 0.1
)

var (
	playerTextures map[C.entityPtr]*texture.Texture
)

func init() {
	playerTextures = make(map[C.entityPtr]*texture.Texture)
	cmd.AddCommand("entities", printEntities)
}

func printEntities(_ []cmd.QArg, _ int) {
	if cls.state != ca_connected {
		return
	}
	for i, e := range cl.entities {
		conlog.Printf("%3d:", i)
		if e.ptr.model == nil {
			conlog.Printf("EMPTY\n")
			continue
		}
		n := C.GoString(&e.ptr.model.name[0])
		f := int(e.ptr.frame)
		a := e.Angles
		o := e.Origin
		conlog.Printf("%s:%2d  (%5.1f,%5.1f,%5.1f) [%5.1f %5.1f %5.1f]\n",
			n, f, o[0], o[1], o[2], a[0], a[1], a[2])
	}
}

//export PlayerTexture
func PlayerTexture(ptr C.entityPtr) uint32 {
	t, ok := playerTextures[ptr]
	if !ok || t == nil {
		return 0
	}
	texmap[t.ID()] = t
	return uint32(t.ID())
}

//export CL_NewTranslation
func CL_NewTranslation(i int) {
	if i < 0 || i >= cl.maxClients {
		Error("CL_NewTranslation: slot > cl.maxClients: %d", i)
	}
	R_TranslatePlayerSkin(i)
}

//export R_TranslatePlayerSkin
func R_TranslatePlayerSkin(i int) {
	if cvars.GlNoColors.Bool() {
		return
	}
	if i < 0 || i >= cl.maxClients {
		return
	}
	// s := cl.scores[i]
	e := cl.Entities(i + 1)
	t, ok := playerTextures[e.ptr]
	if !ok || t == nil {
		// There are R_TranslatePlayerSkin calls before we even loaded
		// the player texture. So just ignore.
		return
	}
	// TODO(therjak): do the remap from s.topColor & s.bottomColor
	// we do have indexed colors for the texture
	textureManager.ReloadImage(t)
}

type state struct {
	Origin     vec.Vec3
	Angles     vec.Vec3
	ModelIndex int
	Frame      int
	ColorMap   int
	Skin       int
	Alpha      byte // TODO(therjak): should probably by float32
	Effects    int
}

type Entity struct {
	ptr C.entityPtr

	ForceLink      bool
	Baseline       state
	MsgTime        float64
	MsgOrigin      [2]vec.Vec3
	Origin         vec.Vec3
	MsgAngles      [2]vec.Vec3
	Angles         vec.Vec3
	Model          model.Model
	Fragment       *entityFragment
	Frame          int
	SyncBase       float32
	Effects        int
	SkinNum        int
	VisFrame       int
	DLightFrame    int
	DLightBits     int  // uint32?
	Alpha          byte // TODO(therjak): use the converted float32
	LerpFlags      byte
	LerpStart      float64
	LerpTime       float64
	LerpFinish     float64
	PreviousPose   int
	CurrentPose    int
	MoveLerpStart  float64
	PreviousOrigin vec.Vec3
	CurrentOrigin  vec.Vec3
	PreviousAngles vec.Vec3
	CurrentAngles  vec.Vec3
}

func (c *Client) Entities(i int) *Entity {
	// TODO: make separate sets of Entities(0) for world and
	// Entities(1 to cl.maxClients) for players ?
	return c.entities[i]
}

func (c *Client) ClientEntity(i int) *Entity {
	return c.entities[i+1]
}

func (c *Client) WorldEntity() *Entity {
	return c.entities[0]
}

// Sync synces from the go side to the C side
func (e *Entity) Sync() {
	e.ptr.frame = C.int(e.Frame)
	e.ptr.skinnum = C.int(e.SkinNum)
	e.ptr.alpha2 = C.uchar(e.Alpha)
	e.ptr.origin[0] = C.float(e.Origin[0])
	e.ptr.origin[1] = C.float(e.Origin[1])
	e.ptr.origin[2] = C.float(e.Origin[2])
	e.ptr.angles[0] = C.float(e.Angles[0])
	e.ptr.angles[1] = C.float(e.Angles[1])
	e.ptr.angles[2] = C.float(e.Angles[2])
}

// Sync synces from the C side to the go side
func (e *Entity) SyncC() {
	e.Frame = int(e.ptr.frame)
	e.SkinNum = int(e.ptr.skinnum)
	e.Origin = vec.Vec3{
		float32(e.ptr.origin[0]),
		float32(e.ptr.origin[1]),
		float32(e.ptr.origin[2]),
	}
	e.Angles = vec.Vec3{
		float32(e.ptr.angles[0]),
		float32(e.ptr.angles[1]),
		float32(e.ptr.angles[2]),
	}
}

//TODO(therjak): remove idx and just use a pointer to Entity
func (e *Entity) Relink(frac, bobjrotate float32, idx int) {
	e.SyncC()
	if e.Model == nil { // empty slot
		if e.ForceLink { // just became empty
			e.R_RemoveEfrags()
		}
		return
	}

	// if the object wasn't included in the last packet, remove it
	if e.MsgTime != cl.messageTime {
		e.Model = nil
		e.ptr.model = nil
		// next time this entity slot is reused, the lerp will need to be reset
		e.LerpFlags |= lerpResetMove | lerpResetAnim
		e.Sync()
		return
	}

	oldOrigin := e.Origin

	if e.ForceLink {
		// the entity was not updated in the last message so move to the final spot
		e.Origin = e.MsgOrigin[0]
		e.Angles = e.MsgAngles[0]
	} else {
		var delta vec.Vec3
		// if the delta is large, assume a teleport and don't lerp
		f := frac
		for j := 0; j < 3; j++ {
			delta[j] = e.MsgOrigin[0][j] - e.MsgOrigin[1][j]
			if delta[j] > 100 || delta[j] < -100 {
				// assume a teleportation, not a motion
				f = 1
				e.LerpFlags |= lerpResetMove
			}
		}
		// don't cl_lerp entities that will be r_lerped
		if cvars.RLerpMove.Bool() && e.LerpFlags&lerpMoveStep != 0 {
			f = 1
		}
		// interpolate the origin and angles
		for j := 0; j < 3; j++ {
			e.Origin[j] = e.MsgOrigin[1][j] + f*delta[j]

			d := e.MsgAngles[0][j] - e.MsgAngles[1][j]
			if d > 180 {
				d -= 360
			} else if d < -180 {
				d += 360
			}
			e.Angles[j] = e.MsgAngles[1][j] + f*d
		}
	}

	// rotate binary objects locally
	if e.Model.Flags()&model.EntityEffectRotate != 0 {
		e.Angles[1] = bobjrotate
	}

	if e.Effects&model.EntityEffectBrightField != 0 {
		particlesAddEntity(e.Origin, float32(cl.time))
	}

	if e.Effects&model.EntityEffectMuzzleFlash != 0 {
		dl := cl.GetDynamicLightByKey(idx)
		dl.Key = idx
		dl.Color = vec.Vec3{1, 1, 1}
		dl.Origin = e.Origin
		dl.Origin[2] += 16

		forward, _, _ := vec.AngleVectors(e.Angles)
		dl.Origin.Add(vec.Scale(18, forward))

		dl.Radius = 200 + float32(cRand.Uint32n(32))
		dl.MinLight = 32
		dl.DieTime = cl.time + 0.1
		dl.Sync()

		// assume muzzle flash accompanied by muzzle flare, which looks bad when lerped
		if cvars.RLerpModels.Value() != 2 {
			if e == cl.Entity() {
				// no lerping for two frames
				// FIXME(therjak):
				// cl_viewent.LerpFlags |= lerpResetAnim | lerpResetAnim2
			} else {
				// no lerping for two frames
				e.LerpFlags |= lerpResetAnim | lerpResetAnim2
			}
		}
	}
	if e.Effects&model.EntityEffectBrightLight != 0 {
		dl := cl.GetDynamicLightByKey(idx)
		dl.Key = idx
		dl.Color = vec.Vec3{1, 1, 1}
		dl.Origin = e.Origin
		dl.Origin[2] += 16
		dl.Radius = 400 + float32(cRand.Uint32n(32))
		dl.DieTime = cl.time + 0.001
		dl.Sync()
	}
	if e.Effects&model.EntityEffectDimLight != 0 {
		dl := cl.GetDynamicLightByKey(idx)
		dl.Key = idx
		dl.Color = vec.Vec3{1, 1, 1}
		dl.Origin = e.Origin
		dl.Radius = 200 + float32(cRand.Uint32n(32))
		dl.DieTime = cl.time + 0.001
		dl.Sync()
	}

	switch f := e.Model.Flags(); {
	case f&model.EntityEffectGib != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 2, float32(cl.time))
	case f&model.EntityEffectZomGib != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 4, float32(cl.time))
	case f&model.EntityEffectTracer != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 3, float32(cl.time))
	case f&model.EntityEffectTracer2 != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 5, float32(cl.time))
	case f&model.EntityEffectRocket != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 0, float32(cl.time))
		dl := cl.GetDynamicLightByKey(idx)
		dl.Key = idx
		dl.Color = vec.Vec3{1, 1, 1}
		dl.Origin = e.Origin
		dl.Radius = 200
		dl.DieTime = cl.time + 0.01
		dl.Sync()
	case f&model.EntityEffectGrenade != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 1, float32(cl.time))
	case f&model.EntityEffectTracer3 != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 6, float32(cl.time))
	}
	e.ForceLink = false
	e.Sync()
	if idx == cl.viewentity && !cvars.ChaseActive.Bool() {
		return
	}
	cl.AddVisibleEntity(e)
}

// This one adds error checks to cl_entities
//export CL_EntityNum
func CL_EntityNum(num int) C.entityPtr {
	return cl.GetOrCreateEntity(num).ptr
}

var (
	clientWeapon       Entity
	clientTempEntities []Entity
)

func init() {
	clientWeapon.ptr = &C.cl_viewent
	clientTempEntities = make([]Entity, 0, 256)
}

func (c *Client) WeaponEntity() *Entity {
	return &clientWeapon
}

func (c *Client) CreateStaticEntity() *Entity {
	if len(c.staticEntities) == cap(c.staticEntities) {
		Error("Too many static entities")
	}
	i := len(c.staticEntities)
	c.staticEntities = append(c.staticEntities, Entity{
		ptr: &C.cl_static_entities[i],
	})
	return &c.staticEntities[i]
}

// GetOrCreateEntity returns cl.entities[num] and extends cl.entities if not long enough.
func (c *Client) GetOrCreateEntity(num int) *Entity {
	if num < 0 {
		Error("CL_EntityNum: %d is an invalid number", num)
	}
	if num >= len(cl.entities) {
		if num >= cap(cl.entities) {
			Error("CL_EntityNum: %d is an invalid number", num)
		}
		for i := len(cl.entities); i <= num; i++ {
			e := &Entity{ptr: C.getCLEntity(C.int(i))}
			e.LerpFlags |= lerpResetMove | lerpResetAnim
			e.ptr.lerpflags = C.uchar(e.LerpFlags)
			cl.entities = append(cl.entities, e)
		}
	}
	return cl.entities[num]
}

// Entity return the player entity
func (c *Client) Entity() *Entity {
	return c.Entities(c.viewentity)
}

//export ClientEntity
func ClientEntity(i int) C.entityPtr {
	return cl.ClientEntity(i).ptr
}

//export SetWorldEntityModel
func SetWorldEntityModel(m C.modelPtr) {
	cl.WorldEntity().ptr.model = m
}

func (e *Entity) R_RemoveEfrags() {
	RemoveEntityFragments(e)
}

func (e *Entity) R_AddEfrags() {
	ef := EntityFragmentAdder{entity: e, world: cl.worldModel}
	ef.Do()
}

func (e *Entity) ParseStaticC(index int) {
	C.CL_ParseStaticC(e.ptr, C.int(index))
}

func (r *qRenderer) DrawBrushModel(e *Entity) {
	C.R_DrawBrushModel(e.ptr)
}

//TODO(therjak): should this go into renderer?
var visibleEntities []*Entity // pointers into cl.entities, cl.staticEntities, tempEntities

func ClearVisibleEntities() {
	visibleEntities = visibleEntities[:0]
}

func ClearTempEntities() {
	clientTempEntities = clientTempEntities[:0]
}

func (c *Client) NewTempEntity() *Entity {
	// TODO(therjak): do not return nil, add error return value
	if VisibleEntitiesNum() >= 4096 {
		return nil
	}
	if len(clientTempEntities) == cap(clientTempEntities) {
		return nil
	}
	i := len(clientTempEntities)
	cptr := &C.cl_temp_entities[i]
	C.memset(unsafe.Pointer(cptr), 0, C.sizeof_entity_t)
	clientTempEntities = append(clientTempEntities,
		Entity{
			ptr: cptr,
		})
	ent := &clientTempEntities[i]
	c.AddVisibleEntity(ent)
	return ent
}

func (c *Client) AddVisibleEntity(e *Entity) {
	if len(visibleEntities) >= 4096 {
		return
	}
	visibleEntities = append(visibleEntities, e)
}

//export VisibleEntity
func VisibleEntity(i int) C.entityPtr {
	return visibleEntities[i].ptr
}

//export VisibleEntitiesNum
func VisibleEntitiesNum() int {
	return len(visibleEntities)
}
