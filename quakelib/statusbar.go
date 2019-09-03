package quakelib

//void Sbar_LoadPics(void);
import "C"

import (
	"fmt"
	"math/bits"
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

type spic struct {
	pic []*QPic
	x   int
	y   int
}

type Statusbar struct {
	// if >= vid.numpages, no update needed -- this needs rework
	updates    int
	showScores bool

	// mapping from display order to server side of cl.scores
	// cl.scores[sortByFrags[display element nr]]
	sortByFrags []int

	// all pictures
	nums  [2][11]*QPic
	colon *QPic
	slash *QPic
	items map[int]spic //
	ammo  [4]*QPic
	armor [3]*QPic
	sigil [4]*QPic

	faces             [7][2]*QPic
	face_invis        *QPic
	face_invuln       *QPic
	face_invis_invuln *QPic
	face_quad         *QPic

	sbar     *QPic
	ibar     *QPic //
	scorebar *QPic

	hweapons [7][5]*QPic
	hitems   [2]*QPic

	rinvbar   [2]*QPic //
	rweapons  [5]*QPic
	ritems    [2]*QPic
	rteambord *QPic
	rammo     [3]*QPic
}

//sortFrags updates s.sortByFrags to have descending frag counts
func (s *Statusbar) sortFrags() {
	// There are no more than 16 elements, so performance does not matter
	if cap(s.sortByFrags) < 16 {
		s.sortByFrags = make([]int, 16)
	}
	s.sortByFrags = s.sortByFrags[:0]
	for i, sc := range cl.scores {
		if len(sc.name) > 0 {
			s.sortByFrags = append(s.sortByFrags, i)
		}
	}
	for i := 0; i < len(s.sortByFrags); i++ {
		for j := 0; j < len(s.sortByFrags)-1-i; j++ {
			if cl.scores[s.sortByFrags[j]].frags < cl.scores[s.sortByFrags[j+1]].frags {
				s.sortByFrags[j], s.sortByFrags[j+1] = s.sortByFrags[j+1], s.sortByFrags[j]
			}
		}
	}
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

	getw := func(s string) []*QPic {
		return []*QPic{
			GetPictureFromWad("inv_" + s),
			GetPictureFromWad("inv2_" + s),
			GetPictureFromWad("inva1_" + s),
			GetPictureFromWad("inva2_" + s),
			GetPictureFromWad("inva3_" + s),
			GetPictureFromWad("inva4_" + s),
			GetPictureFromWad("inva5_" + s),
			GetPictureFromWad("inva6_" + s),
		}
	}
	s.items = make(map[int]spic)
	s.items[progs.ItemShotgun] = spic{
		pic: getw("shotgun"),
		x:   0 * 24,
		y:   8,
	}
	s.items[progs.ItemSuperShotgun] = spic{
		pic: getw("sshotgun"),
		x:   1 * 24,
		y:   8,
	}
	s.items[progs.ItemNailgun] = spic{
		pic: getw("nailgun"),
		x:   2 * 24,
		y:   8,
	}
	s.items[progs.ItemSuperNailgun] = spic{
		pic: getw("snailgun"),
		x:   3 * 24,
		y:   8,
	}
	s.items[progs.ItemGrenadeLauncher] = spic{
		pic: getw("rlaunch"),
		x:   4 * 24,
		y:   8,
	}
	s.items[progs.ItemRocketLauncher] = spic{
		pic: getw("srlaunch"),
		x:   4 * 24,
		y:   8,
	}
	s.items[progs.ItemLightning] = spic{
		pic: getw("lightng"),
		x:   5 * 24,
		y:   8,
	}

	s.ammo[0] = GetPictureFromWad("sb_shells")
	s.ammo[1] = GetPictureFromWad("sb_nails")
	s.ammo[2] = GetPictureFromWad("sb_rocket")
	s.ammo[3] = GetPictureFromWad("sb_cells")

	s.armor[0] = GetPictureFromWad("sb_armor1")
	s.armor[1] = GetPictureFromWad("sb_armor2")
	s.armor[2] = GetPictureFromWad("sb_armor3")

	s.items[progs.ItemKey1] = spic{
		pic: []*QPic{GetPictureFromWad("sb_key1")},
		x:   192 + 0*16,
		y:   8,
	}
	s.items[progs.ItemKey2] = spic{
		pic: []*QPic{GetPictureFromWad("sb_key2")},
		x:   192 + 1*16,
		y:   8,
	}
	s.items[progs.ItemInvisibility] = spic{
		pic: []*QPic{GetPictureFromWad("sb_invis")},
		x:   192 + 2*16,
		y:   8,
	}
	s.items[progs.ItemInvulnerability] = spic{
		pic: []*QPic{GetPictureFromWad("sb_invuln")},
		x:   192 + 3*16,
		y:   8,
	}
	s.items[progs.ItemSuit] = spic{
		pic: []*QPic{GetPictureFromWad("sb_suit")},
		x:   192 + 4*16,
		y:   8,
	}
	s.items[progs.ItemQuad] = spic{
		pic: []*QPic{GetPictureFromWad("sb_quad")},
		x:   192 + 5*16,
		y:   8,
	}

	if !cmdl.Rogue() {
		s.items[progs.ItemSigil1] = spic{
			pic: []*QPic{GetPictureFromWad("sb_sigil1")},
			x:   288 + 0*8,
			y:   8,
		}
		s.items[progs.ItemSigil2] = spic{
			pic: []*QPic{GetPictureFromWad("sb_sigil2")},
			x:   288 + 1*8,
			y:   8,
		}
		s.items[progs.ItemSigil3] = spic{
			pic: []*QPic{GetPictureFromWad("sb_sigil3")},
			x:   288 + 2*8,
			y:   8,
		}
		s.items[progs.ItemSigil4] = spic{
			pic: []*QPic{GetPictureFromWad("sb_sigil4")},
			x:   288 + 3*8,
			y:   8,
		}
	} else {
		// These collide with ItemSigil2/ItemSigil3
		s.items[progs.RogueItemShield] = spic{
			pic: []*QPic{GetPictureFromWad("r_shield1")},
			x:   288 + 0*16,
			y:   8,
		}
		s.items[progs.RogueItemAntigrav] = spic{
			pic: []*QPic{GetPictureFromWad("r_agrav1")},
			x:   288 + 1*16,
			y:   8,
		}
	}

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

		s.items[progs.HipnoticItemWetsuit] = spic{
			pic: []*QPic{GetPictureFromWad("sb_wsuit")},
			x:   288 + 0*16,
			y:   8,
		}
		s.items[progs.HipnoticItemEmpathyShields] = spic{
			pic: []*QPic{GetPictureFromWad("sb_eshld")},
			x:   288 + 1*16,
			y:   8,
		}
	}

	if cmdl.Rogue() {
		s.rinvbar[0] = GetPictureFromWad("r_invbar1")
		s.rinvbar[1] = GetPictureFromWad("r_invbar2")

		s.rweapons[0] = GetPictureFromWad("r_lava")
		s.rweapons[1] = GetPictureFromWad("r_superlava")
		s.rweapons[2] = GetPictureFromWad("r_gren")
		s.rweapons[3] = GetPictureFromWad("r_multirock")
		s.rweapons[4] = GetPictureFromWad("r_plasma")

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
	l := len(str) * 8
	if l < width {
		DrawStringWhite(x+(width-l)/2, y, str)
		return
	}

	scale := cvars.ScreenStatusbarScale.Value()
	scale = math.Clamp32(1.0, scale, float32(viewport.width)/320.0)
	left := float32(x) * scale
	if cl.gameType != svc.GameDeathmatch {
		left += (float32(viewport.width) - 320.0*scale) / 2
	}

	// TODO: there rest should probably go into draw.go as helper function
	gl.Enable(gl.SCISSOR_TEST)
	gl.Scissor(int32(left), 0, int32(float32(width)*scale), int32(viewport.height))

	ps := fmt.Sprintf("%s /// %s", str, str)
	l = (len(str) + 5) * 8
	ofs := int(host.time*30) % l
	DrawStringWhite(x-ofs, y, ps)

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

	for _, weapon := range []int{
		progs.ItemShotgun,
		progs.ItemSuperShotgun,
		progs.ItemNailgun,
		progs.ItemSuperNailgun,
		progs.ItemGrenadeLauncher,
		progs.ItemRocketLauncher,
		progs.ItemLightning,
	} {
		if cl.items&uint32(weapon) == 0 {
			continue
		}
		frame := 0
		if cl.stats.weapon == weapon {
			frame = 1
		}
		b := bits.TrailingZeros32(uint32(weapon))
		if f := int((cl.time - cl.itemGetTime[b]) * 10); f < 10 {
			frame = (f % 5) + 2
			// force update to remove flash
			s.MarkChanged()
		}
		w := s.items[weapon]
		DrawPicture(w.x, w.y, w.pic[frame])
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
			DrawCharacterWhite((6*pos+1)*8+2, 0, 18+v)
		}
		v = val / 10
		val -= v * 10
		p = p || v != 0
		if p {
			DrawCharacterWhite((6*pos+2)*8+2, 0, 18+v)
		}
		v = val
		DrawCharacterWhite((6*pos+3)*8+2, 0, 18+v)
	}
	drawAmmo(cl.stats.shells, 0)
	drawAmmo(cl.stats.nails, 1)
	drawAmmo(cl.stats.rockets, 2)
	drawAmmo(cl.stats.cells, 3)

	//other items
	items := []int{
		progs.ItemInvisibility,
		progs.ItemInvulnerability,
		progs.ItemSuit,
		progs.ItemQuad,
	}
	if !cmdl.Hipnotic() {
		items = append(items, progs.ItemKey1, progs.ItemKey2)
	} else {
		items = append(items, progs.HipnoticItemWetsuit, progs.HipnoticItemEmpathyShields)
	}
	if cmdl.Rogue() {
		items = append(items, progs.RogueItemShield, progs.RogueItemAntigrav)
	} else {
		items = append(items, progs.ItemSigil1, progs.ItemSigil2, progs.ItemSigil3, progs.ItemSigil4)
	}

	for _, item := range items {
		if cl.items&uint32(item) == 0 {
			continue
		}
		i := s.items[item]
		DrawPicture(i.x, i.y, i.pic[0])
		b := bits.TrailingZeros32(uint32(item))
		if t := cl.itemGetTime[b]; t != 0 && t > cl.time-2 {
			s.MarkChanged()
		}
	}
}

