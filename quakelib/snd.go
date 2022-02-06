// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"

	"goquake/cmd"
	"goquake/commandline"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/math/vec"
	"goquake/snd"
)

// it should support to play U8, S8 and S16 sounds (is it necessary to replicate this?)

// there are 4 ambient sound channel

const (
	loopingSound = true
)

func init() {
	cvars.Volume.SetCallback(onVolumeChange)
}

func onVolumeChange(cv *cvar.Cvar) {
	v := cv.Value()
	if v > 1 {
		cv.SetByString("1")
		// recursion so exit early
		return
	}
	if v < 0 {
		cv.SetByString("0")
		// recursion so exit early
		return
	}
	snd.SetVolume(v)
}

func soundInit() {
	snd.Init(commandline.Sound() && !cvars.NoSound.Bool())
	onVolumeChange(cvars.Volume)
}

/*
func cfloatToVec3(f *C.float) vec.Vec3 {
	a := C.cf(0, f)
	b := C.cf(1, f)
	c := C.cf(2, f)
	return vec.Vec3{float32(a), float32(b), float32(c)}
}

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
}*/

func localSound(name string) {
	// This is mostly for the menu sounds
	sfx := snd.PrecacheSound(name)
	snd.Start(cl.viewentity, -1, sfx, vec.Vec3{}, 1, 1, !loopingSound)
}

func init() {
	addCommand("play", playCmd)
	addCommand("playvol", playVolCmd)
	addCommand("stopsound", stopSoundCmd)
	addCommand("soundlist", soundListCmd)
	addCommand("soundinfo", soundInfoCmd)
}

func playCmd(args []cmd.QArg, _ int) error {
	log.Println("play CMD from snd")
	return nil
}
func playVolCmd(args []cmd.QArg, _ int) error {
	log.Println("play vol CMD from snd")
	return nil
}
func stopSoundCmd(args []cmd.QArg, _ int) error {
	log.Println("stop sound CMD from snd")
	return nil
}
func soundListCmd(args []cmd.QArg, _ int) error {
	log.Println("sound list CMD from snd")
	return nil
}
func soundInfoCmd(args []cmd.QArg, _ int) error {
	log.Println("sound info CMD from snd")
	return nil
}
