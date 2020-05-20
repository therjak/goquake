package quakelib

import (
	"github.com/therjak/goquake/cbuf"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/input"
	kc "github.com/therjak/goquake/keycode"
	"github.com/therjak/goquake/math"
	"github.com/therjak/goquake/menu"
	"time"
)

type qOptionsMenu struct {
	selectedIndex int
	items         []MenuItem
}

func (m *qOptionsMenu) HandleKey(key kc.KeyCode) {
	switch key {
	case kc.ESCAPE, kc.BBUTTON:
		enterMenuMain()
	case kc.ENTER, kc.KP_ENTER, kc.ABUTTON:
		qmenu.playEnterSound = true
		m.items[m.selectedIndex].Enter()
	case kc.UPARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + len(m.items) - 1) % len(m.items)
	case kc.DOWNARROW:
		localSound("misc/menu1.wav")
		m.selectedIndex = (m.selectedIndex + 1) % len(m.items)
	case kc.LEFTARROW:
		localSound("misc/menu3.wav")
		m.items[m.selectedIndex].Left()
	case kc.RIGHTARROW:
		localSound("misc/menu3.wav")
		m.items[m.selectedIndex].Right()
	}
}

func (m *qOptionsMenu) Draw() {
	DrawPicture(16, 4, GetCachedPicture("gfx/qplaque.lmp"))

	p := GetCachedPicture("gfx/p_option.lmp")
	DrawPicture((320-p.Width)/2, 4, p)

	for _, i := range m.items {
		i.Draw()
	}
	m.items[m.selectedIndex].DrawCursor()
}

type MenuItemControls struct {
	qMenuItem
}

func (m *MenuItemControls) Draw() {
	drawString(16, m.Y, "              Controls")
}

func (m *MenuItemControls) Enter() {
	enterMenuKeys()
}

type MenuItemGotoConsole struct {
	qMenuItem
}

func (m *MenuItemGotoConsole) Draw() {
	drawString(16, m.Y, "          Goto console")
}

func (m *MenuItemGotoConsole) Enter() {
	qmenu.state = menu.None
	console.Toggle()
}

type MenuItemResetConfig struct {
	qMenuItem
}

func (m *MenuItemResetConfig) Draw() {
	drawString(16, m.Y, "          Reset config")
}

func (m *MenuItemResetConfig) Enter() {
	if screen.ModalMessage(
		"This will reset all controls\nand stored cvars. Continue? (y/n)\n",
		time.Second*15) {
		cbuf.AddText("resetcfg\n")
		cbuf.AddText("exec default.cfg\n")
	}
}

type MenuItemScale struct {
	qMenuItem
}

func (m *MenuItemScale) Draw() {
	drawString(16, m.Y, "                 Scale")
	l := (float32(screen.Width) / 320.0) - 1
	r := func() float32 {
		if l > 0 {
			return (cvars.ScreenConsoleScale.Value() - 1) / l
		}
		return 0
	}()
	drawSlider(220, m.Y, r)
}

func (m *MenuItemScale) Left() {
	m.change(cvars.ScreenConsoleScale.Value() - 0.1)
}
func (m *MenuItemScale) Right() {
	m.change(cvars.ScreenConsoleScale.Value() + 0.1)
}
func (m *MenuItemScale) change(f float32) {
	l := float32((screen.Width+31)/32) / 10.0
	v := math.Clamp32(1, f, l)
	cvars.ScreenConsoleScale.SetValue(v)
	cvars.ScreenMenuScale.SetValue(v)
	cvars.ScreenStatusbarScale.SetValue(v)
}

type MenuItemScreenSize struct {
	qMenuItem
}

func (m *MenuItemScreenSize) Draw() {
	drawString(16, m.Y, "           Screen size")
	r := (cvars.ViewSize.Value() - 30) / (120 - 30)
	drawSlider(220, m.Y, r)
}

func (m *MenuItemScreenSize) Left() {
	m.change(cvars.ViewSize.Value() - 10)
}
func (m *MenuItemScreenSize) Right() {
	m.change(cvars.ViewSize.Value() + 10)
}
func (m *MenuItemScreenSize) change(f float32) {
	v := math.Clamp32(30, f, 120)
	cvars.ViewSize.SetValue(v)
	// TODO: there is some bug in interaction between ViewSize and ScreenSize
}

type MenuItemBrightness struct {
	qMenuItem
}