func toPalette(c int) int {
	return (c << 4) + 8
}

//export Sbar_DrawFrags
func Sbar_DrawFrags() {
	statusbar.drawFrags()
}

//export Sbar_DrawFace
func Sbar_DrawFace() {
	if cmdl.Rogue() && cl.maxClients != 1 && cvars.TeamPlay.Value() > 3 && cvars.TeamPlay.Value() < 7 {
		// draw some scores
	} else {
		statusbar.drawFace()
	}
}

//export Sbar_SoloScoreboard
func Sbar_SoloScoreboard() {
	statusbar.soloScoreboard()
}

//export Sbar_FinaleOverlay
func Sbar_FinaleOverlay() {
	statusbar.finaleOverlay()
}

func (s *Statusbar) drawFrags() {
	s.sortFrags()
	x := 190
	for i, f := range s.sortByFrags {
		if i >= 4 {
			return
		}
		score := cl.scores[f]
		DrawFill(x+4, 1, 28, 4, toPalette(score.topColor), 1)
		DrawFill(x+4, 5, 28, 3, toPalette(score.bottomColor), 1)
		DrawStringWhite(x+6, 0, fmt.Sprintf("%3d", score.frags))
		if f == cl.viewentity-1 {
			// mark the local player
			DrawCharacterWhite(x, 0, 16)
			DrawCharacterWhite(x+26, 0, 17)
		}

		x += 32
	}
}

