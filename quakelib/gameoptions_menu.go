package quakelib

import (
	"fmt"
	"github.com/therjak/goquake/cbuf"
	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/cvars"
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/maps"
	"github.com/therjak/goquake/menu"
)

func enterGameOptionsMenu() {
	IN_Deactivate()
	keyDestination = keys.Menu
	qmenu.state = menu.GameOptions
	qmenu.playEnterSound = true
	gameOptionsMenu.Update()
}

var (
	gameOptionsMenu = qGameOptionsMenu{
		items: makeGameOptionsMenuItems(),
	}
)

type mapSelector struct {
	episodes []maps.Episode
	episode  int
	level    int
}

func (m *mapSelector) Accept() {
	cbuf.AddText(fmt.Sprintf("map %s\n", m.Level().ID))
}

func (m *mapSelector) NextLevel() {
	e := &m.Episode().Maps
	m.level = (m.level + 1) % len(*e)
}

func (m *mapSelector) PreviousLevel() {
	e := &m.Episode().Maps
	m.level = (m.level + len(*e) - 1) % len(*e)
}

func (m *mapSelector) NextEpisode() {
	m.level = 0
	m.episode = (m.episode + 1) % len(m.episodes)
}

func (m *mapSelector) PreviousEpisode() {
	m.level = 0
	m.episode = (m.episode + len(m.episodes) - 1) % len(m.episodes)
}

func (m *mapSelector) Episode() *maps.Episode {
	return &m.episodes[m.episode]
}

func (m *mapSelector) Level() *maps.Map {
	return &m.Episode().Maps[m.level]
}

func NewMapSelector() *mapSelector {
	e := func() []maps.Episode {
		if cmdl.Rogue() {
			return maps.Rogue()
		}
		if cmdl.Hipnotic() || cmdl.Quoth() {
			return maps.Hipnotic()
		}
		return maps.Base()
	}()
	return &mapSelector{e, 0, 0}
}

func makeGameOptionsMenuItems() []MenuItem {
	selector := NewMapSelector()
	return []MenuItem{
		&beginGameMenuItem{qMenuItem{144, 40}, nil},
		&maxPlayersMenuItem{qMenuItem{144, 56}, 1},
		&gameTypeMenuItem{qMenuItem{144, 64}},
		&teamPlayMenuItem{qMenuItem{144, 72}},
		&skillMenuItem{qMenuItem{144, 80}},
		&fragLimitMenuItem{qMenuItem{144, 88}},
		&timeLimitMenuItem{qMenuItem{144, 96}},
		&episodeMenuItem{qMenuItem{144, 112}, selector},
		&levelMenuItem{qMenuItem{144, 120}, selector},
	}
}

type beginGameMenuItem struct {
	qMenuItem
	accepter qAccept
}
type maxPlayersMenuItem struct {
	qMenuItem
	maxPlayers int
}
type gameTypeMenuItem struct {
	qMenuItem
}
type teamPlayMenuItem struct {
	qMenuItem
}
type skillMenuItem struct {
	qMenuItem
}
type fragLimitMenuItem struct{ qMenuItem }
type timeLimitMenuItem struct{ qMenuItem }
type episodeMenuItem struct {
	qMenuItem
	selector *mapSelector
}
type levelMenuItem struct {
	qMenuItem
	selector *mapSelector
}

func (m *beginGameMenuItem) Update(a qAccept) {
	m.accepter = a
}

func (m *beginGameMenuItem) Draw() {
	drawTextbox(152, m.Y-8, 10, 1)
	drawString(160, m.Y, "begin game")
}

func (m *maxPlayersMenuItem) Update(a qAccept) {
	if m.maxPlayers == 0 {
		m.maxPlayers = svs.maxClients
	}
	if m.maxPlayers < 2 {
		m.maxPlayers = svs.maxClientsLimit
	}
}

func (m *maxPlayersMenuItem) Accept() {
	cbuf.AddText(fmt.Sprintf("maxplayers %d\n", m.maxPlayers))
}

