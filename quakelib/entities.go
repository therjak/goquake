package quakelib

//#ifndef ENTITIES_H
//#define ENTITIES_H
//#include "q_stdinc.h"
//#include "gl_model.h"
//#include "render.h"
//extern entity_t *cl_entities;
//extern entity_t cl_viewent;
//extern int cl_numvisedicts;
//extern entity_t *cl_visedicts[4096];
//extern entity_t cl_temp_entities[256];
//typedef entity_t* entityPtr;
//typedef qmodel_t* modelPtr;
//inline entity_t* getCLEntity(int i) { return &cl_entities[i]; }
//extern entity_t cl_static_entities[512];
//inline entity_t* getStaticEntity(int i) { return &cl_static_entities[i]; }
//void R_AddEfrags(entity_t* e);
//void CL_ParseStaticC(entity_t* e);
//void R_DrawAliasModel(entity_t* e);
//int CL_RelinkEntitiesI(float frac, float bobjrotate, entity_t* e, int i);
//#endif
import "C"

import (
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

//export EntityIsPlayerWeapon
func EntityIsPlayerWeapon(ptr C.entityPtr) bool {
	return ptr == cl.WeaponEntity().ptr
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

	ForceLink  bool
	UpdateType int
	Baseline   EntityState
	MsgTime    float64
	MsgOrigin  [2]vec.Vec3
	Origin     vec.Vec3
	MsgAngles  [2]vec.Vec3
	Angles     vec.Vec3
	Model      *model.QModel
	// efrag *efrag
	Frame         int
	SyncBase      float32
	Effects       int
	SkinNum       int
	VisFrame      int
	DLightFrame   int
	DLightBits    int // uint32?
	TrivialAccept int
	// topNode *MNode_s
	Alpha          byte
	LerpFlags      byte
	LerpStart      float32
	LerpTime       float32
	LerpFinish     float32
	PreviousPose   int16
	CurrentPose    int16
	MoveLerpStart  float32
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

//TODO(therjak): remove idx and just use a pointer to Entity
func (e *Entity) Relink(frac, bobjrotate float32, idx int) {
	r := C.CL_RelinkEntitiesI(C.float(frac), C.float(bobjrotate), e.ptr, C.int(idx))
	if r != 0 {
		AddVisibleEntity(e.ptr)
	}
}

// This one adds error checks to cl_entities
//export CL_EntityNum
func CL_EntityNum(num int) C.entityPtr {
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
	return cl.entities[num].ptr
}

var (
	clientWeapon       Entity
	staticEntity       [512]Entity
	clientTempEntities [256]Entity
)

func init() {
	clientWeapon.ptr = &C.cl_viewent
	for i, _ := range staticEntity {
		staticEntity[i].ptr = &C.cl_static_entities[i]
	}
	for i, _ := range clientTempEntities {
		clientTempEntities[i].ptr = &C.cl_temp_entities[i]
	}
}

func (c *Client) WeaponEntity() *Entity {
	return &clientWeapon
}

func (c *Client) StaticEntityNum(num int) *Entity {
	return &staticEntity[num]
}

func (c *Client) EntityNum(num int) *Entity {
	return &Entity{ptr: CL_EntityNum(num)}
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

func (e *Entity) R_AddEfrags() {
	C.R_AddEfrags(e.ptr)
}

func (e *Entity) ParseStaticC() {
	C.CL_ParseStaticC(e.ptr)
}

func (e *Entity) SetBaseline(state *EntityState) {
	e.Baseline = *state
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

func (r *qRenderer) DrawAliasModel(e *Entity) {
	C.R_DrawAliasModel(e.ptr)
}

var clientVisibleEntities []*Entity // pointers into cl.entities, staticEntities, tempEntities

//export ClearVisibleEntities
func ClearVisibleEntities() {
	C.cl_numvisedicts = 0
	clientVisibleEntities = clientVisibleEntities[:0]
}

//export AddVisibleTempEntity
func AddVisibleTempEntity(e C.entityPtr) {
	if len(clientVisibleEntities) >= 4096 {
		return
	}
	// clientTempEntities [256]Entity
	// clientVisibleEntities = append(clientVisibleEntities,
}

//export AddVisibleStaticEntity
func AddVisibleStaticEntity(e C.entityPtr) {
	if len(clientVisibleEntities) >= 4096 {
		return
	}
	// staticEntity       [512]Entity
	// clientVisibleEntities = append(clientVisibleEntities,
}

//export AddVisibleEntity
func AddVisibleEntity(e C.entityPtr) {
	if len(clientVisibleEntities) >= 4096 {
		return
	}
	if C.cl_numvisedicts < 4096 {
		C.cl_visedicts[C.cl_numvisedicts] = e
		C.cl_numvisedicts++
	}
	// clientEntities     []*Entity
	// clientVisibleEntities = append(clientVisibleEntities,
}

//export VisibleEntity
func VisibleEntity(i int) C.entityPtr {
	return C.cl_visedicts[i]
	// return clientVisibleEntities[i].ptr
}

//export VisibleEntitiesNum
func VisibleEntitiesNum() int {
	return int(C.cl_numvisedicts)
	// return len(clientVisibleEntities)
}
