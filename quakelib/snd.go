package quakelib

// #include "q_sound.h"

// typedef int sfx_t;
// #include "cgo_help.h"
import "C"

import (
	"log"
	"quake/cmd"
	"quake/math/vec"
	"quake/snd"
)

// it should support to play U8, S8 and S16 sounds (is it necessary to replicate this?)

// there are 4 ambient sound channel

const (
	loopingSound = true
)

//export S_Init
func S_Init() {
	snd.Init()
}

//export S_Shutdown
func S_Shutdown() {
	snd.Shutdown()
}

//export S_StartSound
func S_StartSound(entnum C.int, entchannel C.int, sfx C.sfx_t, origin *C.float,
	fvol C.float, attenuation C.float) {
	snd.Start(int(entnum), int(entchannel), int(sfx), cfloatToVec3(origin), float32(fvol), float32(attenuation), !loopingSound)
}

//export S_StaticSound
func S_StaticSound(sfx C.sfx_t, origin *C.float, vol C.float, attenuation C.float) {
	// sfx is cached from S_PrecacheSound
	// distance from origin to cl.viewentity
	snd.Start(0, 0, int(sfx), cfloatToVec3(origin), float32(vol/255), float32(attenuation/64), loopingSound)
}

//export S_StopSound
func S_StopSound(entnum C.int, entchannel C.int) {
	// why does the server know which channel to stop?
	snd.Stop(int(entnum), int(entchannel))
}

//export S_StopAllSounds
func S_StopAllSounds(clear C.int) {
	// clear is bool, indicates if S_ClearBuffer should be called
	// clear is always true
	snd.StopAll()
}

//export S_ClearBuffer
func S_ClearBuffer() {
	// remove stuff already in the pipeline to be played
}

func cfloatToVec3(f *C.float) vec.Vec3 {
	a := C.cf(0, f)
	b := C.cf(1, f)
	c := C.cf(2, f)
	return vec.Vec3{X: float32(a), Y: float32(b), Z: float32(c)}
}

//export S_Update
func S_Update(origin *C.float, _ *C.float, right *C.float, _ *C.float) {
	// update the direction and distance to all sound sources
	listener := snd.Listener{
		Origin: cfloatToVec3(origin),
		Right:  cfloatToVec3(right),
		ID:     cl.viewentity,
	}
	// TODO(therjak): snd.UpdateAmbientSounds(ambient_levels)
	// with ambient_levels containing
	// ambient_level
	// ambient_sound_level per ambient channel [4]
	snd.Update(listener)
}

//export S_ExtraUpdate
func S_ExtraUpdate() {}

//export S_PrecacheSound
func S_PrecacheSound(sample *C.char) C.sfx_t {
	n := C.GoString(sample)
	r := snd.PrecacheSound(n)
	return C.sfx_t(r)
}

//export S_TouchSound
func S_TouchSound(sample *C.char) {
	// Just ignore and let PrecacheSound handle it
}

//export S_LocalSound
func S_LocalSound(name *C.char) {
	// TODO name is a 'path/filename.wav'
	// This is mostly for the menu sounds
	// sfx := S_PrecacheSound(name)
	// S_StartSound(cl.viewentity, -1, sfx, vec3_origin /* {0,0,0} */, 1,1 )
	localSound(C.GoString(name))
}

func localSound(name string) {
}

func init() {
	cmd.AddCommand("play", playCmd)
	cmd.AddCommand("playvol", playVolCmd)
	cmd.AddCommand("stopsound", stopSoundCmd)
	cmd.AddCommand("soundlist", soundListCmd)
	cmd.AddCommand("soundinfo", soundInfoCmd)
}

func playCmd(args []cmd.QArg) {
	log.Println("play CMD from snd")
}
func playVolCmd(args []cmd.QArg) {
	log.Println("play vol CMD from snd")
}
func stopSoundCmd(args []cmd.QArg) {
	log.Println("stop sound CMD from snd")
}
func soundListCmd(args []cmd.QArg) {
	log.Println("sound list CMD from snd")
}
func soundInfoCmd(args []cmd.QArg) {
	log.Println("sound info CMD from snd")
}