func (m *maxPlayersMenuItem) Draw() {
	drawString(0, m.Y, "      Max players")
	drawString(160, m.Y, fmt.Sprintf("%d", m.maxPlayers))
}

func (m *gameTypeMenuItem) Draw() {
	drawString(0, m.Y, "        Game Type")
	t := func() string {
		if cvars.Coop.Value() == 0 {
			return "Deathmatch"
		}
		return "Cooperative"
	}()
	drawString(160, m.Y, t)
}

func (m *teamPlayMenuItem) Draw() {
	drawString(0, m.Y, "        Teamplay")
	drawString(160, m.Y, teamPlayMessage())
}

func teamPlayMessage() string {
	tp := int(cvars.TeamPlay.Value())
	if cmdl.Rogue() {
		switch tp {
		case 1:
			return "No Friendly Fire"
		case 2:
			return "Friendly Fire"
		case 3:
			return "Tag"
		case 4:
			return "Capture the Flag"
		case 5:
			return "One Flag CTF"
		case 6:
			return "Three Team CTF"
		default:
			return "Off"
		}
	} else {
		switch tp {
		case 1:
			return "No Friendly Fire"
		case 2:
			return "Friendly Fire"
		default:
			return "Off"
		}
	}
}

func (m *skillMenuItem) Draw() {
	drawString(0, m.Y, "            Skill")
	l := func() string {
		switch int(cvars.Skill.Value()) {
		case 0:
			return "Easy difficulty"
		case 1:
			return "Normal difficulty"
		case 2:
			return "Hard difficulty"
		default:
			return "Nightmare difficulty"
		}
	}()
	drawString(160, m.Y, l)
}

func (m *fragLimitMenuItem) Draw() {
	drawString(0, m.Y, "       Frag Limit")
	l := func() string {
		l := int(cvars.FragLimit.Value())
		if l == 0 {
			return "none"
		}
		return fmt.Sprintf("%d frags", l)
	}()
	drawString(160, m.Y, l)
}

func (m *timeLimitMenuItem) Draw() {
	drawString(0, m.Y, "       Time Limit")
	l := func() string {
		l := int(cvars.TimeLimit.Value())
		if l == 0 {
			return "none"
		}
		return fmt.Sprintf("%d minutes", l)
	}()
	drawString(160, m.Y, l)
}

func (m *episodeMenuItem) Draw() {
	drawString(0, m.Y, "         Episode")
	drawString(160, m.Y, m.selector.Episode().Name)
}

func (m *levelMenuItem) Draw() {
	drawString(0, m.Y, "           Level")
	drawString(160, m.Y, m.selector.Level().Name)
	drawString(160, m.Y+8, m.selector.Level().ID)
}

type qGameOptionsMenu struct {
	selectedIndex int
	items         []MenuItem
}

func (m *qGameOptionsMenu) Update() {
	for _, item := range m.items {
		item.Update(m)
	}
}

func (m *qGameOptionsMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterNetSetupMenu()
	case kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.LEFTARROW:
		m.items[m.selectedIndex].Left()
	case kc.RIGHTARROW:
		m.items[m.selectedIndex].Right()
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		m.items[m.selectedIndex].Enter()
	}
}

func (m *beginGameMenuItem) Enter() {
	m.accepter.Accept()
}

func (m *qGameOptionsMenu) Accept() {
	localSound("misc/menu2.wav")
	if sv.active {
		cbuf.AddText("disconnect\n")
	}
	cbuf.AddText("listen 0\n") // this seems to be a workaround to get the port set

	m.items[1].Accept() // maxPlayersMenuItem
	screen.BeginLoadingPlaque()
	m.items[7].Accept() // episodeMenuItem
}

func (m *episodeMenuItem) Accept() {
	m.selector.Accept()
}

