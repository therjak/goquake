package quakelib

//void Sbar_LoadPics(void);
import "C"

import (
	"fmt"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/cvars"
	"quake/math"
	"quake/progs"
	svc "quake/protocol/server"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	statusbar Statusbar
)

//export SBResetUpdates
func SBResetUpdates() {
	statusbar.MarkChanged()
}

//export SBUpdatesInc
func SBUpdatesInc() {
	statusbar.updates++
}

//export SBUpdates
func SBUpdates() int {
	return statusbar.updates
}

//export Sbar_Changed
func Sbar_Changed() {
	statusbar.MarkChanged()
}

//export Sbar_DoesShowScores
func Sbar_DoesShowScores() bool {
	return statusbar.showScores
}

//export Sbar_Init
func Sbar_Init() {
	statusbar.LoadPictures()
	C.Sbar_LoadPics()
}

//export Sbar_DrawInventory
func Sbar_DrawInventory() {
	statusbar.DrawInventory()
}

func init() {
	cmd.AddCommand("+showscores", func(_ []cmd.QArg, _ int) { statusbar.ShowScores() })
	cmd.AddCommand("-showscores", func(_ []cmd.QArg, _ int) { statusbar.HideScores() })
}

type Statusbar struct {
	// if >= vid.numpages, no update needed -- this needs rework
	updates    int
	showScores bool

	// all pictures
	nums    [2][11]*QPic
	colon   *QPic
	slash   *QPic
	weapons [7][8]*QPic
	ammo    [4]*QPic
	armor   [3]*QPic
	items   [32]*QPic
	sigil   [4]*QPic

	faces             [7][2]*QPic
	face_invis        *QPic
	face_invuln       *QPic
	face_invis_invuln *QPic
	face_quad         *QPic

	sbar     *QPic
	ibar     *QPic
	scorebar *QPic

	hweapons [7][5]*QPic
	hitems   [2]*QPic

	rinvbar   [2]*QPic
	rweapons  [5]*QPic
	ritems    [2]*QPic
	rteambord *QPic
	rammo     [3]*QPic
}

