// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"

	"goquake/cbuf"
	"goquake/commandline"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/math/vec"
	qsnd "goquake/snd"
)

// it should support to play U8, S8 and S16 sounds (is it necessary to replicate this?)

// there are 4 ambient sound channel

type soundsystem interface {
	Start(entnum int, entchannel int, sfx int, sndOrigin vec.Vec3, fvol float32, attenuation float32, looping bool)
	Stop(entnum, entchannel int)
	StopAll()
	PrecacheSound(n string) int
	Update(id int, origin vec.Vec3, right vec.Vec3)
	Shutdown()
	Unblock()
	Block()
	SetVolume(v float32)
	NewPrecache() *qsnd.SoundPrecache
}

var snd soundsystem

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

type lSound int

var localSounds [4]int

const (
	lsMenu1 lSound = iota
	lsMenu2
	lsMenu3
	lsTalk
)

func soundInit() {
	snd = qsnd.InitSoundSystem(commandline.Sound() && !cvars.NoSound.Bool())
	onVolumeChange(cvars.Volume)
	localSounds[lsMenu1] = snd.PrecacheSound("misc/menu1.wav")
	localSounds[lsMenu2] = snd.PrecacheSound("misc/menu2.wav")
	localSounds[lsMenu3] = snd.PrecacheSound("misc/menu3.wav")
	localSounds[lsTalk] = snd.PrecacheSound("misc/talk.wav")
}

func localSound(sfx lSound) {
	snd.Start(-1, -1, localSounds[sfx], vec.Vec3{}, 1, 1, false)
}

func init() {
	addCommand("play", playCmd)
	addCommand("playvol", playVolCmd)
	addCommand("stopsound", stopSoundCmd)
	addCommand("soundlist", soundListCmd)
	addCommand("soundinfo", soundInfoCmd)
}

func playCmd(args cbuf.Arguments) error {
	log.Println("play CMD from snd")
	return nil
}
func playVolCmd(args cbuf.Arguments) error {
	log.Println("play vol CMD from snd")
	return nil
}
func stopSoundCmd(args cbuf.Arguments) error {
	log.Println("stop sound CMD from snd")
	return nil
}
func soundListCmd(args cbuf.Arguments) error {
	log.Println("sound list CMD from snd")
	return nil
}
func soundInfoCmd(args cbuf.Arguments) error {
	log.Println("sound info CMD from snd")
	return nil
}
