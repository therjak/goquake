package quakelib

import (
	"testing"

	"goquake/cvars"
	"goquake/math/vec"
	qsnd "goquake/snd"
)

func TestSoundNilSafety(t *testing.T) {
	// Ensure snd is (*qsnd.SndSys)(nil) and defaultSounds is nil
	origSnd := snd
	origDefaultSounds := defaultSounds
	snd = (*qsnd.SndSys)(nil)
	defaultSounds = nil
	t.Cleanup(func() {
		snd = origSnd
		defaultSounds = origDefaultSounds
	})

	// Changing volume cvar should not panic when snd is (*qsnd.SndSys)(nil)
	cvars.Volume.SetValue(0.5)
	cvars.Volume.SetValue(1.5)
	cvars.Volume.SetValue(-0.5)

	// Direct calls to snd methods on nil receiver should not panic
	snd.Stop(0, 0)
	snd.StopAll()
	snd.Update(0, vec.Vec3{}, vec.Vec3{})
	snd.Shutdown()
	snd.Unblock()
	snd.Block()
	snd.SetVolume(0.5)
	precache := snd.NewPrecache(qsnd.Sound{ID: 0, Name: "test"})
	if precache != nil {
		t.Errorf("expected nil precache from nil SndSys, got %v", precache)
	}

	// localSound and clientSound should not panic when defaultSounds is nil
	localSound(lsMenu1)
	clientSound(lsMenu1, vec.Vec3{})

	// SoundPrecache methods on nil receiver should not panic
	var nilPrecache *qsnd.SoundPrecache
	nilPrecache.Start(0, 0, 0, vec.Vec3{}, 1.0, 1.0)
	nilPrecache.StartAmbient(0, vec.Vec3{}, 1.0, 1.0)
}
