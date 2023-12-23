// SPDX-License-Identifier: GPL-2.0-or-later

package cvars

import (
	"strings"

	"goquake/cvar"
	"goquake/math/vec"

	"github.com/chewxy/math32"
)

var (
	ClientColor = cvar.New("_cl_color", "0", cvar.ARCHIVE)
	ClientName  = cvar.New("_cl_name", "player", cvar.ARCHIVE)

	AmbientFade           = cvar.New("ambient_fade", "100", cvar.NONE)
	AmbientLevel          = cvar.New("ambient_level", "0.3", cvar.NONE)
	BackgroundVolume      = cvar.New("bgmvolume", "1", cvar.ARCHIVE) // cd music volume, therjak: this is dead, only used in menu
	Campaign              = cvar.New("campaign", "0", cvar.NONE)     // 2021 release
	CfgUnbindAll          = cvar.New("cfg_unbindall", "1", cvar.ARCHIVE)
	ChaseActive           = cvar.New("chase_active", "0", cvar.NONE)
	ChaseBack             = cvar.New("chase_back", "100", cvar.NONE)
	ChaseRight            = cvar.New("chase_right", "0", cvar.NONE)
	ChaseUp               = cvar.New("chase_up", "16", cvar.NONE)
	ClientAngleSpeedKey   = cvar.New("cl_anglespeedkey", "1.5", cvar.NONE)
	ClientBackSpeed       = cvar.New("cl_backspeed", "200", cvar.ARCHIVE)
	ClientBob             = cvar.New("cl_bob", "0.02", cvar.NONE)
	ClientBobCycle        = cvar.New("cl_bobcycle", "0.6", cvar.NONE)
	ClientBobUp           = cvar.New("cl_bobup", "0.5", cvar.NONE)
	ClientForwardSpeed    = cvar.New("cl_forwardspeed", "200", cvar.ARCHIVE)
	ClientMaxPitch        = cvar.New("cl_maxpitch", "90", cvar.ARCHIVE)
	ClientMinPitch        = cvar.New("cl_minpitch", "-90", cvar.ARCHIVE)
	ClientMoveSpeedKey    = cvar.New("cl_movespeedkey", "2.0", cvar.NONE)
	ClientNoLerp          = cvar.New("cl_nolerp", "0", cvar.NONE)
	ClientPitchSpeed      = cvar.New("cl_pitchspeed", "150", cvar.NONE)
	ClientRollAngle       = cvar.New("cl_rollangle", "2.0", cvar.NONE)
	ClientRollSpeed       = cvar.New("cl_rollspeed", "200", cvar.NONE)
	ClientShowNet         = cvar.New("cl_shownet", "0", cvar.NONE)
	ClientSideSpeed       = cvar.New("cl_sidespeed", "350", cvar.NONE)
	ClientUpSpeed         = cvar.New("cl_upspeed", "200", cvar.NONE)
	ClientYawSpeed        = cvar.New("cl_yawspeed", "140", cvar.NONE)
	ConsoleLogCenterPrint = cvar.New("con_logcenterprint", "1", cvar.NONE)
	ConsoleNotifyTime     = cvar.New("con_notifytime", "3", cvar.NONE)
	Contrast              = cvar.New("contrast", "1", cvar.ARCHIVE)
	Coop                  = cvar.New("coop", "0", cvar.NONE)
	Crosshair             = cvar.New("crosshair", "0", cvar.ARCHIVE)
	DeathMatch            = cvar.New("deathmatch", "0", cvar.NONE)
	DevStats              = cvar.New("devstats", "0", cvar.NONE)
	Developer             = cvar.New("developer", "0", cvar.NONE)
	ExternalEnts          = cvar.New("external_ents", "1", cvar.ARCHIVE)
	Fov                   = cvar.New("fov", "90", cvar.NONE)
	FovAdapt              = cvar.New("fov_adapt", "1", cvar.ARCHIVE)
	FragLimit             = cvar.New("fraglimit", "0", cvar.NOTIFY|cvar.SERVERINFO)
	GameCfg               = cvar.New("gamecfg", "0", cvar.NONE)
	Gamma                 = cvar.New("gamma", "1", cvar.ARCHIVE)
	GlAffineModels        = cvar.New("gl_affinemodels", "0", cvar.NONE)
	GlColorShiftPercent   = cvar.New("gl_cshiftpercent", "100", cvar.NONE)
	GlClear               = cvar.New("gl_clear", "1", cvar.NONE)
	GlCull                = cvar.New("gl_cull", "1", cvar.NONE)
	GlFarClip             = cvar.New("gl_farclip", "16384", cvar.ARCHIVE)
	GlFinish              = cvar.New("gl_finish", "0", cvar.NONE)
	GlFlashBlend          = cvar.New("gl_flashblend", "0", cvar.ARCHIVE)
	GlFullBrights         = cvar.New("gl_fullbrights", "1", cvar.ARCHIVE)
	GlMaxSize             = cvar.New("gl_max_size", "0", cvar.NONE)
	GlNoColors            = cvar.New("gl_nocolors", "0", cvar.NONE)
	GlOverBright          = cvar.New("gl_overbright", "1", cvar.ARCHIVE)
	GlOverBrightModels    = cvar.New("gl_overbright_models", "1", cvar.ARCHIVE)
	GlPicMip              = cvar.New("gl_picmip", "0", cvar.NONE)
	GlPlayerMip           = cvar.New("gl_playermip", "0", cvar.NONE)
	GlPolyBlend           = cvar.New("gl_polyblend", "1", cvar.NONE)
	GlSmoothModels        = cvar.New("gl_smoothmodels", "1", cvar.NONE)
	GlSubdivideSize       = cvar.New("gl_subdivide_size", "128", cvar.ARCHIVE)
	// correct value is filled in later.
	GlTextureMode          = cvar.New("gl_texturemode", "", cvar.ARCHIVE)
	GlTextureAnisotropy    = cvar.New("gl_texture_anisotropy", "1", cvar.ARCHIVE)
	GlTripleBuffer         = cvar.New("gl_triplebuffer", "1", cvar.ARCHIVE)
	GlZFix                 = cvar.New("gl_zfix", "0", cvar.NONE)
	HostFrameRate          = cvar.New("host_framerate", "0", cvar.NONE)
	HostMaxFps             = cvar.New("host_maxfps", "72", cvar.ARCHIVE)
	HostName               = cvar.New("hostname", "UNNAMED", cvar.NONE)
	HostSpeeds             = cvar.New("host_speeds", "0", cvar.NONE)
	HostTimeScale          = cvar.New("host_timescale", "0", cvar.NONE)
	InputDebugKeys         = cvar.New("in_debugkeys", "0", cvar.NONE)
	LoadAs8Bit             = cvar.New("loadas8bit", "0", cvar.NONE)
	LookSpring             = cvar.New("lookspring", "0", cvar.ARCHIVE)
	LookStrafe             = cvar.New("lookstrafe", "0", cvar.ARCHIVE)
	MaxEdicts              = cvar.New("max_edicts", "15000", cvar.NONE)
	MouseForward           = cvar.New("m_forward", "1", cvar.ARCHIVE)
	MousePitch             = cvar.New("m_pitch", "0.022", cvar.ARCHIVE)
	MouseSide              = cvar.New("m_side", "0.8", cvar.ARCHIVE)
	MouseYaw               = cvar.New("m_yaw", "0.022", cvar.ARCHIVE)
	NetMessageTimeout      = cvar.New("net_messagetimeout", "300", cvar.NONE)
	NoExit                 = cvar.New("noexit", "0", cvar.NOTIFY|cvar.SERVERINFO)
	NoMonsters             = cvar.New("nomonsters", "0", cvar.NONE)
	NoSound                = cvar.New("nosound", "0", cvar.NONE)
	Pausable               = cvar.New("pausable", "1", cvar.NONE)
	Precache               = cvar.New("precache", "1", cvar.NONE)
	RClearColor            = cvar.New("r_clearcolor", "2", cvar.ARCHIVE)
	RDrawEntities          = cvar.New("r_drawentities", "1", cvar.NONE)
	RDrawFlat              = cvar.New("r_drawflat", "0", cvar.NONE)
	RDrawViewModel         = cvar.New("r_drawviewmodel", "1", cvar.NONE)
	RDrawWorld             = cvar.New("r_drawworld", "1", cvar.NONE)
	RDynamic               = cvar.New("r_dynamic", "1", cvar.ARCHIVE)
	RFastSky               = cvar.New("r_fastsky", "0", cvar.NONE)
	RFlatLightStyles       = cvar.New("r_flatlightstyles", "0", cvar.NONE)
	RFullBright            = cvar.New("r_fullbright", "0", cvar.NONE)
	RLavaAlpha             = cvar.New("r_lavaalpha", "0", cvar.NONE)
	RLerpModels            = cvar.New("r_lerpmodels", "1", cvar.NONE)
	RLerpMove              = cvar.New("r_lerpmove", "1", cvar.NONE)
	RLightMap              = cvar.New("r_lightmap", "0", cvar.NONE)
	RNoRefresh             = cvar.New("r_norefresh", "0", cvar.NONE)
	RNoVis                 = cvar.New("r_novis", "0", cvar.ARCHIVE)
	ROldSkyLeaf            = cvar.New("r_oldskyleaf", "0", cvar.NONE)
	ROldWater              = cvar.New("r_oldwater", "0", cvar.ARCHIVE)
	RParticles             = cvar.New("r_particles", "1", cvar.ARCHIVE)
	RPos                   = cvar.New("r_pos", "0", cvar.NONE)
	RQuadParticles         = cvar.New("r_quadparticles", "1", cvar.ARCHIVE)
	RShadows               = cvar.New("r_shadows", "0", cvar.ARCHIVE)
	RShowBoxes             = cvar.New("r_showbboxes", "0", cvar.NONE)
	RShowTris              = cvar.New("r_showtris", "0", cvar.NONE)
	RSkyAlpha              = cvar.New("r_skyalpha", "1", cvar.NONE)
	RSkyFog                = cvar.New("r_skyfog", "0.5", cvar.NONE)
	RSkyQuality            = cvar.New("r_sky_quality", "12", cvar.NONE)
	RSlimeAlpha            = cvar.New("r_slimealpha", "0", cvar.NONE)
	RSpeeds                = cvar.New("r_speeds", "0", cvar.NONE)
	RTeleAlpha             = cvar.New("r_telealpha", "0", cvar.NONE)
	RWaterAlpha            = cvar.New("r_wateralpha", "1", cvar.ARCHIVE)
	RWaterQuality          = cvar.New("r_waterquality", "8", cvar.NONE)
	RWaterWarp             = cvar.New("r_waterwarp", "1", cvar.NONE)
	SameLevel              = cvar.New("samelevel", "0", cvar.NONE)
	Saved1                 = cvar.New("saved1", "0", cvar.ARCHIVE)
	Saved2                 = cvar.New("saved2", "0", cvar.ARCHIVE)
	Saved3                 = cvar.New("saved3", "0", cvar.ARCHIVE)
	Saved4                 = cvar.New("saved4", "0", cvar.ARCHIVE)
	SavedGameCfg           = cvar.New("savedgamecfg", "0", cvar.ARCHIVE)
	Scratch1               = cvar.New("scratch1", "0", cvar.NONE)
	Scratch2               = cvar.New("scratch2", "0", cvar.NONE)
	Scratch3               = cvar.New("scratch3", "0", cvar.NONE)
	Scratch4               = cvar.New("scratch4", "0", cvar.NONE)
	ScreenCenterTime       = cvar.New("scr_centertime", "2", cvar.NONE)
	ScreenClock            = cvar.New("scr_clock", "0", cvar.NONE)
	ScreenConsoleAlpha     = cvar.New("scr_conalpha", "0.5", cvar.ARCHIVE)
	ScreenConsoleSpeed     = cvar.New("scr_conspeed", "500", cvar.ARCHIVE)
	ScreenConsoleScale     = cvar.New("scr_conscale", "1", cvar.ARCHIVE)
	ScreenConsoleWidth     = cvar.New("scr_conwidth", "0", cvar.ARCHIVE)
	ScreenCrosshairScale   = cvar.New("scr_crosshairscale", "1", cvar.ARCHIVE)
	ScreenMenuScale        = cvar.New("scr_menuscale", "1", cvar.ARCHIVE)
	ScreenOffsetX          = cvar.New("scr_ofsx", "0", cvar.NONE)
	ScreenOffsetY          = cvar.New("scr_ofsy", "0", cvar.NONE)
	ScreenOffsetZ          = cvar.New("scr_ofsz", "0", cvar.NONE)
	ScreenPrintSpeed       = cvar.New("scr_printspeed", "8", cvar.NONE)
	ScreenStatusbarAlpha   = cvar.New("scr_sbaralpha", "0.75", cvar.ARCHIVE)
	ScreenStatusbarScale   = cvar.New("scr_sbarscale", "1", cvar.ARCHIVE)
	ScreenShowFps          = cvar.New("scr_showfps", "0", cvar.NONE)
	Sensitivity            = cvar.New("sensitivity", "3", cvar.ARCHIVE)
	ServerAccelerate       = cvar.New("sv_accelerate", "10", cvar.NONE)
	ServerAim              = cvar.New("sv_aim", "1", cvar.NONE)
	ServerAltNoClip        = cvar.New("sv_altnoclip", "1", cvar.ARCHIVE)
	ServerEdgeFriction     = cvar.New("edgefriction", "2", cvar.NONE)
	ServerFreezeNonClients = cvar.New("sv_freezenonclients", "0", cvar.NONE)
	ServerFriction         = cvar.New("sv_friction", "4", cvar.NOTIFY|cvar.SERVERINFO)
	ServerGravity          = cvar.New("sv_gravity", "800", cvar.NOTIFY|cvar.SERVERINFO)
	ServerIdealPitchScale  = cvar.New("sv_idealpitchscale", "0.8", cvar.NONE)
	ServerMaxSpeed         = cvar.New("sv_maxspeed", "320", cvar.NOTIFY|cvar.SERVERINFO)
	ServerMaxVelocity      = cvar.New("sv_maxvelocity", "2000", cvar.NONE)
	ServerNoStep           = cvar.New("sv_nostep", "0", cvar.NONE)
	ServerProfile          = cvar.New("serverprofile", "0", cvar.NONE)
	ServerStopSpeed        = cvar.New("sv_stopspeed", "100", cvar.NONE)
	ShowPause              = cvar.New("showpause", "1", cvar.NONE)
	ShowRAM                = cvar.New("showram", "1", cvar.NONE)
	ShowTurtle             = cvar.New("showturtle", "0", cvar.NONE)
	Skill                  = cvar.New("skill", "1", cvar.NONE)
	SoundFilterQuality     = cvar.New("snd_filterquality", "1", cvar.NONE) // 5 on win, 1 on all other
	SoundMixAhead          = cvar.New("snd_mixahead", "0.1", cvar.ARCHIVE)
	SoundMixSpeed          = cvar.New("snd_mixspeed", "44100", cvar.NONE)
	SoundNoExtraUpdate     = cvar.New("snd_noextraupdate", "0", cvar.NONE)
	SoundShow              = cvar.New("snd_show", "0", cvar.NONE)
	SoundSpeed             = cvar.New("sndspeed", "11025", cvar.NONE)
	TeamPlay               = cvar.New("teamplay", "0", cvar.NOTIFY|cvar.SERVERINFO)
	Temp1                  = cvar.New("temp1", "0", cvar.NONE)
	Throttle               = cvar.New("sys_throttle", "0.02", cvar.ARCHIVE)
	TicRate                = cvar.New("sys_ticrate", "0.05", cvar.NONE)
	TimeLimit              = cvar.New("timelimit", "0", cvar.NOTIFY|cvar.SERVERINFO)
	VideoBorderLess        = cvar.New("vid_borderless", "0", cvar.ARCHIVE)
	VideoDesktopFullscreen = cvar.New("vid_desktopfullscreen", "0", cvar.ARCHIVE)
	VideoFsaa              = cvar.New("vid_fsaa", "0", cvar.ARCHIVE)
	VideoFullscreen        = cvar.New("vid_fullscreen", "0", cvar.ARCHIVE)
	VideoHeight            = cvar.New("vid_height", "600", cvar.ARCHIVE)
	VideoVerticalSync      = cvar.New("vid_vsync", "0", cvar.ARCHIVE)
	VideoWidth             = cvar.New("vid_width", "800", cvar.ARCHIVE)
	ViewCenterMove         = cvar.New("v_centermove", "0.15", cvar.NONE)
	ViewCenterSpeed        = cvar.New("v_centerspeed", "500", cvar.NONE)
	ViewGunKick            = cvar.New("v_gunkick", "1", cvar.NONE)
	ViewIPitchCycle        = cvar.New("v_ipitch_cycle", "1", cvar.NONE)
	ViewIPitchLevel        = cvar.New("v_ipitch_level", "0.3", cvar.NONE)
	ViewIRollCycle         = cvar.New("v_iroll_cycle", "0.5", cvar.NONE)
	ViewIRollLevel         = cvar.New("v_iroll_level", "0.1", cvar.NONE)
	ViewIYawCycle          = cvar.New("v_iyaw_cycle", "2", cvar.NONE)
	ViewIYawLevel          = cvar.New("v_iyaw_level", "0.3", cvar.NONE)
	ViewIdleScale          = cvar.New("v_idlescale", "0", cvar.NONE)
	ViewKickPitch          = cvar.New("v_kickpitch", "0.6", cvar.NONE)
	ViewKickRoll           = cvar.New("v_kickroll", "0.6", cvar.NONE)
	ViewKickTime           = cvar.New("v_kicktime", "0.5", cvar.NONE)
	ViewSize               = cvar.New("viewsize", "100", cvar.ARCHIVE)
	Volume                 = cvar.New("volume", "0.7", cvar.ARCHIVE)

	// this cvar gets read from within the vm
	registered = cvar.New("registered", "1", cvar.ROM)

	RNoLerpList = cvar.New("r_nolerp_list", strings.Join([]string{
		"progs/flame.mdl",
		"progs/flame2.mdl",
		"progs/braztall.mdl",
		"progs/brazshrt.mdl",
		"progs/longtrch.mdl",
		"progs/flame_pyre.mdl",
		"progs/v_saw.mdl",
		"progs/v_xfist.mdl",
		"progs/h2stuff/newfire.mdl",
	}, ","), cvar.NONE)
	RNoShadowList = cvar.New("r_noshadow_list", strings.Join([]string{
		"progs/flame2.mdl",
		"progs/flame.mdl",
		"progs/bolt1.mdl",
		"progs/bolt2.mdl",
		"progs/bolt3.mdl",
		"progs/laser.mdl",
	}, ","), cvar.NONE)

	RFullBrightList = cvar.New("r_fullbright_list", strings.Join([]string{
		"progs/flame2.mdl",
		"progs/flame.mdl",
		"progs/boss.mdl",
	}, ","), cvar.NONE)
)