func (s *Statusbar) LoadPictures() {
	for i := 0; i < 10; i++ {
		s.nums[0][i] = GetPictureFromWad(fmt.Sprintf("num_%d", i))
		s.nums[1][i] = GetPictureFromWad(fmt.Sprintf("anum_%d", i))
	}

	s.nums[0][10] = GetPictureFromWad("num_minus")
	s.nums[1][10] = GetPictureFromWad("anum_minus")

	s.colon = GetPictureFromWad("num_colon")
	s.slash = GetPictureFromWad("num_slash")

	s.weapons[0][0] = GetPictureFromWad("inv_shotgun")
	s.weapons[0][1] = GetPictureFromWad("inv_sshotgun")
	s.weapons[0][2] = GetPictureFromWad("inv_nailgun")
	s.weapons[0][3] = GetPictureFromWad("inv_snailgun")
	s.weapons[0][4] = GetPictureFromWad("inv_rlaunch")
	s.weapons[0][5] = GetPictureFromWad("inv_srlaunch")
	s.weapons[0][6] = GetPictureFromWad("inv_lightng")

	s.weapons[1][0] = GetPictureFromWad("inv2_shotgun")
	s.weapons[1][1] = GetPictureFromWad("inv2_sshotgun")
	s.weapons[1][2] = GetPictureFromWad("inv2_nailgun")
	s.weapons[1][3] = GetPictureFromWad("inv2_snailgun")
	s.weapons[1][4] = GetPictureFromWad("inv2_rlaunch")
	s.weapons[1][5] = GetPictureFromWad("inv2_srlaunch")
	s.weapons[1][6] = GetPictureFromWad("inv2_lightng")

	for i := 0; i < 5; i++ {
		s.weapons[2+i][0] = GetPictureFromWad(fmt.Sprintf("inva%d_shotgun", i+1))
		s.weapons[2+i][1] = GetPictureFromWad(fmt.Sprintf("inva%d_sshotgun", i+1))
		s.weapons[2+i][2] = GetPictureFromWad(fmt.Sprintf("inva%d_nailgun", i+1))
		s.weapons[2+i][3] = GetPictureFromWad(fmt.Sprintf("inva%d_snailgun", i+1))
		s.weapons[2+i][4] = GetPictureFromWad(fmt.Sprintf("inva%d_rlaunch", i+1))
		s.weapons[2+i][5] = GetPictureFromWad(fmt.Sprintf("inva%d_srlaunch", i+1))
		s.weapons[2+i][6] = GetPictureFromWad(fmt.Sprintf("inva%d_lightng", i+1))
	}

	s.ammo[0] = GetPictureFromWad("sb_shells")
	s.ammo[1] = GetPictureFromWad("sb_nails")
	s.ammo[2] = GetPictureFromWad("sb_rocket")
	s.ammo[3] = GetPictureFromWad("sb_cells")

	s.armor[0] = GetPictureFromWad("sb_armor1")
	s.armor[1] = GetPictureFromWad("sb_armor2")
	s.armor[2] = GetPictureFromWad("sb_armor3")

	s.items[0] = GetPictureFromWad("sb_key1")
	s.items[1] = GetPictureFromWad("sb_key2")
	s.items[2] = GetPictureFromWad("sb_invis")
	s.items[3] = GetPictureFromWad("sb_invuln")
	s.items[4] = GetPictureFromWad("sb_suit")
	s.items[5] = GetPictureFromWad("sb_quad")

	s.sigil[0] = GetPictureFromWad("sb_sigil1")
	s.sigil[1] = GetPictureFromWad("sb_sigil2")
	s.sigil[2] = GetPictureFromWad("sb_sigil3")
	s.sigil[3] = GetPictureFromWad("sb_sigil4")

	s.faces[4][0] = GetPictureFromWad("face1")
	s.faces[4][1] = GetPictureFromWad("face_p1")
	s.faces[3][0] = GetPictureFromWad("face2")
	s.faces[3][1] = GetPictureFromWad("face_p2")
	s.faces[2][0] = GetPictureFromWad("face3")
	s.faces[2][1] = GetPictureFromWad("face_p3")
	s.faces[1][0] = GetPictureFromWad("face4")
	s.faces[1][1] = GetPictureFromWad("face_p4")
	s.faces[0][0] = GetPictureFromWad("face5")
	s.faces[0][1] = GetPictureFromWad("face_p5")

	s.face_invis = GetPictureFromWad("face_invis")
	s.face_invuln = GetPictureFromWad("face_invul2")
	s.face_invis_invuln = GetPictureFromWad("face_inv2")
	s.face_quad = GetPictureFromWad("face_quad")

	s.sbar = GetPictureFromWad("sbar")
	s.ibar = GetPictureFromWad("ibar")
	s.scorebar = GetPictureFromWad("scorebar")

	if cmdl.Hipnotic() {
		s.hweapons[0][0] = GetPictureFromWad("inv_laser")
		s.hweapons[0][1] = GetPictureFromWad("inv_mjolnir")
		s.hweapons[0][2] = GetPictureFromWad("inv_gren_prox")
		s.hweapons[0][3] = GetPictureFromWad("inv_prox_gren")
		s.hweapons[0][4] = GetPictureFromWad("inv_prox")

		s.hweapons[1][0] = GetPictureFromWad("inv2_laser")
		s.hweapons[1][1] = GetPictureFromWad("inv2_mjolnir")
		s.hweapons[1][2] = GetPictureFromWad("inv2_gren_prox")
		s.hweapons[1][3] = GetPictureFromWad("inv2_prox_gren")
		s.hweapons[1][4] = GetPictureFromWad("inv2_prox")

		for i := 0; i < 5; i++ {
			s.hweapons[2+i][0] = GetPictureFromWad(fmt.Sprintf("inva%d_laser", i+1))
			s.hweapons[2+i][1] = GetPictureFromWad(fmt.Sprintf("inva%d_mjolnir", i+1))
			s.hweapons[2+i][2] = GetPictureFromWad(fmt.Sprintf("inva%d_gren_prox", i+1))
			s.hweapons[2+i][3] = GetPictureFromWad(fmt.Sprintf("inva%d_prox_gren", i+1))
			s.hweapons[2+i][4] = GetPictureFromWad(fmt.Sprintf("inva%d_prox", i+1))
		}

		s.hitems[0] = GetPictureFromWad("sb_wsuit")
		s.hitems[1] = GetPictureFromWad("sb_eshld")
	}

	if cmdl.Rogue() {
		s.rinvbar[0] = GetPictureFromWad("r_invbar1")
		s.rinvbar[1] = GetPictureFromWad("r_invbar2")

		s.rweapons[0] = GetPictureFromWad("r_lava")
		s.rweapons[1] = GetPictureFromWad("r_superlava")
		s.rweapons[2] = GetPictureFromWad("r_gren")
		s.rweapons[3] = GetPictureFromWad("r_multirock")
		s.rweapons[4] = GetPictureFromWad("r_plasma")

		s.ritems[0] = GetPictureFromWad("r_shield1")
		s.ritems[1] = GetPictureFromWad("r_agrav1")

		s.rteambord = GetPictureFromWad("r_teambord")

		s.rammo[0] = GetPictureFromWad("r_ammolava")
		s.rammo[1] = GetPictureFromWad("r_ammomulti")
		s.rammo[2] = GetPictureFromWad("r_ammoplasma")
	}
}