func (m *MenuItemBrightness) Draw() {
	drawString(16, m.Y, "            Brightness")
	r := (1.0 - cvars.Gamma.Value()) / 0.5
	drawSlider(220, m.Y, r)
}

func (m *MenuItemBrightness) Left() {
	m.change(cvars.Gamma.Value() + 0.05)
}
func (m *MenuItemBrightness) Right() {
	m.change(cvars.Gamma.Value() - 0.05)
}
func (m *MenuItemBrightness) change(f float32) {
	v := math.Clamp32(0.5, f, 1)
	cvars.Gamma.SetValue(v)
}

type MenuItemContrast struct {
	qMenuItem
}

func (m *MenuItemContrast) Draw() {
	drawString(16, m.Y, "              Contrast")
	r := cvars.Contrast.Value() - 1.0
	drawSlider(220, m.Y, r)
}

func (m *MenuItemContrast) Left() {
	m.change(cvars.Contrast.Value() - 0.1)
}
func (m *MenuItemContrast) Right() {
	m.change(cvars.Contrast.Value() + 0.1)
}
func (m *MenuItemContrast) change(f float32) {
	v := math.Clamp32(1, f, 2)
	cvars.Contrast.SetValue(v)
}

type MenuItemMouseSpeed struct {
	qMenuItem
}

func (m *MenuItemMouseSpeed) Draw() {
	drawString(16, m.Y, "           Mouse Speed")
	r := (cvars.Sensitivity.Value() - 1) / 10
	drawSlider(220, m.Y, r)
}

func (m *MenuItemMouseSpeed) Left() {
	m.change(cvars.Sensitivity.Value() - 0.5)
}
func (m *MenuItemMouseSpeed) Right() {
	m.change(cvars.Sensitivity.Value() + 0.5)
}
func (m *MenuItemMouseSpeed) change(f float32) {
	v := math.Clamp32(1, f, 11)
	cvars.Sensitivity.SetValue(v)
}

type MenuItemStatusbarAlpha struct {
	qMenuItem
}

func (m *MenuItemStatusbarAlpha) Draw() {
	drawString(16, m.Y, "       Statusbar alpha")
	r := (1.0 - cvars.ScreenStatusbarAlpha.Value())
	// scr_sbaralpha range is 1.0 to 0.0
	drawSlider(220, m.Y, r)
}

func (m *MenuItemStatusbarAlpha) Left() {
	m.change(cvars.ScreenStatusbarAlpha.Value() + 0.05)
}
func (m *MenuItemStatusbarAlpha) Right() {
	m.change(cvars.ScreenStatusbarAlpha.Value() - 0.05)
}
func (m *MenuItemStatusbarAlpha) change(f float32) {
	v := math.Clamp32(0, f, 1)
	cvars.ScreenStatusbarAlpha.SetValue(v)
}

type MenuItemSoundVolume struct {
	qMenuItem
}

func (m *MenuItemSoundVolume) Draw() {
	drawString(16, m.Y, "          Sound Volume")
	r := cvars.Volume.Value()
	drawSlider(220, m.Y, r)
}

func (m *MenuItemSoundVolume) Left() {
	m.change(cvars.Volume.Value() - 0.1)
}
func (m *MenuItemSoundVolume) Right() {
	m.change(cvars.Volume.Value() + 0.1)
}
func (m *MenuItemSoundVolume) change(f float32) {
	v := math.Clamp32(0, f, 1)
	cvars.Volume.SetValue(v)
}

type MenuItemAlwaysRun struct {
	qMenuItem
}

func (m *MenuItemAlwaysRun) Draw() {
	drawString(16, m.Y, "            Always Run")
	drawCheckbox(220, m.Y, cvars.ClientForwardSpeed.Value() > 200)
}

func (m *MenuItemAlwaysRun) Left() {
	m.Enter()
}
func (m *MenuItemAlwaysRun) Right() {
	m.Enter()
}
func (m *MenuItemAlwaysRun) Enter() {
	if cvars.ClientMoveSpeedKey.Value() <= 1 {
		cvars.ClientMoveSpeedKey.SetValue(2)
	}
	if cvars.ClientForwardSpeed.Value() > 200 {
		cvars.ClientForwardSpeed.SetValue(200)
		cvars.ClientBackSpeed.SetValue(200)
	} else {
		v := 200 * cvars.ClientMoveSpeedKey.Value()
		cvars.ClientForwardSpeed.SetValue(v)
		cvars.ClientBackSpeed.SetValue(v)
	}
}

