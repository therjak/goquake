package quakelib

//#ifndef ENTITIES_H
//#define ENTITIES_H
//#include "q_stdinc.h"
//#include "gl_model.h"
//#include "render.h"
//extern entity_t *cl_entities;
//extern entity_t cl_viewent;
//typedef entity_t* entityPtr;
//typedef qmodel_t* modelPtr;
//inline entity_t* getCLEntity(int i) { return &cl_entities[i]; }
//extern entity_t cl_static_entities[512];
//inline entity_t* getStaticEntity(int i) { return &cl_static_entities[i]; }
//#endif
import "C"

import (
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/math/vec"
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
	for i := 0; i < cl.numEntities; i++ {
		conlog.Printf("%3d:", i)
		e := cl.Entities(i)
		if e.ptr.model == nil {
			conlog.Printf("EMPTY\n")
			continue
		}
		n := C.GoString(&e.ptr.model.name[0])
		f := int(e.ptr.frame)
		a := e.angles()
		o := e.origin()
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

//export EntityIsPlayer
func EntityIsPlayer(ptr C.entityPtr) bool {
	for i := 0; i < cl.maxClients; i++ {
		if cl.Entities(i+1).ptr == ptr {
			return true
		}
	}
	return false
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

type Entity struct {
	ptr C.entityPtr
}

func (c *Client) Entities(i int) *Entity {
	// TODO: make separate sets of Entities(0) for world and
	// Entities(1 to cl.maxClients) for players
	return &Entity{C.getCLEntity(C.int(i))}
}

func (c *Client) ClientEntity(i int) *Entity {
	return &Entity{C.getCLEntity(C.int(i + 1))}
}

func (c *Client) WorldEntity() *Entity {
	return &Entity{C.getCLEntity(0)}
}

func (e *Entity) origin() vec.Vec3 {
	return vec.Vec3{
		float32(e.ptr.origin[0]),
		float32(e.ptr.origin[1]),
		float32(e.ptr.origin[2]),
	}
}

func (e *Entity) angles() vec.Vec3 {
	return vec.Vec3{
		float32(e.ptr.angles[0]),
		float32(e.ptr.angles[1]),
		float32(e.ptr.angles[2]),
	}
}

func cl_weapon() Entity {
	return Entity{&C.cl_viewent}
}

// This one adds error checks to cl_entities
//export CL_EntityNum
func CL_EntityNum(num int) C.entityPtr {
	if num < 0 {
		Error("CL_EntityNum: %d is an invalid number", num)
	}
	if num >= cl.numEntities {
		if num >= cl.maxEdicts {
			Error("CL_EntityNum: %d is an invalid number", num)
		}
		for cl.numEntities <= num {
			cl.Entities(num).ptr.lerpflags |= lerpResetMove | lerpResetAnim
			cl.numEntities++
		}
	}

	return cl.Entities(num).ptr
}

func (c *Client) StaticEntityNum(num int) *Entity {
	return &Entity{C.getStaticEntity(C.int(num))}
}

func (c *Client) EntityNum(num int) *Entity {
	return &Entity{CL_EntityNum(num)}
}

// Entity return the player entity
func (c *Client) Entity() *Entity {
	return c.Entities(c.viewentity)
}

//export CLViewEntity
func CLViewEntity() C.entityPtr {
	return cl.Entity().ptr
}

//export ClientEntity
func ClientEntity(i int) C.entityPtr {
	return cl.ClientEntity(i).ptr
}

//export SetWorldEntityModel
func SetWorldEntityModel(m C.modelPtr) {
	cl.WorldEntity().ptr.model = m
}

func (e *Entity) SetBaseline(state *EntityState) {
	e.ptr.baseline.origin[0] = C.float(state.Origin[0])
	e.ptr.baseline.origin[1] = C.float(state.Origin[1])
	e.ptr.baseline.origin[2] = C.float(state.Origin[2])
	e.ptr.baseline.angles[0] = C.float(state.Angles[0])
	e.ptr.baseline.angles[1] = C.float(state.Angles[1])
	e.ptr.baseline.angles[2] = C.float(state.Angles[2])
	e.ptr.baseline.modelindex = C.ushort(state.ModelIndex)
	e.ptr.baseline.frame = C.ushort(state.Frame)
	e.ptr.baseline.skin = C.uchar(state.Skin)
	e.ptr.baseline.alpha = C.uchar(state.Alpha)
}
