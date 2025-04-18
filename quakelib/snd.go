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
	Stop(entnum, entchannel int)
	StopAll()
	Update(id int, origin vec.Vec3, right vec.Vec3)
	Shutdown()
	Unblock()
	Block()
	SetVolume(v float32)
	NewPrecache() *qsnd.SoundPrecache
}

var (
	snd           soundsystem
	defaultSounds *qsnd.SoundPrecache
)

const (
	loopingSound = true
)

type lSound int

const (
	lsMenu1 lSound = iota
	lsMenu2
	lsMenu3
	lsTalk
	WizHit
	KnightHit
	Tink1
	Ric1
	Ric2
	Ric3
	RExp3
)

func soundInit() {
	snd = qsnd.InitSoundSystem(commandline.Sound() && !cvars.NoSound.Bool())
	onVolumeChange(cvars.Volume)
	defaultSounds = snd.NewPrecache()
	// Order must match lSound constants
	defaultSounds.Set(
		"misc/menu1.wav",
		"misc/menu2.wav",
		"misc/menu3.wav",
		"misc/talk.wav",
		"wizard/hit.wav",
		"hknight/hit.wav",
		"weapons/tink1.wav",
		"weapons/ric1.wav",
		"weapons/ric2.wav",
		"weapons/ric3.wav",
		"weapons/r_exp3.wav")
}

func localSound(sfx lSound) {
	defaultSounds.Start(-1, -1, int(sfx), vec.Vec3{}, 1, 1, false)
}

func clientSound(sfx lSound, pos vec.Vec3) {
	defaultSounds.Start(-1, 0, int(sfx), pos, 1, 1, false)
}

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
