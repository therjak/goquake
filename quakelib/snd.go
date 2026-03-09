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
	NewPrecache(snds ...qsnd.Sound) *qsnd.SoundPrecache
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

func soundInit(stop chan struct{}) {
	if !commandline.Sound() || cvars.NoSound.Bool() {
		return
	}
	snd = qsnd.InitSoundSystem(stop)
	onVolumeChange(cvars.Volume)
	defaultSounds = snd.NewPrecache(
		qsnd.Sound{ID: int(lsMenu1), Name: "misc/menu1.wav"},
		qsnd.Sound{ID: int(lsMenu2), Name: "misc/menu2.wav"},
		qsnd.Sound{ID: int(lsMenu3), Name: "misc/menu3.wav"},
		qsnd.Sound{ID: int(lsTalk), Name: "misc/talk.wav"},
		qsnd.Sound{ID: int(WizHit), Name: "wizard/hit.wav"},
		qsnd.Sound{ID: int(KnightHit), Name: "hknight/hit.wav"},
		qsnd.Sound{ID: int(Tink1), Name: "weapons/tink1.wav"},
		qsnd.Sound{ID: int(Ric1), Name: "weapons/ric1.wav"},
		qsnd.Sound{ID: int(Ric2), Name: "weapons/ric2.wav"},
		qsnd.Sound{ID: int(Ric3), Name: "weapons/ric3.wav"},
		qsnd.Sound{ID: int(RExp3), Name: "weapons/r_exp3.wav"},
	)
}

func localSound(sfx lSound) {
	defaultSounds.Start(qsnd.Local, -1, int(sfx), vec.Vec3{}, 1, 1)
}

func clientSound(sfx lSound, pos vec.Vec3) {
	defaultSounds.Start(qsnd.Local, 0, int(sfx), pos, 1, 1)
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
