// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"

	"goquake/cmd"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/math/vec"
	"goquake/mdl"
	"goquake/model"
	"goquake/texture"
)

const (
	lerpMoveStep   = 1 << iota // this is a MOVETYPE_STEP entity, enable movement lerp
	lerpResetAnim              // disable anim lerping until next anim frame
	lerpResetAnim2             // set his and the previous flag to disable anim lerping for two anim frames
	lerpResetMove              // disable movement lerping until next origin/angles change
	lerpFinish                 // use lerpfinish time from server update instead of assuming interval of 0.1
)

var (
	// TODO(therjak): this should probably be a two stage thing
	// one of idx to Entity and one idx to texture.
	// currently it is not realy possible to clean up
	playerTextures map[*Entity]*texture.Texture
)

func init() {
	playerTextures = make(map[*Entity]*texture.Texture)
	addCommand("entities", printEntities)
}

func printEntities(_ []cmd.QArg, _ int) error {
	if cls.state != ca_connected {
		return nil
	}
	for i, e := range cl.entities {
		conlog.Printf("%3d:", i)
		if e.Model == nil {
			conlog.Printf("EMPTY\n")
			continue
		}
		n := e.Model.Name
		f := e.Frame
		a := e.Angles
		o := e.Origin
		conlog.Printf("%s:%2d  (%5.1f,%5.1f,%5.1f) [%5.1f %5.1f %5.1f]\n",
			n, f, o[0], o[1], o[2], a[0], a[1], a[2])
	}
	return nil
}

func updatePlayerSkin(i int) {
	if i < 0 || i >= cl.maxClients {
		Error("CL_NewTranslation: slot > cl.maxClients: %d", i)
	}
	e := cl.Entities(i + 1)
	translatePlayerSkin(e)
}

func translatePlayerSkin(e *Entity) {
	if cvars.GlNoColors.Bool() {
		return
	}
	// s := cl.scores[i]
	t, ok := playerTextures[e]
	if !ok || t == nil {
		// There are R_TranslatePlayerSkin calls before we even loaded
		// the player texture. So just ignore.
		return
	}
	// TODO(therjak): do the remap from s.topColor & s.bottomColor
	// we do have indexed colors for the texture
	textureManager.ReloadImage(t)
}

func createPlayerSkin(i int, e *Entity) {
	m, ok := e.Model.(*mdl.Model)
	if !ok || m == nil {
		return
	}
	skinNum := e.SkinNum
	if skinNum < 0 || skinNum >= len(m.Textures) {
		skinNum = 0
	}
	name := fmt.Sprintf("player_%d", i-1) // make it 0 based
	// copy the texture with our new name
	ot := m.Textures[skinNum][0]
	flags := texture.TexPrefPad | texture.TexPrefOverwrite
	t := texture.NewTexture(ot.Width, ot.Height, flags, name, ot.Typ, ot.Data)
	textureManager.addActiveTexture(t)
	textureManager.loadIndexed(t, t.Data)
	playerTextures[e] = t
	translatePlayerSkin(e)
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

//TODO(therjak): remove idx and just use a pointer to Entity
func (e *Entity) Relink(frac, bobjrotate float32, idx int) {
	if e.Model == nil { // empty slot
		if e.ForceLink { // just became empty
			e.R_RemoveEfrags()
		}
		return
	}

	// if the object wasn't included in the last packet, remove it
	if e.MsgTime != cl.messageTime {
		e.Model = nil
		// next time this entity slot is reused, the lerp will need to be reset
		e.LerpFlags |= lerpResetMove | lerpResetAnim
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
		dl.key = idx
		dl.color = vec.Vec3{1, 1, 1}
		dl.origin = e.Origin
		dl.origin[2] += 16

		forward, _, _ := vec.AngleVectors(e.Angles)
		dl.origin.Add(vec.Scale(18, forward))

		dl.radius = 200 + float32(cRand.Uint32n(32))
		dl.minLight = 32
		dl.dieTime = cl.time + 0.1

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
		dl.key = idx
		dl.color = vec.Vec3{1, 1, 1}
		dl.origin = e.Origin
		dl.origin[2] += 16
		dl.radius = 400 + float32(cRand.Uint32n(32))
		dl.dieTime = cl.time + 0.001
	}
	if e.Effects&model.EntityEffectDimLight != 0 {
		dl := cl.GetDynamicLightByKey(idx)
		dl.key = idx
		dl.color = vec.Vec3{1, 1, 1}
		dl.origin = e.Origin
		dl.radius = 200 + float32(cRand.Uint32n(32))
		dl.dieTime = cl.time + 0.001
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
		dl.key = idx
		dl.color = vec.Vec3{1, 1, 1}
		dl.origin = e.Origin
		dl.radius = 200
		dl.dieTime = cl.time + 0.01
	case f&model.EntityEffectGrenade != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 1, float32(cl.time))
	case f&model.EntityEffectTracer3 != 0:
		particlesAddRocketTrail(oldOrigin, e.Origin, 6, float32(cl.time))
	}
	e.ForceLink = false
	if idx == cl.viewentity && !cvars.ChaseActive.Bool() {
		return
	}
	cl.AddVisibleEntity(e)
}

var (
	clientWeapon       Entity
	clientTempEntities []Entity
)

func init() {
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
	c.staticEntities = append(c.staticEntities, Entity{})
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
			e := &Entity{}
			e.LerpFlags |= lerpResetMove | lerpResetAnim
			cl.entities = append(cl.entities, e)
		}
	}
	return cl.entities[num]
}

// Entity return the player entity
func (c *Client) Entity() *Entity {
	return c.Entities(c.viewentity)
}

func (e *Entity) R_RemoveEfrags() {
	RemoveEntityFragments(e)
}

func (e *Entity) R_AddEfrags() {
	ef := EntityFragmentAdder{entity: e, world: cl.worldModel}
	ef.Do()
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
	clientTempEntities = append(clientTempEntities, Entity{})
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

func VisibleEntitiesNum() int {
	return len(visibleEntities)
}