func (s *Statusbar) drawFace() {
	getFace := func() *QPic {
		switch {
		case cl.items&(progs.ItemInvisibility|progs.ItemInvulnerability) != 0:
			return s.face_invis_invuln
		case cl.items&progs.ItemQuad != 0:
			return s.face_quad
		case cl.items&progs.ItemInvisibility != 0:
			return s.face_invis
		case cl.items&progs.ItemInvulnerability != 0:
			return s.face_invuln
		default:
			f := math.ClampI(0, cl.stats.health/20, 4)
			if cl.CheckFaceAnimTime() {
				s.MarkChanged() // this is an animation so force update
				return s.faces[f][1]
			}
			return s.faces[f][0]
		}
	}
	DrawPicture(112, 24, getFace())
}

func (s *Statusbar) soloScoreboard() {
	monsters := fmt.Sprintf("Kills: %d/%d", cl.stats.monsters, cl.stats.totalMonsters)
	DrawStringWhite(8, 12+24, monsters)

	secrets := fmt.Sprintf("Secrets: %d/%d", cl.stats.secrets, cl.stats.totalSecrets)
	DrawStringWhite(312-len(secrets)*8, 12+24, secrets)

	if !cmdl.Fitz() {
		skill := fmt.Sprintf("skill %d", int(cvars.Skill.Value()+0.5))
		DrawStringWhite(160-len(skill)*4, 12+24, skill)

		currMap := fmt.Sprintf("%s (%s)", cl.levelName, cl.mapName)
		s.DrawScrollString(0, 4+24, 320, currMap)
		return
	}

	minutes := cl.time / 60
	seconds := cl.time - 60*minutes
	tens := seconds / 10
	units := seconds - 10*tens
	currTime := fmt.Sprintf("%d:%d%d", minutes, tens, units)
	DrawStringWhite(160-len(currTime)*4, 12+24, currTime)

	s.DrawScrollString(0, 4+24, 320, cl.levelName)
}

func (s *Statusbar) finaleOverlay() {
	SetCanvas(CANVAS_MENU)

	pic := GetCachedPicture("gfx/finale.lmp")
	DrawPicture((320-pic.width)/2, 16, pic)
}