func (m *qGameOptionsMenu) Draw() {
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))
	p := GetCachedPicture("gfx/p_multi.lmp")
	DrawPicture((320-p.Width)/2, 4, p)

	for _, item := range m.items {
		item.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
	/*
	   if (m_serverInfoMessage) {
	     if ((HostRealTime() - m_serverInfoMessageTime) < 5.0) {
	       x = (320 - 26 * 8) / 2;
	       M_DrawTextBox(x, 138, 24, 4);
	       x += 8;
	       M_Print(x, 146, "  More than 4 players   ");
	       M_Print(x, 154, " requires using command ");
	       M_Print(x, 162, "line parameters; please ");
	       M_Print(x, 170, "   see techinfo.txt.    ");
	     } else {
	       m_serverInfoMessage = false;
	     }
	   }

	*/
}

func (m *maxPlayersMenuItem) Left() {
	localSound("misc/menu3.wav")
	if m.maxPlayers <= 2 {
		m.maxPlayers = 2
		return
	}
	m.maxPlayers -= 1
}

func (m *maxPlayersMenuItem) Right() {
	localSound("misc/menu3.wav")
	if m.maxPlayers >= svs.maxClientsLimit {
		m.maxPlayers = svs.maxClientsLimit
		return
	}
	m.maxPlayers += 1
}

func (m *gameTypeMenuItem) Left() {
	localSound("misc/menu3.wav")
	cvars.Coop.Toggle()
}
func (m *gameTypeMenuItem) Right() {
	localSound("misc/menu3.wav")
	cvars.Coop.Toggle()
}

func maxTeamPlayItems() int {
	if cmdl.Rogue() {
		return 6
	}
	return 2
}

func (m *teamPlayMenuItem) Left() {
	localSound("misc/menu3.wav")
	max := maxTeamPlayItems()
	v := int(cvars.TeamPlay.Value())
	v = (v + max - 1) % max
	cvars.TeamPlay.SetValue(float32(v))
}

func (m *teamPlayMenuItem) Right() {
	localSound("misc/menu3.wav")
	max := maxTeamPlayItems()
	v := int(cvars.TeamPlay.Value())
	v = (v + 1) % max
	cvars.TeamPlay.SetValue(float32(v))
}

func (m *skillMenuItem) Left() {
	localSound("misc/menu3.wav")
	max := 3
	v := int(cvars.Skill.Value())
	v = (v + max - 1) % max
	cvars.Skill.SetValue(float32(v))
}
func (m *skillMenuItem) Right() {
	localSound("misc/menu3.wav")
	max := 3
	v := int(cvars.Skill.Value())
	v = (v + 1) % max
	cvars.Skill.SetValue(float32(v))
}
func (m *fragLimitMenuItem) Left() {
	localSound("misc/menu3.wav")
	max := 11
	v := int(cvars.FragLimit.Value()) / 10
	v = ((v + max - 1) % max) * 10
	cvars.FragLimit.SetValue(float32(v))
}
func (m *fragLimitMenuItem) Right() {
	localSound("misc/menu3.wav")
	max := 11
	v := int(cvars.FragLimit.Value()) / 10
	v = ((v + 1) % max) * 10
	cvars.FragLimit.SetValue(float32(v))
}
func (m *timeLimitMenuItem) Left() {
	// 0-60 in steps of 5
	localSound("misc/menu3.wav")
	max := 13
	v := int(cvars.TimeLimit.Value()) / 5
	v = ((v + max - 1) % max) * 5
	cvars.TimeLimit.SetValue(float32(v))
}
func (m *timeLimitMenuItem) Right() {
	localSound("misc/menu3.wav")
	max := 13
	v := int(cvars.TimeLimit.Value()) / 5
	v = ((v + 1) % max) * 5
	cvars.TimeLimit.SetValue(float32(v))
}

func (m *episodeMenuItem) Left() {
	localSound("misc/menu3.wav")
	m.selector.PreviousEpisode()
}

func (m *episodeMenuItem) Right() {
	localSound("misc/menu3.wav")
	m.selector.NextEpisode()
}

func (m *levelMenuItem) Left() {
	localSound("misc/menu3.wav")
	m.selector.PreviousLevel()
}

func (m *levelMenuItem) Right() {
	localSound("misc/menu3.wav")
	m.selector.NextLevel()
}