type MenuItemInvertMouse struct {
	qMenuItem
}

func (m *MenuItemInvertMouse) Draw() {
	drawString(16, m.Y, "          Invert Mouse")
	drawCheckbox(220, m.Y, cvars.MousePitch.Value() < 0)
}

func (m *MenuItemInvertMouse) Left() {
	m.Enter()
}
func (m *MenuItemInvertMouse) Right() {
	m.Enter()
}
func (m *MenuItemInvertMouse) Enter() {
	cvars.MousePitch.SetValue(-cvars.MousePitch.Value())
}

type MenuItemMouseLook struct {
	qMenuItem
}

func (m *MenuItemMouseLook) Draw() {
	drawString(16, m.Y, "            Mouse Look")
	drawCheckbox(220, m.Y, input.MLook.Down())
}

func (m *MenuItemMouseLook) Left() {
	m.Enter()
}
func (m *MenuItemMouseLook) Right() {
	m.Enter()
}
func (m *MenuItemMouseLook) Enter() {
	if input.MLook.Down() {
		cbuf.AddText("-mlook\n")
	} else {
		cbuf.AddText("+mlook\n")
	}
}

type MenuItemLookspring struct {
	qMenuItem
}

func (m *MenuItemLookspring) Draw() {
	drawString(16, m.Y, "            Lookspring")
	drawCheckbox(220, m.Y, cvars.LookSpring.Value() != 0)
}

func (m *MenuItemLookspring) Left() {
	m.Enter()
}
func (m *MenuItemLookspring) Right() {
	m.Enter()
}
func (m *MenuItemLookspring) Enter() {
	if cvars.LookSpring.Value() != 0 {
		cvars.LookSpring.SetValue(0)
	} else {
		cvars.LookSpring.SetValue(1)
	}
}

type MenuItemLookstrafe struct {
	qMenuItem
}

func (m *MenuItemLookstrafe) Draw() {
	drawString(16, m.Y, "            Lookstrafe")
	drawCheckbox(220, m.Y, cvars.LookStrafe.Value() != 0)
}

func (m *MenuItemLookstrafe) Left() {
	m.Enter()
}
func (m *MenuItemLookstrafe) Right() {
	m.Enter()
}
func (m *MenuItemLookstrafe) Enter() {
	if cvars.LookStrafe.Value() != 0 {
		cvars.LookStrafe.SetValue(0)
	} else {
		cvars.LookStrafe.SetValue(1)
	}
}

type MenuItemVideoOptions struct {
	qMenuItem
}

func (m *MenuItemVideoOptions) Draw() {
	drawString(16, m.Y, "         Video Options")
}

func (m *MenuItemVideoOptions) Enter() {
	enterMenuVideo()
}

var (
	optionsMenu = qOptionsMenu{
		items: makeOptionsMenuItems(),
	}
)

func makeOptionsMenuItems() []MenuItem {
	return []MenuItem{
		&MenuItemControls{qMenuItem{200, 32}},
		&MenuItemGotoConsole{qMenuItem{200, 32 + 8}},
		&MenuItemResetConfig{qMenuItem{200, 32 + 8*2}},
		&MenuItemScale{qMenuItem{200, 32 + 8*3}},
		&MenuItemScreenSize{qMenuItem{200, 32 + 8*4}},
		&MenuItemBrightness{qMenuItem{200, 32 + 8*5}},
		&MenuItemContrast{qMenuItem{200, 32 + 8*6}},
		&MenuItemMouseSpeed{qMenuItem{200, 32 + 8*7}},
		&MenuItemStatusbarAlpha{qMenuItem{200, 32 + 8*8}},
		&MenuItemSoundVolume{qMenuItem{200, 32 + 8*9}},
		&MenuItemAlwaysRun{qMenuItem{200, 32 + 8*10}},
		&MenuItemInvertMouse{qMenuItem{200, 32 + 8*11}},
		&MenuItemMouseLook{qMenuItem{200, 32 + 8*12}},
		&MenuItemLookspring{qMenuItem{200, 32 + 8*13}},
		&MenuItemLookstrafe{qMenuItem{200, 32 + 8*14}},
		&MenuItemVideoOptions{qMenuItem{200, 32 + 8*15}},
	}
}