func (s *Statusbar) ShowScores() {
	if s.showScores {
		return
	}
	s.showScores = true
	s.updates = 0
}

func (s *Statusbar) HideScores() {
	if !s.showScores {
		return
	}
	s.showScores = false
	s.updates = 0
}

// MarkChanged marks the statusbar to update during the next frame
func (s *Statusbar) MarkChanged() {
	s.updates = 0
}

func StatusbarChanged() {
	statusbar.MarkChanged()
}

//export Sbar_DrawScrollString
func Sbar_DrawScrollString(x int, y int, width int, str *C.char) {
	statusbar.DrawScrollString(x, y, width, C.GoString(str))
}

// scroll the string inside a glscissor region
func (s *Statusbar) DrawScrollString(x, y, width int, str string) {

	scale := cvars.ScreenStatusbarScale.Value()
	scale = math.Clamp32(1.0, scale, float32(viewport.width)/320.0)
	left := float32(x) * scale
	if cl.gameType != svc.GameDeathmatch {
		left += (float32(viewport.width) - 320.0*scale) / 2
	}

	// TODO: there rest should probably go into draw.go as helper function
	gl.Enable(gl.SCISSOR_TEST)
	gl.Scissor(int32(left), 0, int32(float32(width)*scale), int32(viewport.height))

	len := len(str)*8 + 40
	ofs := int(host.time*30) % len
	drawString(x-ofs, y+24, str)
	DrawCharacter(x-ofs+len-32, y+24, '/')
	DrawCharacter(x-ofs+len-24, y+24, '/')
	DrawCharacter(x-ofs+len-16, y+24, '/')
	drawString(x-ofs+len, y+24, str)

	gl.Disable(gl.SCISSOR_TEST)
}