func Register(c *cvar.Cvars) error {
	if err := c.Add(AmbientFade); err != nil {
		return err
	}

	if err := c.Add(AmbientLevel); err != nil {
		return err
	}

	if err := c.Add(BackgroundVolume); err != nil {
		return err
	}

	if err := c.Add(Campaign); err != nil {
		return err
	}

	if err := c.Add(CfgUnbindAll); err != nil {
		return err
	}

	if err := c.Add(ChaseActive); err != nil {
		return err
	}

	if err := c.Add(ChaseBack); err != nil {
		return err
	}

	if err := c.Add(ChaseRight); err != nil {
		return err
	}

	if err := c.Add(ChaseUp); err != nil {
		return err
	}

	if err := c.Add(ClientAngleSpeedKey); err != nil {
		return err
	}

	if err := c.Add(ClientBackSpeed); err != nil {
		return err
	}

	if err := c.Add(ClientBob); err != nil {
		return err
	}

	if err := c.Add(ClientBobCycle); err != nil {
		return err
	}

	if err := c.Add(ClientBobUp); err != nil {
		return err
	}

	if err := c.Add(ClientColor); err != nil {
		return err
	}

	if err := c.Add(ClientForwardSpeed); err != nil {
		return err
	}

	if err := c.Add(ClientMaxPitch); err != nil {
		return err
	}

	if err := c.Add(ClientMinPitch); err != nil {
		return err
	}

	if err := c.Add(ClientMoveSpeedKey); err != nil {
		return err
	}

	if err := c.Add(ClientName); err != nil {
		return err
	}

	if err := c.Add(ClientNoLerp); err != nil {
		return err
	}

	if err := c.Add(ClientPitchSpeed); err != nil {
		return err
	}

	if err := c.Add(ClientRollAngle); err != nil {
		return err
	}

	if err := c.Add(ClientRollSpeed); err != nil {
		return err
	}

	if err := c.Add(ClientShowNet); err != nil {
		return err
	}

	if err := c.Add(ClientSideSpeed); err != nil {
		return err
	}

	if err := c.Add(ClientUpSpeed); err != nil {
		return err
	}

	if err := c.Add(ClientYawSpeed); err != nil {
		return err
	}

	if err := c.Add(ConsoleLogCenterPrint); err != nil {
		return err
	}

	if err := c.Add(ConsoleNotifyTime); err != nil {
		return err
	}

	if err := c.Add(Contrast); err != nil {
		return err
	}

	if err := c.Add(Coop); err != nil {
		return err
	}

	if err := c.Add(Crosshair); err != nil {
		return err
	}

	if err := c.Add(DeathMatch); err != nil {
		return err
	}

	if err := c.Add(DevStats); err != nil {
		return err
	}

	if err := c.Add(Developer); err != nil {
		return err
	}

	if err := c.Add(ExternalEnts); err != nil {
		return err
	}

	if err := c.Add(Fov); err != nil {
		return err
	}

	if err := c.Add(FovAdapt); err != nil {
		return err
	}

	if err := c.Add(FragLimit); err != nil {
		return err
	}

	if err := c.Add(GameCfg); err != nil {
		return err
	}

	if err := c.Add(Gamma); err != nil {
		return err
	}

	if err := c.Add(GlAffineModels); err != nil {
		return err
	}

	if err := c.Add(GlColorShiftPercent); err != nil {
		return err
	}

	if err := c.Add(GlClear); err != nil {
		return err
	}

	if err := c.Add(GlCull); err != nil {
		return err
	}

	if err := c.Add(GlFarClip); err != nil {
		return err
	}

	if err := c.Add(GlFinish); err != nil {
		return err
	}

	if err := c.Add(GlFlashBlend); err != nil {
		return err
	}

	if err := c.Add(GlFullBrights); err != nil {
		return err
	}

	if err := c.Add(GlMaxSize); err != nil {
		return err
	}

	if err := c.Add(GlNoColors); err != nil {
		return err
	}

	if err := c.Add(GlOverBright); err != nil {
		return err
	}

	if err := c.Add(GlOverBrightModels); err != nil {
		return err
	}

	if err := c.Add(GlPicMip); err != nil {
		return err
	}

	if err := c.Add(GlPlayerMip); err != nil {
		return err
	}

	if err := c.Add(GlPolyBlend); err != nil {
		return err
	}

	if err := c.Add(GlSmoothModels); err != nil {
		return err
	}

	if err := c.Add(GlSubdivideSize); err != nil {
		return err
	}

	if err := c.Add(GlTextureMode); err != nil {
		return err
	}

	if err := c.Add(GlTextureAnisotropy); err != nil {
		return err
	}

	if err := c.Add(GlTripleBuffer); err != nil {
		return err
	}

	if err := c.Add(GlZFix); err != nil {
		return err
	}

	if err := c.Add(HostFrameRate); err != nil {
		return err
	}

	if err := c.Add(HostMaxFps); err != nil {
		return err
	}

	if err := c.Add(HostName); err != nil {
		return err
	}

	if err := c.Add(HostSpeeds); err != nil {
		return err
	}

	if err := c.Add(HostTimeScale); err != nil {
		return err
	}

	if err := c.Add(InputDebugKeys); err != nil {
		return err
	}

	if err := c.Add(LoadAs8Bit); err != nil {
		return err
	}

	if err := c.Add(LookSpring); err != nil {
		return err
	}

	if err := c.Add(LookStrafe); err != nil {
		return err
	}

	if err := c.Add(MaxEdicts); err != nil {
		return err
	}

	if err := c.Add(MouseForward); err != nil {
		return err
	}

	if err := c.Add(MousePitch); err != nil {
		return err
	}

	if err := c.Add(MouseSide); err != nil {
		return err
	}

	if err := c.Add(MouseYaw); err != nil {
		return err
	}

	if err := c.Add(NetMessageTimeout); err != nil {
		return err
	}

	if err := c.Add(NoExit); err != nil {
		return err
	}

	if err := c.Add(NoMonsters); err != nil {
		return err
	}

	if err := c.Add(NoSound); err != nil {
		return err
	}

	if err := c.Add(Pausable); err != nil {
		return err
	}

	if err := c.Add(Precache); err != nil {
		return err
	}

	if err := c.Add(RClearColor); err != nil {
		return err
	}

	if err := c.Add(RDrawEntities); err != nil {
		return err
	}

	if err := c.Add(RDrawFlat); err != nil {
		return err
	}

	if err := c.Add(RDrawViewModel); err != nil {
		return err
	}

	if err := c.Add(RDrawWorld); err != nil {
		return err
	}

	if err := c.Add(RDynamic); err != nil {
		return err
	}

	if err := c.Add(RFastSky); err != nil {
		return err
	}

	if err := c.Add(RFlatLightStyles); err != nil {
		return err
	}

	if err := c.Add(RFullBright); err != nil {
		return err
	}

	if err := c.Add(RLavaAlpha); err != nil {
		return err
	}

	if err := c.Add(RLerpModels); err != nil {
		return err
	}

	if err := c.Add(RLerpMove); err != nil {
		return err
	}

	if err := c.Add(RLightMap); err != nil {
		return err
	}

	if err := c.Add(RNoLerpList); err != nil {
		return err
	}

	if err := c.Add(RNoRefresh); err != nil {
		return err
	}

	if err := c.Add(RNoShadowList); err != nil {
		return err
	}

	if err := c.Add(RFullBrightList); err != nil {
		return err
	}

	if err := c.Add(RNoVis); err != nil {
		return err
	}

	if err := c.Add(ROldSkyLeaf); err != nil {
		return err
	}

	if err := c.Add(ROldWater); err != nil {
		return err
	}

	if err := c.Add(RParticles); err != nil {
		return err
	}

	if err := c.Add(RPos); err != nil {
		return err
	}

	if err := c.Add(RQuadParticles); err != nil {
		return err
	}

	if err := c.Add(RShadows); err != nil {
		return err
	}

	if err := c.Add(RShowBoxes); err != nil {
		return err
	}

	if err := c.Add(RShowTris); err != nil {
		return err
	}

	if err := c.Add(RSkyAlpha); err != nil {
		return err
	}

	if err := c.Add(RSkyFog); err != nil {
		return err
	}

	if err := c.Add(RSkyQuality); err != nil {
		return err
	}

	if err := c.Add(RSlimeAlpha); err != nil {
		return err
	}

	if err := c.Add(RSpeeds); err != nil {
		return err
	}

	if err := c.Add(RTeleAlpha); err != nil {
		return err
	}

	if err := c.Add(RWaterAlpha); err != nil {
		return err
	}

	if err := c.Add(RWaterQuality); err != nil {
		return err
	}

	if err := c.Add(RWaterWarp); err != nil {
		return err
	}

	if err := c.Add(SameLevel); err != nil {
		return err
	}

	if err := c.Add(Saved1); err != nil {
		return err
	}

	if err := c.Add(Saved2); err != nil {
		return err
	}

	if err := c.Add(Saved3); err != nil {
		return err
	}

	if err := c.Add(Saved4); err != nil {
		return err
	}

	if err := c.Add(SavedGameCfg); err != nil {
		return err
	}

	if err := c.Add(Scratch1); err != nil {
		return err
	}

	if err := c.Add(Scratch2); err != nil {
		return err
	}

	if err := c.Add(Scratch3); err != nil {
		return err
	}

	if err := c.Add(Scratch4); err != nil {
		return err
	}

	if err := c.Add(ScreenCenterTime); err != nil {
		return err
	}

	if err := c.Add(ScreenClock); err != nil {
		return err
	}

	if err := c.Add(ScreenConsoleAlpha); err != nil {
		return err
	}

	if err := c.Add(ScreenConsoleScale); err != nil {
		return err
	}

	if err := c.Add(ScreenConsoleSpeed); err != nil {
		return err
	}

	if err := c.Add(ScreenConsoleWidth); err != nil {
		return err
	}

	if err := c.Add(ScreenCrosshairScale); err != nil {
		return err
	}

	if err := c.Add(ScreenMenuScale); err != nil {
		return err
	}

	if err := c.Add(ScreenOffsetX); err != nil {
		return err
	}

	if err := c.Add(ScreenOffsetY); err != nil {
		return err
	}

	if err := c.Add(ScreenOffsetZ); err != nil {
		return err
	}

	if err := c.Add(ScreenPrintSpeed); err != nil {
		return err
	}

	if err := c.Add(ScreenStatusbarAlpha); err != nil {
		return err
	}

	if err := c.Add(ScreenStatusbarScale); err != nil {
		return err
	}

	if err := c.Add(ScreenShowFps); err != nil {
		return err
	}

	if err := c.Add(Sensitivity); err != nil {
		return err
	}

	if err := c.Add(ServerAccelerate); err != nil {
		return err
	}

	if err := c.Add(ServerAim); err != nil {
		return err
	}

	if err := c.Add(ServerAltNoClip); err != nil {
		return err
	}

	if err := c.Add(ServerEdgeFriction); err != nil {
		return err
	}

	if err := c.Add(ServerFreezeNonClients); err != nil {
		return err
	}

	if err := c.Add(ServerFriction); err != nil {
		return err
	}

	if err := c.Add(ServerGravity); err != nil {
		return err
	}

	if err := c.Add(ServerIdealPitchScale); err != nil {
		return err
	}

	if err := c.Add(ServerMaxSpeed); err != nil {
		return err
	}

	if err := c.Add(ServerMaxVelocity); err != nil {
		return err
	}

	if err := c.Add(ServerNoStep); err != nil {
		return err
	}

	if err := c.Add(ServerProfile); err != nil {
		return err
	}

	if err := c.Add(ServerStopSpeed); err != nil {
		return err
	}

	if err := c.Add(ShowPause); err != nil {
		return err
	}

	if err := c.Add(ShowRAM); err != nil {
		return err
	}

	if err := c.Add(ShowTurtle); err != nil {
		return err
	}

	if err := c.Add(Skill); err != nil {
		return err
	}

	if err := c.Add(SoundFilterQuality); err != nil {
		return err
	}

	if err := c.Add(SoundMixAhead); err != nil {
		return err
	}

	if err := c.Add(SoundMixSpeed); err != nil {
		return err
	}

	if err := c.Add(SoundNoExtraUpdate); err != nil {
		return err
	}

	if err := c.Add(SoundShow); err != nil {
		return err
	}

	if err := c.Add(SoundSpeed); err != nil {
		return err
	}

	if err := c.Add(TeamPlay); err != nil {
		return err
	}

	if err := c.Add(Temp1); err != nil {
		return err
	}

	if err := c.Add(Throttle); err != nil {
		return err
	}

	if err := c.Add(TicRate); err != nil {
		return err
	}

	if err := c.Add(TimeLimit); err != nil {
		return err
	}

	if err := c.Add(VideoBorderLess); err != nil {
		return err
	}

	if err := c.Add(VideoDesktopFullscreen); err != nil {
		return err
	}

	if err := c.Add(VideoFsaa); err != nil {
		return err
	}

	if err := c.Add(VideoFullscreen); err != nil {
		return err
	}

	if err := c.Add(VideoHeight); err != nil {
		return err
	}

	if err := c.Add(VideoVerticalSync); err != nil {
		return err
	}

	if err := c.Add(VideoWidth); err != nil {
		return err
	}

	if err := c.Add(ViewCenterMove); err != nil {
		return err
	}

	if err := c.Add(ViewCenterSpeed); err != nil {
		return err
	}

	if err := c.Add(ViewGunKick); err != nil {
		return err
	}

	if err := c.Add(ViewIPitchCycle); err != nil {
		return err
	}

	if err := c.Add(ViewIPitchLevel); err != nil {
		return err
	}

	if err := c.Add(ViewIRollCycle); err != nil {
		return err
	}

	if err := c.Add(ViewIRollLevel); err != nil {
		return err
	}

	if err := c.Add(ViewIYawCycle); err != nil {
		return err
	}

	if err := c.Add(ViewIYawLevel); err != nil {
		return err
	}

	if err := c.Add(ViewIdleScale); err != nil {
		return err
	}

	if err := c.Add(ViewKickPitch); err != nil {
		return err
	}

	if err := c.Add(ViewKickRoll); err != nil {
		return err
	}

	if err := c.Add(ViewKickTime); err != nil {
		return err
	}

	if err := c.Add(ViewSize); err != nil {
		return err
	}

	if err := c.Add(Volume); err != nil {
		return err
	}

	if err := c.Add(registered); err != nil {
		return err
	}
	return nil
}

func init() {
	cvar.Must(Register(cvar.Global()))
}

func CalcRoll(angles, velocity vec.Vec3) float32 {
	_, right, _ := vec.AngleVectors(angles)

	side := vec.Dot(velocity, right)
	neg := math32.Signbit(side)
	side = math32.Abs(side)

	r := ClientRollAngle.Value()
	rs := ClientRollSpeed.Value()

	if side < rs {
		side *= r / rs
		if neg {
			return -side
		}
		return side
	}
	if neg {
		return -r
	}
	return r
}
