package quakelib

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/therjak/goquake/cbuf"
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/menu"
	"github.com/therjak/goquake/protos"

	"google.golang.org/protobuf/proto"
)

func enterLoadMenu() {
	qmenu.playEnterSound = true
	qmenu.state = menu.Load
	IN_Deactivate()
	keyDestination = keys.Menu
	loadMenu.update()
}
func enterSaveMenu() {
	if !sv.active || (cl.intermission != 0) || (svs.maxClients != 1) {
		return
	}

	qmenu.playEnterSound = true
	qmenu.state = menu.Save
	IN_Deactivate()
	keyDestination = keys.Menu
	saveMenu.update()
}

const (
	unusedSaveName = "--- UNUSED SLOT ---"
)

var (
	loadMenu = qLoadMenu{makeFileMenu()}
	saveMenu = qSaveMenu{makeFileMenu()}
)

func makeFileMenu() qFileMenu {
	return qFileMenu{0, makeFileMenuItems()}
}
func makeFileMenuItems() [20]*fileMenuItem {
	var items [20]*fileMenuItem
	for i := 0; i < len(items); i++ {
		f := fmt.Sprintf("s%d.sav", i)
		items[i] = &fileMenuItem{qMenuItem{8, 32 + 8*i}, unusedSaveName, f, false}
	}
	return items
}

type qFileMenu struct {
	selectedIndex int
	items         [20]*fileMenuItem
}

type fileMenuItem struct {
	qMenuItem
	comment  string // max 39 chars
	filename string
	loadable bool
}

func (m *fileMenuItem) Draw() {
	drawString(16, m.Y, m.comment)
}

func (m *fileMenuItem) Save() {
	qmenu.state = menu.None
	IN_Activate()
	keyDestination = keys.Game
	// TODO: use a direct save m.filename not cbuf style
	cbuf.AddText(fmt.Sprintf("save %s\n", m.filename))
}
func (m *fileMenuItem) Load() {

	if !m.loadable {
		return
	}
	qmenu.state = menu.None
	IN_Activate()
	keyDestination = keys.Game

	// Host_Loadgame_f can't bring up the loading plaque because too much
	// stack space has been used, so do it now
	screen.BeginLoadingPlaque()

	// This should be direct instead of cbuf style
	cbuf.AddText(fmt.Sprintf("load %s\n", m.filename))
}

func (m *qFileMenu) update() {
	sg := &protos.SaveGame{}
	for _, i := range m.items {
		i.loadable = false
		i.comment = unusedSaveName
		n := filepath.Join(GameDirectory(), i.filename)

		in, err := ioutil.ReadFile(n)
		if err != nil {
			continue
		}
		if err := proto.Unmarshal(in, sg); err != nil {
			log.Printf("Failed to parse savegame: %v", err)
			continue
		}

		i.comment = sg.Comment
		if len(i.comment) > 39 {
			i.comment = i.comment[:39] // orig says 39 but includes \0
		}
		i.loadable = true
	}
}

type qLoadMenu struct {
	qFileMenu
}
type qSaveMenu struct {
	qFileMenu
}

func (m *qLoadMenu) Draw() {
	p := GetCachedPicture("gfx/p_load.lmp")
	DrawPicture((320-p.Width)/2, 4, p)
	for _, item := range m.items {
		item.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}
func (m *qSaveMenu) Draw() {
	p := GetCachedPicture("gfx/p_save.lmp")
	DrawPicture((320-p.Width)/2, 4, p)
	for _, i := range m.items {
		i.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}
func (m *qLoadMenu) HandleKey(k kc.KeyCode) {
	switch k {
	case kc.ESCAPE, kc.BBUTTON:
		enterSinglePlayerMenu()
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		localSound("misc/menu2.wav")
		m.items[m.selectedIndex].Load()
	case kc.DOWNARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	}
}

func (m *qSaveMenu) HandleKey(k kc.KeyCode) {
	switch k {
	case kc.ESCAPE, kc.BBUTTON:
		enterSinglePlayerMenu()
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		m.items[m.selectedIndex].Save()
	case kc.DOWNARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	}
}
