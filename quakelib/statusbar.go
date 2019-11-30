package quakelib

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
	statusbar qstatusbar
)

//export Sbar_Changed
func Sbar_Changed() {
	statusbar.MarkChanged()
}

//export Sbar_Init
func Sbar_Init() {
	statusbar.LoadPictures()
}

//export Sbar_LoadPics
func Sbar_LoadPics() {
	statusbar.LoadPictures()
}

//export Sbar_Lines
func Sbar_Lines() int {
	return statusbar.Lines()
}

//export Sbar_IntermissionOverlay
func Sbar_IntermissionOverlay() {
	statusbar.IntermissionOverlay()
}

//export Sbar_FinaleOverlay
func Sbar_FinaleOverlay() {
	statusbar.FinaleOverlay()
}

//export Sbar_Draw
func Sbar_Draw() {
	statusbar.Draw()
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

type qstatusbar struct {
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
	items map[int]spic
	ammo  [4]*QPic
	armor [3]*QPic
	sigil [4]*QPic

	faces             [7][2]*QPic
	face_invis        *QPic
	face_invuln       *QPic
	face_invis_invuln *QPic
	face_quad         *QPic

	sbar     *QPic
	ibar     *QPic
	scorebar *QPic

	hweapons [5]spic
	hitems   [2]*QPic

	rinvbar   [2]*QPic
	rweapons  map[int]spic
	ritems    [2]*QPic
	rteambord *QPic
	rammo     [3]*QPic

	disc *QPic
}

//sortFrags updates s.sortByFrags to have descending frag counts
func (s *qstatusbar) sortFrags() {
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

func (s *qstatusbar) LoadPictures() {
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

	s.disc = GetPictureFromWad("disc")

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
		s.hweapons[0] = spic{
			pic: getw("laser"),
			x:   176 + 0*24,
			y:   8,
		}
		s.hweapons[1] = spic{
			pic: getw("mjolnir"),
			x:   176 + 1*24,
			y:   8,
		}
		s.hweapons[2] = spic{
			pic: getw("gren_prox"),
			x:   96,
			y:   8,
		}
		s.hweapons[3] = spic{
			pic: getw("prox_gren"),
			x:   96,
			y:   8,
		}
		s.hweapons[4] = spic{
			pic: getw("prox"),
			x:   96,
			y:   8,
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

		s.rweapons[progs.RogueItemLavaNailgun] = spic{
			pic: []*QPic{GetPictureFromWad("r_lava")},
			x:   (0 + 2) * 24,
			y:   8,
		}
		s.rweapons[progs.RogueItemLavaSuperNailgun] = spic{
			pic: []*QPic{GetPictureFromWad("r_superlava")},
			x:   (1 + 2) * 24,
			y:   8,
		}
		s.rweapons[progs.RogueItemMultiGrenade] = spic{
			pic: []*QPic{GetPictureFromWad("r_gren")},
			x:   (2 + 2) * 24,
			y:   8,
		}
		s.rweapons[progs.RogueItemMultiRocket] = spic{
			pic: []*QPic{GetPictureFromWad("r_multirock")},
			x:   (3 + 2) * 24,
			y:   8,
		}
		s.rweapons[progs.RogueItemPlasmaGun] = spic{
			pic: []*QPic{GetPictureFromWad("r_plasma")},
			x:   (4 + 2) * 24,
			y:   8,
		}

		s.rteambord = GetPictureFromWad("r_teambord")

		s.rammo[0] = GetPictureFromWad("r_ammolava")
		s.rammo[1] = GetPictureFromWad("r_ammomulti")
		s.rammo[2] = GetPictureFromWad("r_ammoplasma")
	}
}

func (s *qstatusbar) ShowScores() {
	if s.showScores {
		return
	}
	s.showScores = true
	s.updates = 0
}

func (s *qstatusbar) HideScores() {
	if !s.showScores {
		return
	}
	s.showScores = false
	s.updates = 0
}

// MarkChanged marks the statusbar to update during the next frame
func (s *qstatusbar) MarkChanged() {
	s.updates = 0
}

func StatusbarChanged() {
	statusbar.MarkChanged()
}

// scroll the string inside a glscissor region
func (s *qstatusbar) DrawScrollString(x, y, width int, str string) {
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

func (s *qstatusbar) drawInventory() {
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
		  if cmdl.Hipnotic() {
		    int grenadeflashing = 0;
		    for (i = 0; i < 4; i++) {
					//hipweapons: [23,7,4,16]
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
	*/

	if cmdl.Rogue() {
		for k, v := range s.rweapons {
			if cl.stats.activeWeapon == k {
				DrawPicture(v.x, v.y, v.pic[0])
			}
		}
	}

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

func (s *qstatusbar) Lines() int {
	// scan lines to draw
	size := cvars.ViewSize.Value()
	scale := math.Clamp32(1, cvars.ScreenStatusbarScale.Value(), float32(viewport.width)/320)
	if size >= 120 || cl.intermission != 0 || cvars.ScreenStatusbarAlpha.Value() < 1 {
		return 0
	} else if size >= 110 {
		return int(24 * scale)
	}
	return int(48 * scale)
}

func (s *qstatusbar) drawFrags() {
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

func (s *qstatusbar) drawFace() {
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

func (s *qstatusbar) drawAmmo() {
	color := 0
	if cl.stats.ammo <= 10 {
		color = 1
	}
	s.drawNumber(248, 24, cl.stats.ammo, color)

	d := func(ammo *QPic) { DrawPicture(224, 24, ammo) }
	if cmdl.Rogue() {
		switch {
		case cl.items&progs.RogueItemShells != 0:
			d(s.ammo[0])
		case cl.items&progs.RogueItemNails != 0:
			d(s.ammo[1])
		case cl.items&progs.RogueItemRockets != 0:
			d(s.ammo[2])
		case cl.items&progs.RogueItemCells != 0:
			d(s.ammo[3])
		case cl.items&progs.RogueItemLavaNails != 0:
			d(s.rammo[0])
		case cl.items&progs.RogueItemPlasmaAmmo != 0:
			d(s.rammo[1])
		case cl.items&progs.RogueItemMultiRockets != 0:
			d(s.rammo[2])
		}
		return
	}
	switch {
	case cl.items&progs.ItemShells != 0:
		d(s.ammo[0])
	case cl.items&progs.ItemNails != 0:
		d(s.ammo[1])
	case cl.items&progs.ItemRockets != 0:
		d(s.ammo[2])
	case cl.items&progs.ItemCells != 0:
		d(s.ammo[3])
	}
}

func (s *qstatusbar) drawHealth() {
	color := 0
	if cl.stats.health <= 25 {
		color = 1
	}
	s.drawNumber(136, 24, cl.stats.health, color)
}

func (s *qstatusbar) drawNumber(x, y, num, color int) {
	if num > 999 {
		num = 999
	}
	n1 := num / 10
	frame := num - n1*10
	DrawPicture(x+48, y, s.nums[color][frame])
	if n1 != 0 {
		n2 := n1 / 10
		frame = n1 - n2*10
		DrawPicture(x+24, y, s.nums[color][frame])
		if n2 != 0 {
			DrawPicture(x, y, s.nums[color][n2])
		}
	}
}

func (s *qstatusbar) drawScoreboard() {
	s.soloScoreboard()
	if cl.gameType == svc.GameDeathmatch {
		s.deathmatchOverlay()
	}
}

func (s *qstatusbar) soloScoreboard() {
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

	minutes := int(cl.time) / 60
	seconds := int(cl.time) - 60*minutes
	tens := seconds / 10
	units := seconds - 10*tens
	currTime := fmt.Sprintf("%d:%d%d", minutes, tens, units)
	DrawStringWhite(160-len(currTime)*4, 12+24, currTime)

	s.DrawScrollString(0, 4+24, 320, cl.levelName)
}

func (s *qstatusbar) deathmatchOverlay() {
	SetCanvas(CANVAS_MENU)

	pic := GetCachedPicture("gfx/ranking.lmp")
	DrawPicture((320-pic.Width)/2, 8, pic)

	s.sortFrags()

	x := 80
	y := 40
	for _, f := range s.sortByFrags {
		score := &cl.scores[f]

		DrawFill(x, y, 40, 4, toPalette(score.topColor), 1)
		DrawFill(x, y+4, 40, 4, toPalette(score.bottomColor), 1)

		frags := score.frags
		if frags > 999 {
			frags = 999
		}
		DrawStringWhite(x+8, y, fmt.Sprintf("%3d", frags))

		if f == cl.viewentity-1 {
			DrawCharacterWhite(x-8, y, 12)
		}

		DrawStringCopper(x+64, y, score.name)

		y += 10
	}

	SetCanvas(CANVAS_STATUSBAR)
}

func (s *qstatusbar) IntermissionOverlay() {
	if cl.gameType == svc.GameDeathmatch {
		s.deathmatchOverlay()
		return
	}
	SetCanvas(CANVAS_MENU)

	DrawPicture(64, 24, GetCachedPicture("gfx/complete.lmp"))
	DrawPicture(0, 56, GetCachedPicture("gfx/inter.lmp"))

	dig := cl.intermissionTime / 60
	s.drawNumber(152, 64, dig, 0)
	num := cl.intermissionTime - dig*60
	DrawPicture(224, 64, s.colon)
	DrawPicture(240, 64, s.nums[0][num/10])
	DrawPicture(264, 64, s.nums[0][num%10])

	s.drawNumber(152, 104, cl.stats.secrets, 0)
	DrawPicture(224, 104, s.slash)
	s.drawNumber(240, 104, cl.stats.totalSecrets, 0)

	s.drawNumber(152, 144, cl.stats.monsters, 0)
	DrawPicture(224, 144, s.slash)
	s.drawNumber(240, 144, cl.stats.totalMonsters, 0)
}

func (s *qstatusbar) FinaleOverlay() {
	SetCanvas(CANVAS_MENU)

	pic := GetCachedPicture("gfx/finale.lmp")
	DrawPicture((320-pic.Width)/2, 16, pic)
}

func (s *qstatusbar) drawArmor() {
	d := func(armorPicture *QPic) { DrawPicture(0, 24, armorPicture) }
	if cl.items&progs.ItemInvulnerability != 0 {
		s.drawNumber(24, 24, 666, 1)
		d(s.disc)
		return
	}
	color := 0
	if cl.stats.armor <= 25 {
		color = 1
	}
	s.drawNumber(24, 24, cl.stats.armor, color)
	if cmdl.Rogue() {
		switch {
		case cl.items&progs.RogueItemArmor3 != 0:
			d(s.armor[2])
		case cl.items&progs.RogueItemArmor2 != 0:
			d(s.armor[1])
		case cl.items&progs.RogueItemArmor1 != 0:
			d(s.armor[0])
		}
		return
	}
	switch {
	case cl.items&progs.ItemArmor3 != 0:
		d(s.armor[2])
	case cl.items&progs.ItemArmor2 != 0:
		d(s.armor[1])
	case cl.items&progs.ItemArmor1 != 0:
		d(s.armor[0])
	}
}

func (s *qstatusbar) miniDeathmatchOverlay() {
	scale := math.Clamp32(1.0,
		cvars.ScreenStatusbarScale.Value(),
		float32(viewport.width)/320.0)

	// MAX_SCOREBOARDNAME = 32, so total width for this overlay plus sbar is 632,
	// but we can cut off some i guess
	if float32(viewport.width)/scale < 512 || cvars.ViewSize.Value() >= 120 {
		return
	}
	s.sortFrags()

	numLines := 6
	x := 324
	y := 0
	if cvars.ViewSize.Value() >= 110 {
		numLines = 3
		y = 24
	}
	// display the local player and ones with frag counts close by
	i := func() int {
		for i, fs := range s.sortByFrags {
			if fs == cl.viewentity-1 {
				// local player found
				return i
			}
		}
		return 0
	}()
	// move the window to have the player centered
	i -= numLines / 2
	i = math.ClampI(0, i, len(s.sortByFrags)-numLines)
	for i < len(s.sortByFrags) && y <= 48 {
		score := &cl.scores[s.sortByFrags[i]]

		DrawFill(x, y+1, 40, 4, toPalette(score.topColor), 1)
		DrawFill(x, y+5, 40, 3, toPalette(score.bottomColor), 1)

		frags := score.frags
		if frags > 999 {
			frags = 999
		}
		DrawStringWhite(x+8, y, fmt.Sprintf("%3d", frags))

		if s.sortByFrags[i] == cl.viewentity-1 {
			DrawCharacterWhite(x, y, 16)
			DrawCharacterWhite(x+32, y, 17)
		}

		DrawStringCopper(x+48, y, score.name)

		i++
		y += 8
	}
}

func (s *qstatusbar) drawCTFScores() {
	score := &cl.scores[cl.viewentity-1]

	xofs := 113
	if cl.gameType != svc.GameDeathmatch {
		xofs += (screen.Width - 320) / 2
	}

	DrawPicture(112, 24, s.rteambord)
	DrawFill(xofs, 24+3, 22, 9, toPalette(score.topColor), 1)
	DrawFill(xofs, 24+12, 22, 9, toPalette(score.bottomColor), 1)

	// TODO: should the other scores get the same copper variant?
	if score.topColor == 1 {
		// orig has only 7 pixel wide chars --- in all other places chars are 8 pixel
		DrawStringCopper(113, 3+24, fmt.Sprintf("%3d", score.frags))
	} else {
		DrawStringWhite(113, 3+24, fmt.Sprintf("%3d", score.frags))
	}
}

func (s *qstatusbar) Draw() {
	if console.currentHeight() == screen.Height {
		return
	}
	if cl.intermission != 0 {
		return
	}
	if s.updates >= screen.numPages &&
		!cvars.GlClear.Bool() &&
		cvars.ScreenStatusbarAlpha.Value() >= 1 &&
		cvars.Gamma.Value() == 1 {
		// must draw every frame if doing glsl gamma
		return
	}

	s.updates++

	SetCanvas(CANVAS_DEFAULT)

	alpha := cvars.ScreenStatusbarAlpha.Value()
	lines := s.Lines()
	vw := int(viewport.width)
	vh := int(viewport.height)
	w := math.ClampI(320, int(cvars.ScreenStatusbarScale.Value()*320), vw)
	if lines != 0 && vw > w {
		if alpha < 1 {
			// #############
			DrawTileClear(0, vh-lines, vw, lines)
		} else if cl.gameType == svc.GameDeathmatch {
			// ------#######
			DrawTileClear(w, vh-lines, vw-w, lines)
		} else {
			// ####-----####
			cw := (vw - w) / 2
			DrawTileClear(0, vh-lines, cw, lines)
			DrawTileClear(cw+w, vh-lines, cw, lines)
		}
	}

	SetCanvas(CANVAS_STATUSBAR)

	if cvars.ViewSize.Value() < 110 {
		s.drawInventory()
		if cl.maxClients != 1 {
			s.drawFrags()
		}
	}

	if s.showScores || cl.stats.health <= 0 {
		DrawPictureAlpha(0, 24, s.scorebar, alpha)
		s.drawScoreboard()
		s.updates = 0
	} else if cvars.ViewSize.Value() < 120 {
		DrawPictureAlpha(0, 24, s.sbar, alpha)

		if cmdl.Hipnotic() {
			// prevent overwriting keys
			if cl.items&progs.ItemKey1 != 0 {
				DrawPicture(209, 3+24, s.items[progs.ItemKey1].pic[0])
			}
			if cl.items&progs.ItemKey2 != 0 {
				DrawPicture(209, 12+24, s.items[progs.ItemKey2].pic[0])
			}
		}
		s.drawArmor()

		if cmdl.Rogue() && cl.maxClients != 1 && cvars.TeamPlay.Value() > 3 && cvars.TeamPlay.Value() < 7 {
			// draw some scores
			s.drawCTFScores()
		} else {
			s.drawFace()
		}
		s.drawHealth()
		s.drawAmmo()
	}

	if cl.gameType == svc.GameDeathmatch {
		s.miniDeathmatchOverlay()
	}
}