func (s *Statusbar) DrawInventory() {
	ibar := func() *QPic {
		if cmdl.Rogue() {
			if cl.stats.weapon >= progs.RogueItemLavaNailgun {
				return s.rinvbar[0]
			}
			return s.rinvbar[1]
		} else {
			return s.ibar
		}
	}()
	DrawPictureAlpha(0, 0, ibar, cvars.ScreenStatusbarAlpha.Value())

	for i := uint32(0); i < 7; i++ {
		weapon := uint32(1) << i
		if (cl.items & weapon) != 0 {
			flashon := int((cl.time - cl.itemGetTime[i]) * 10)
			if flashon >= 10 {
				if cl.stats.weapon == int(weapon) {
					flashon = 1
				} else {
					flashon = 0
				}
			} else {
				flashon = (flashon % 5) + 2
			}

			DrawPicture(int(i)*24, -16+24, s.weapons[flashon][i])

			if flashon > 1 {
				// force update to remove flash
				s.MarkChanged()
			}
		}
	}
	/*
	  if (CMLHipnotic()) {
	    int grenadeflashing = 0;
	    for (i = 0; i < 4; i++) {
	      if (CL_HasItem(1 << hipweapons[i])) {
	        time = CL_ItemGetTime(hipweapons[i]);
	        flashon = (int)((CL_Time() - time) * 10);
	        if (flashon >= 10) {
	          if (CL_Stats(STAT_ACTIVEWEAPON) == (1 << hipweapons[i]))
	            flashon = 1;
	          else
	            flashon = 0;
	        } else
	          flashon = (flashon % 5) + 2;

	        // check grenade launcher
	        if (i == 2) {
	          if (CL_HasItem(HIT_PROXIMITY_GUN)) {
	            if (flashon) {
	              grenadeflashing = 1;
	              Draw_Pic(96, -16 + 24, hsb_weapons[flashon][2]);
	            }
	          }
	        } else if (i == 3) {
	          if (CL_HasItem(IT_SHOTGUN << 4)) {
	            if (flashon && !grenadeflashing) {
	              Draw_Pic(96, -16 + 24, hsb_weapons[flashon][3]);
	            } else if (!grenadeflashing) {
	              Draw_Pic(96, -16 + 24, hsb_weapons[0][3]);
	            }
	          } else
	            Draw_Pic(96, -16 + 24, hsb_weapons[flashon][4]);
	        } else
	          Draw_Pic(176 + (i * 24), -16 + 24, hsb_weapons[flashon][i]);

	        if (flashon > 1) {
	          // force update to remove flash
	          SBResetUpdates();
	        }
	      }
	    }
	  }

	  if (CMLRogue()) {
	    // check for powered up weapon.
	    if (CL_Stats(STAT_ACTIVEWEAPON) >= RIT_LAVA_NAILGUN) {
	      for (i = 0; i < 5; i++) {
	        if (CL_Stats(STAT_ACTIVEWEAPON) == (RIT_LAVA_NAILGUN << i)) {
	          Draw_Pic((i + 2) * 24, -16 + 24, rsb_weapons[i]);
	        }
	      }
	    }
	  }
	*/

	// ammo counts
	drawAmmo := func(num int, pos int) {
		val := math.ClampI(0, num, 999)
		v := val / 100
		val -= v * 100
		p := v != 0
		if p {
			DrawCharacter((6*pos+1)*8+2, 0, 18+v)
		}
		v = val / 10
		val -= v * 10
		p = p || v != 0
		if p {
			DrawCharacter((6*pos+2)*8+2, 0, 18+v)
		}
		v = val
		DrawCharacter((6*pos+3)*8+2, 0, 18+v)
	}
	drawAmmo(cl.stats.shells, 0)
	drawAmmo(cl.stats.nails, 1)
	drawAmmo(cl.stats.rockets, 2)
	drawAmmo(cl.stats.cells, 3)

	/*
	  flashon = 0;
	  // items
	  for (i = 0; i < 6; i++) {
	    if (CL_HasItem(1 << (17 + i))) {
	      time = CL_ItemGetTime(17 + i);
	      if (time && time > CL_Time() - 2 && flashon) {  // flash frame
	        SBResetUpdates();
	      } else {
	        if (!CMLHipnotic() || (i > 1)) {
	          Draw_Pic(192 + i * 16, -16 + 24, sb_items[i]);
	        }
	      }
	      if (time && time > CL_Time() - 2) {
	        SBResetUpdates();
	      }
	    }
	  }

	  if (CMLHipnotic()) {
	    for (i = 0; i < 2; i++) {
	      if (CL_HasItem(1 << (24 + i))) {
	        time = CL_ItemGetTime(24 + i);
	        if (time && time > CL_Time() - 2 && flashon) {  // flash frame
	          SBResetUpdates();
	        } else {
	          Draw_Pic(288 + i * 16, -16 + 24, hsb_items[i]);
	        }
	        if (time && time > CL_Time() - 2) {
	          SBResetUpdates();
	        }
	      }
	    }
	  }

	  if (CMLRogue()) {
	    // new rogue items
	    for (i = 0; i < 2; i++) {
	      if (CL_HasItem(1 << (29 + i))) {
	        time = CL_ItemGetTime(29 + i);
	        if (time && time > CL_Time() - 2 && flashon) {  // flash frame
	          SBResetUpdates();
	        } else {
	          Draw_Pic(288 + i * 16, -16 + 24, rsb_items[i]);
	        }
	        if (time && time > CL_Time() - 2) {
	          SBResetUpdates();
	        }
	      }
	    }
	  } else {
	    // sigils
	    for (i = 0; i < 4; i++) {
	      if (CL_HasItem(1 << (28 + i))) {
	        time = CL_ItemGetTime(28 + i);
	        if (time && time > CL_Time() - 2 && flashon) {  // flash frame
	          SBResetUpdates();
	        } else
	          Draw_Pic(320 - 32 + i * 8, -16 + 24, sb_sigil[i]);
	        if (time && time > CL_Time() - 2) {
	          SBResetUpdates();
	        }
	      }
	    }
	  }
	*/
}
