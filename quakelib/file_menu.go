package quakelib

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"quake/cbuf"
	kc "quake/keycode"
	"quake/keys"
	"quake/menu"
	"strings"
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
	SCR_BeginLoadingPlaque()

	// This should be direct instead of cbuf style
	cbuf.AddText(fmt.Sprintf("load %s\n", m.filename))
}

func (m *qFileMenu) update() {
	for _, i := range m.items {
		i.loadable = false
		n := filepath.Join(gameDirectory, i.filename)
		f, err := os.Open(n)
		if err != nil {
			i.comment = unusedSaveName
			continue
		}
		defer f.Close()
		var version int32
		_, err = fmt.Fscanf(f, "%d\n", &version) // why read?
		if err != nil {
			log.Printf("could not read version %s", err)
			continue
		}
		var comment string
		// why read 79? afterwards we only want 39
		_, err = fmt.Fscanf(f, "%79s\n", &comment)
		if err != nil {
			log.Printf("could not read comment %s", err)
			continue
		}
		i.comment = strings.Replace(comment[:39], "_", " ", -1)
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
	p := getCachePic("gfx/p_load.lmp")
	drawPic((320-p.width)/2, 4, p)
	for _, item := range m.items {
		item.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}
func (m *qSaveMenu) Draw() {
	p := getCachePic("gfx/p_save.lmp")
	drawPic((320-p.width)/2, 4, p)
	for _, i := range m.items {
		i.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}
func (m *qLoadMenu) HandleKey(k int) {
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

func (m *qSaveMenu) HandleKey(k int) {
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
