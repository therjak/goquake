// SPDX-License-Identifier: GPL-2.0-or-later

package cvars

import (
	"strings"

	"goquake/cvar"
)

var (
	AmbientFade            *cvar.Cvar
	AmbientLevel           *cvar.Cvar
	BackgroundVolume       *cvar.Cvar
	Campaign               *cvar.Cvar
	CfgUnbindAll           *cvar.Cvar
	ChaseActive            *cvar.Cvar
	ChaseBack              *cvar.Cvar
	ChaseRight             *cvar.Cvar
	ChaseUp                *cvar.Cvar
	ClientAngleSpeedKey    *cvar.Cvar
	ClientBackSpeed        *cvar.Cvar
	ClientBob              *cvar.Cvar
	ClientBobCycle         *cvar.Cvar
	ClientBobUp            *cvar.Cvar
	ClientColor            *cvar.Cvar
	ClientForwardSpeed     *cvar.Cvar
	ClientMaxPitch         *cvar.Cvar
	ClientMinPitch         *cvar.Cvar
	ClientMoveSpeedKey     *cvar.Cvar
	ClientName             *cvar.Cvar
	ClientNoLerp           *cvar.Cvar
	ClientPitchSpeed       *cvar.Cvar
	ClientRollAngle        *cvar.Cvar
	ClientRollSpeed        *cvar.Cvar
	ClientShowNet          *cvar.Cvar
	ClientSideSpeed        *cvar.Cvar
	ClientUpSpeed          *cvar.Cvar
	ClientYawSpeed         *cvar.Cvar
	ConsoleLogCenterPrint  *cvar.Cvar
	ConsoleNotifyTime      *cvar.Cvar
	Contrast               *cvar.Cvar
	Coop                   *cvar.Cvar
	Crosshair              *cvar.Cvar
	DeathMatch             *cvar.Cvar
	DevStats               *cvar.Cvar
	Developer              *cvar.Cvar
	ExternalEnts           *cvar.Cvar
	Fov                    *cvar.Cvar
	FovAdapt               *cvar.Cvar
	FragLimit              *cvar.Cvar
	GameCfg                *cvar.Cvar
	Gamma                  *cvar.Cvar
	GlAffineModels         *cvar.Cvar
	GlColorShiftPercent    *cvar.Cvar
	GlClear                *cvar.Cvar
	GlCull                 *cvar.Cvar
	GlFarClip              *cvar.Cvar
	GlFinish               *cvar.Cvar
	GlFlashBlend           *cvar.Cvar
	GlFullBrights          *cvar.Cvar
	GlMaxSize              *cvar.Cvar
	GlNoColors             *cvar.Cvar
	GlOverBright           *cvar.Cvar
	GlOverBrightModels     *cvar.Cvar
	GlPicMip               *cvar.Cvar
	GlPlayerMip            *cvar.Cvar
	GlPolyBlend            *cvar.Cvar
	GlSmoothModels         *cvar.Cvar
	GlSubdivideSize        *cvar.Cvar
	GlTextureMode          *cvar.Cvar
	GlTextureAnisotropy    *cvar.Cvar
	GlTripleBuffer         *cvar.Cvar
	GlZFix                 *cvar.Cvar
	HostFrameRate          *cvar.Cvar
	HostMaxFps             *cvar.Cvar
	HostName               *cvar.Cvar
	HostSpeeds             *cvar.Cvar
	HostTimeScale          *cvar.Cvar
	InputDebugKeys         *cvar.Cvar
	LoadAs8Bit             *cvar.Cvar
	LookSpring             *cvar.Cvar
	LookStrafe             *cvar.Cvar
	MaxEdicts              *cvar.Cvar
	MouseForward           *cvar.Cvar
	MousePitch             *cvar.Cvar
	MouseSide              *cvar.Cvar
	MouseYaw               *cvar.Cvar
	NetMessageTimeout      *cvar.Cvar
	NoExit                 *cvar.Cvar
	NoMonsters             *cvar.Cvar
	NoSound                *cvar.Cvar
	Pausable               *cvar.Cvar
	Precache               *cvar.Cvar
	RClearColor            *cvar.Cvar
	RDrawEntities          *cvar.Cvar
	RDrawFlat              *cvar.Cvar
	RDrawViewModel         *cvar.Cvar
	RDrawWorld             *cvar.Cvar
	RDynamic               *cvar.Cvar
	RFastSky               *cvar.Cvar
	RFlatLightStyles       *cvar.Cvar
	RFullBright            *cvar.Cvar
	RLavaAlpha             *cvar.Cvar
	RLerpModels            *cvar.Cvar
	RLerpMove              *cvar.Cvar
	RLightMap              *cvar.Cvar
	RNoLerpList            *cvar.Cvar
	RNoRefresh             *cvar.Cvar
	RNoShadowList          *cvar.Cvar
	RFullBrightList        *cvar.Cvar
	RNoVis                 *cvar.Cvar
	ROldSkyLeaf            *cvar.Cvar
	ROldWater              *cvar.Cvar
	RParticles             *cvar.Cvar
	RPos                   *cvar.Cvar
	RQuadParticles         *cvar.Cvar
	RShadows               *cvar.Cvar
	RShowBoxes             *cvar.Cvar
	RShowTris              *cvar.Cvar
	RSkyAlpha              *cvar.Cvar
	RSkyFog                *cvar.Cvar
	RSkyQuality            *cvar.Cvar
	RSlimeAlpha            *cvar.Cvar
	RSpeeds                *cvar.Cvar
	RTeleAlpha             *cvar.Cvar
	RWaterAlpha            *cvar.Cvar
	RWaterQuality          *cvar.Cvar
	RWaterWarp             *cvar.Cvar
	SameLevel              *cvar.Cvar
	Saved1                 *cvar.Cvar
	Saved2                 *cvar.Cvar
	Saved3                 *cvar.Cvar
	Saved4                 *cvar.Cvar
	SavedGameCfg           *cvar.Cvar
	Scratch1               *cvar.Cvar
	Scratch2               *cvar.Cvar
	Scratch3               *cvar.Cvar
	Scratch4               *cvar.Cvar
	ScreenCenterTime       *cvar.Cvar
	ScreenClock            *cvar.Cvar
	ScreenConsoleAlpha     *cvar.Cvar
	ScreenConsoleScale     *cvar.Cvar
	ScreenConsoleSpeed     *cvar.Cvar
	ScreenConsoleWidth     *cvar.Cvar
	ScreenCrosshairScale   *cvar.Cvar
	ScreenMenuScale        *cvar.Cvar
	ScreenOffsetX          *cvar.Cvar
	ScreenOffsetY          *cvar.Cvar
	ScreenOffsetZ          *cvar.Cvar
	ScreenPrintSpeed       *cvar.Cvar
	ScreenStatusbarAlpha   *cvar.Cvar
	ScreenStatusbarScale   *cvar.Cvar
	ScreenShowFps          *cvar.Cvar
	Sensitivity            *cvar.Cvar
	ServerAccelerate       *cvar.Cvar
	ServerAim              *cvar.Cvar
	ServerAltNoClip        *cvar.Cvar
	ServerEdgeFriction     *cvar.Cvar
	ServerFreezeNonClients *cvar.Cvar
	ServerFriction         *cvar.Cvar
	ServerGravity          *cvar.Cvar
	ServerIdealPitchScale  *cvar.Cvar
	ServerMaxSpeed         *cvar.Cvar
	ServerMaxVelocity      *cvar.Cvar
	ServerNoStep           *cvar.Cvar
	ServerProfile          *cvar.Cvar
	ServerStopSpeed        *cvar.Cvar
	ShowPause              *cvar.Cvar
	ShowRAM                *cvar.Cvar
	ShowTurtle             *cvar.Cvar
	Skill                  *cvar.Cvar
	SoundFilterQuality     *cvar.Cvar
	SoundMixAhead          *cvar.Cvar
	SoundMixSpeed          *cvar.Cvar
	SoundNoExtraUpdate     *cvar.Cvar
	SoundShow              *cvar.Cvar
	SoundSpeed             *cvar.Cvar
	TeamPlay               *cvar.Cvar
	Temp1                  *cvar.Cvar
	Throttle               *cvar.Cvar
	TicRate                *cvar.Cvar
	TimeLimit              *cvar.Cvar
	VideoBorderLess        *cvar.Cvar
	VideoDesktopFullscreen *cvar.Cvar
	VideoFsaa              *cvar.Cvar
	VideoFullscreen        *cvar.Cvar
	VideoHeight            *cvar.Cvar
	VideoVerticalSync      *cvar.Cvar
	VideoWidth             *cvar.Cvar
	ViewCenterMove         *cvar.Cvar
	ViewCenterSpeed        *cvar.Cvar
	ViewGunKick            *cvar.Cvar
	ViewIPitchCycle        *cvar.Cvar
	ViewIPitchLevel        *cvar.Cvar
	ViewIRollCycle         *cvar.Cvar
	ViewIRollLevel         *cvar.Cvar
	ViewIYawCycle          *cvar.Cvar
	ViewIYawLevel          *cvar.Cvar
	ViewIdleScale          *cvar.Cvar
	ViewKickPitch          *cvar.Cvar
	ViewKickRoll           *cvar.Cvar
	ViewKickTime           *cvar.Cvar
	ViewSize               *cvar.Cvar
	Volume                 *cvar.Cvar
)

func init() {
	ClientColor = cvar.MustRegister("_cl_color", "0", cvar.ARCHIVE)
	ClientName = cvar.MustRegister("_cl_name", "player", cvar.ARCHIVE)

	AmbientFade = cvar.MustRegister("ambient_fade", "100", cvar.NONE)
	AmbientLevel = cvar.MustRegister("ambient_level", "0.3", cvar.NONE)
	BackgroundVolume = cvar.MustRegister("bgmvolume", "1", cvar.ARCHIVE) // cd music volume, therjak: this is dead, only used in menu
	Campaign = cvar.MustRegister("campaign", "0", cvar.NONE)             // 2021 release
	CfgUnbindAll = cvar.MustRegister("cfg_unbindall", "1", cvar.ARCHIVE)
	ChaseActive = cvar.MustRegister("chase_active", "0", cvar.NONE)
	ChaseBack = cvar.MustRegister("chase_back", "100", cvar.NONE)
	ChaseRight = cvar.MustRegister("chase_right", "0", cvar.NONE)
	ChaseUp = cvar.MustRegister("chase_up", "16", cvar.NONE)
	ClientAngleSpeedKey = cvar.MustRegister("cl_anglespeedkey", "1.5", cvar.NONE)
	ClientBackSpeed = cvar.MustRegister("cl_backspeed", "200", cvar.ARCHIVE)
	ClientBob = cvar.MustRegister("cl_bob", "0.02", cvar.NONE)
	ClientBobCycle = cvar.MustRegister("cl_bobcycle", "0.6", cvar.NONE)
	ClientBobUp = cvar.MustRegister("cl_bobup", "0.5", cvar.NONE)
	ClientForwardSpeed = cvar.MustRegister("cl_forwardspeed", "200", cvar.ARCHIVE)
	ClientMaxPitch = cvar.MustRegister("cl_maxpitch", "90", cvar.ARCHIVE)
	ClientMinPitch = cvar.MustRegister("cl_minpitch", "-90", cvar.ARCHIVE)
	ClientMoveSpeedKey = cvar.MustRegister("cl_movespeedkey", "2.0", cvar.NONE)
	ClientNoLerp = cvar.MustRegister("cl_nolerp", "0", cvar.NONE)
	ClientPitchSpeed = cvar.MustRegister("cl_pitchspeed", "150", cvar.NONE)
	ClientRollAngle = cvar.MustRegister("cl_rollangle", "2.0", cvar.NONE)
	ClientRollSpeed = cvar.MustRegister("cl_rollspeed", "200", cvar.NONE)
	ClientShowNet = cvar.MustRegister("cl_shownet", "0", cvar.NONE)
	ClientSideSpeed = cvar.MustRegister("cl_sidespeed", "350", cvar.NONE)
	ClientUpSpeed = cvar.MustRegister("cl_upspeed", "200", cvar.NONE)
	ClientYawSpeed = cvar.MustRegister("cl_yawspeed", "140", cvar.NONE)
	ConsoleLogCenterPrint = cvar.MustRegister("con_logcenterprint", "1", cvar.NONE)
	ConsoleNotifyTime = cvar.MustRegister("con_notifytime", "3", cvar.NONE)
	Contrast = cvar.MustRegister("contrast", "1", cvar.ARCHIVE)
	Coop = cvar.MustRegister("coop", "0", cvar.NONE)
	Crosshair = cvar.MustRegister("crosshair", "0", cvar.ARCHIVE)
	DeathMatch = cvar.MustRegister("deathmatch", "0", cvar.NONE)
	DevStats = cvar.MustRegister("devstats", "0", cvar.NONE)
	Developer = cvar.MustRegister("developer", "0", cvar.NONE)
	ExternalEnts = cvar.MustRegister("external_ents", "1", cvar.ARCHIVE)
	Fov = cvar.MustRegister("fov", "90", cvar.NONE)
	FovAdapt = cvar.MustRegister("fov_adapt", "1", cvar.ARCHIVE)
	FragLimit = cvar.MustRegister("fraglimit", "0", cvar.NOTIFY|cvar.SERVERINFO)
	GameCfg = cvar.MustRegister("gamecfg", "0", cvar.NONE)
	Gamma = cvar.MustRegister("gamma", "1", cvar.ARCHIVE)
	GlAffineModels = cvar.MustRegister("gl_affinemodels", "0", cvar.NONE)
	GlColorShiftPercent = cvar.MustRegister("gl_cshiftpercent", "100", cvar.NONE)
	GlClear = cvar.MustRegister("gl_clear", "1", cvar.NONE)
	GlCull = cvar.MustRegister("gl_cull", "1", cvar.NONE)
	GlFarClip = cvar.MustRegister("gl_farclip", "16384", cvar.ARCHIVE)
	GlFinish = cvar.MustRegister("gl_finish", "0", cvar.NONE)
	GlFlashBlend = cvar.MustRegister("gl_flashblend", "0", cvar.ARCHIVE)
	GlFullBrights = cvar.MustRegister("gl_fullbrights", "1", cvar.ARCHIVE)
	GlMaxSize = cvar.MustRegister("gl_max_size", "0", cvar.NONE)
	GlNoColors = cvar.MustRegister("gl_nocolors", "0", cvar.NONE)
	GlOverBright = cvar.MustRegister("gl_overbright", "1", cvar.ARCHIVE)
	GlOverBrightModels = cvar.MustRegister("gl_overbright_models", "1", cvar.ARCHIVE)
	GlPicMip = cvar.MustRegister("gl_picmip", "0", cvar.NONE)
	GlPlayerMip = cvar.MustRegister("gl_playermip", "0", cvar.NONE)
	GlPolyBlend = cvar.MustRegister("gl_polyblend", "1", cvar.NONE)
	GlSmoothModels = cvar.MustRegister("gl_smoothmodels", "1", cvar.NONE)
	GlSubdivideSize = cvar.MustRegister("gl_subdivide_size", "128", cvar.ARCHIVE)
	// correct value is filled in later.
	GlTextureMode = cvar.MustRegister("gl_texturemode", "", cvar.ARCHIVE)
	GlTextureAnisotropy = cvar.MustRegister("gl_texture_anisotropy", "1", cvar.ARCHIVE)
	GlTripleBuffer = cvar.MustRegister("gl_triplebuffer", "1", cvar.ARCHIVE)
	GlZFix = cvar.MustRegister("gl_zfix", "0", cvar.NONE)
	HostFrameRate = cvar.MustRegister("host_framerate", "0", cvar.NONE)
	HostMaxFps = cvar.MustRegister("host_maxfps", "72", cvar.ARCHIVE)
	HostName = cvar.MustRegister("hostname", "UNNAMED", cvar.NONE)
	HostSpeeds = cvar.MustRegister("host_speeds", "0", cvar.NONE)
	HostTimeScale = cvar.MustRegister("host_timescale", "0", cvar.NONE)
	InputDebugKeys = cvar.MustRegister("in_debugkeys", "0", cvar.NONE)
	LoadAs8Bit = cvar.MustRegister("loadas8bit", "0", cvar.NONE)
	LookSpring = cvar.MustRegister("lookspring", "0", cvar.ARCHIVE)
	LookStrafe = cvar.MustRegister("lookstrafe", "0", cvar.ARCHIVE)
	MaxEdicts = cvar.MustRegister("max_edicts", "15000", cvar.NONE)
	MouseForward = cvar.MustRegister("m_forward", "1", cvar.ARCHIVE)
	MousePitch = cvar.MustRegister("m_pitch", "0.022", cvar.ARCHIVE)
	MouseSide = cvar.MustRegister("m_side", "0.8", cvar.ARCHIVE)
	MouseYaw = cvar.MustRegister("m_yaw", "0.022", cvar.ARCHIVE)
	NetMessageTimeout = cvar.MustRegister("net_messagetimeout", "300", cvar.NONE)
	NoExit = cvar.MustRegister("noexit", "0", cvar.NOTIFY|cvar.SERVERINFO)
	NoMonsters = cvar.MustRegister("nomonsters", "0", cvar.NONE)
	NoSound = cvar.MustRegister("nosound", "0", cvar.NONE)
	Pausable = cvar.MustRegister("pausable", "1", cvar.NONE)
	Precache = cvar.MustRegister("precache", "1", cvar.NONE)
	RClearColor = cvar.MustRegister("r_clearcolor", "2", cvar.ARCHIVE)
	RDrawEntities = cvar.MustRegister("r_drawentities", "1", cvar.NONE)
	RDrawFlat = cvar.MustRegister("r_drawflat", "0", cvar.NONE)
	RDrawViewModel = cvar.MustRegister("r_drawviewmodel", "1", cvar.NONE)
	RDrawWorld = cvar.MustRegister("r_drawworld", "1", cvar.NONE)
	RDynamic = cvar.MustRegister("r_dynamic", "1", cvar.ARCHIVE)
	RFastSky = cvar.MustRegister("r_fastsky", "0", cvar.NONE)
	RFlatLightStyles = cvar.MustRegister("r_flatlightstyles", "0", cvar.NONE)
	RFullBright = cvar.MustRegister("r_fullbright", "0", cvar.NONE)
	RLavaAlpha = cvar.MustRegister("r_lavaalpha", "0", cvar.NONE)
	RLerpModels = cvar.MustRegister("r_lerpmodels", "1", cvar.NONE)
	RLerpMove = cvar.MustRegister("r_lerpmove", "1", cvar.NONE)
	RLightMap = cvar.MustRegister("r_lightmap", "0", cvar.NONE)
	RNoRefresh = cvar.MustRegister("r_norefresh", "0", cvar.NONE)
	RNoVis = cvar.MustRegister("r_novis", "0", cvar.ARCHIVE)
	ROldSkyLeaf = cvar.MustRegister("r_oldskyleaf", "0", cvar.NONE)
	ROldWater = cvar.MustRegister("r_oldwater", "0", cvar.ARCHIVE)
	RParticles = cvar.MustRegister("r_particles", "1", cvar.ARCHIVE)
	RPos = cvar.MustRegister("r_pos", "0", cvar.NONE)
	RQuadParticles = cvar.MustRegister("r_quadparticles", "1", cvar.ARCHIVE)
	RShadows = cvar.MustRegister("r_shadows", "0", cvar.ARCHIVE)
	RShowBoxes = cvar.MustRegister("r_showbboxes", "0", cvar.NONE)
	RShowTris = cvar.MustRegister("r_showtris", "0", cvar.NONE)
	RSkyAlpha = cvar.MustRegister("r_skyalpha", "1", cvar.NONE)
	RSkyFog = cvar.MustRegister("r_skyfog", "0.5", cvar.NONE)
	RSkyQuality = cvar.MustRegister("r_sky_quality", "12", cvar.NONE)
	RSlimeAlpha = cvar.MustRegister("r_slimealpha", "0", cvar.NONE)
	RSpeeds = cvar.MustRegister("r_speeds", "0", cvar.NONE)
	RTeleAlpha = cvar.MustRegister("r_telealpha", "0", cvar.NONE)
	RWaterAlpha = cvar.MustRegister("r_wateralpha", "1", cvar.ARCHIVE)
	RWaterQuality = cvar.MustRegister("r_waterquality", "8", cvar.NONE)
	RWaterWarp = cvar.MustRegister("r_waterwarp", "1", cvar.NONE)
	SameLevel = cvar.MustRegister("samelevel", "0", cvar.NONE)
	Saved1 = cvar.MustRegister("saved1", "0", cvar.ARCHIVE)
	Saved2 = cvar.MustRegister("saved2", "0", cvar.ARCHIVE)
	Saved3 = cvar.MustRegister("saved3", "0", cvar.ARCHIVE)
	Saved4 = cvar.MustRegister("saved4", "0", cvar.ARCHIVE)
	SavedGameCfg = cvar.MustRegister("savedgamecfg", "0", cvar.ARCHIVE)
	Scratch1 = cvar.MustRegister("scratch1", "0", cvar.NONE)
	Scratch2 = cvar.MustRegister("scratch2", "0", cvar.NONE)
	Scratch3 = cvar.MustRegister("scratch3", "0", cvar.NONE)
	Scratch4 = cvar.MustRegister("scratch4", "0", cvar.NONE)
	ScreenCenterTime = cvar.MustRegister("scr_centertime", "2", cvar.NONE)
	ScreenClock = cvar.MustRegister("scr_clock", "0", cvar.NONE)
	ScreenConsoleAlpha = cvar.MustRegister("scr_conalpha", "0.5", cvar.ARCHIVE)
	ScreenConsoleSpeed = cvar.MustRegister("scr_conspeed", "500", cvar.ARCHIVE)
	ScreenConsoleScale = cvar.MustRegister("scr_conscale", "1", cvar.ARCHIVE)
	ScreenConsoleWidth = cvar.MustRegister("scr_conwidth", "0", cvar.ARCHIVE)
	ScreenCrosshairScale = cvar.MustRegister("scr_crosshairscale", "1", cvar.ARCHIVE)
	ScreenMenuScale = cvar.MustRegister("scr_menuscale", "1", cvar.ARCHIVE)
	ScreenOffsetX = cvar.MustRegister("scr_ofsx", "0", cvar.NONE)
	ScreenOffsetY = cvar.MustRegister("scr_ofsy", "0", cvar.NONE)
	ScreenOffsetZ = cvar.MustRegister("scr_ofsz", "0", cvar.NONE)
	ScreenPrintSpeed = cvar.MustRegister("scr_printspeed", "8", cvar.NONE)
	ScreenStatusbarAlpha = cvar.MustRegister("scr_sbaralpha", "0.75", cvar.ARCHIVE)
	ScreenStatusbarScale = cvar.MustRegister("scr_sbarscale", "1", cvar.ARCHIVE)
	ScreenShowFps = cvar.MustRegister("scr_showfps", "0", cvar.NONE)
	Sensitivity = cvar.MustRegister("sensitivity", "3", cvar.ARCHIVE)
	ServerAccelerate = cvar.MustRegister("sv_accelerate", "10", cvar.NONE)
	ServerAim = cvar.MustRegister("sv_aim", "1", cvar.NONE)
	ServerAltNoClip = cvar.MustRegister("sv_altnoclip", "1", cvar.ARCHIVE)
	ServerEdgeFriction = cvar.MustRegister("edgefriction", "2", cvar.NONE)
	ServerFreezeNonClients = cvar.MustRegister("sv_freezenonclients", "0", cvar.NONE)
	ServerFriction = cvar.MustRegister("sv_friction", "4", cvar.NOTIFY|cvar.SERVERINFO)
	ServerGravity = cvar.MustRegister("sv_gravity", "800", cvar.NOTIFY|cvar.SERVERINFO)
	ServerIdealPitchScale = cvar.MustRegister("sv_idealpitchscale", "0.8", cvar.NONE)
	ServerMaxSpeed = cvar.MustRegister("sv_maxspeed", "320", cvar.NOTIFY|cvar.SERVERINFO)
	ServerMaxVelocity = cvar.MustRegister("sv_maxvelocity", "2000", cvar.NONE)
	ServerNoStep = cvar.MustRegister("sv_nostep", "0", cvar.NONE)
	ServerProfile = cvar.MustRegister("serverprofile", "0", cvar.NONE)
	ServerStopSpeed = cvar.MustRegister("sv_stopspeed", "100", cvar.NONE)
	ShowPause = cvar.MustRegister("showpause", "1", cvar.NONE)
	ShowRAM = cvar.MustRegister("showram", "1", cvar.NONE)
	ShowTurtle = cvar.MustRegister("showturtle", "0", cvar.NONE)
	Skill = cvar.MustRegister("skill", "1", cvar.NONE)
	SoundFilterQuality = cvar.MustRegister("snd_filterquality", "1", cvar.NONE) // 5 on win, 1 on all other
	SoundMixAhead = cvar.MustRegister("snd_mixahead", "0.1", cvar.ARCHIVE)
	SoundMixSpeed = cvar.MustRegister("snd_mixspeed", "44100", cvar.NONE)
	SoundNoExtraUpdate = cvar.MustRegister("snd_noextraupdate", "0", cvar.NONE)
	SoundShow = cvar.MustRegister("snd_show", "0", cvar.NONE)
	SoundSpeed = cvar.MustRegister("sndspeed", "11025", cvar.NONE)
	TeamPlay = cvar.MustRegister("teamplay", "0", cvar.NOTIFY|cvar.SERVERINFO)
	Temp1 = cvar.MustRegister("temp1", "0", cvar.NONE)
	Throttle = cvar.MustRegister("sys_throttle", "0.02", cvar.ARCHIVE)
	TicRate = cvar.MustRegister("sys_ticrate", "0.05", cvar.NONE)
	TimeLimit = cvar.MustRegister("timelimit", "0", cvar.NOTIFY|cvar.SERVERINFO)
	VideoBorderLess = cvar.MustRegister("vid_borderless", "0", cvar.ARCHIVE)
	VideoDesktopFullscreen = cvar.MustRegister("vid_desktopfullscreen", "0", cvar.ARCHIVE)
	VideoFsaa = cvar.MustRegister("vid_fsaa", "0", cvar.ARCHIVE)
	VideoFullscreen = cvar.MustRegister("vid_fullscreen", "0", cvar.ARCHIVE)
	VideoHeight = cvar.MustRegister("vid_height", "600", cvar.ARCHIVE)
	VideoVerticalSync = cvar.MustRegister("vid_vsync", "0", cvar.ARCHIVE)
	VideoWidth = cvar.MustRegister("vid_width", "800", cvar.ARCHIVE)
	ViewCenterMove = cvar.MustRegister("v_centermove", "0.15", cvar.NONE)
	ViewCenterSpeed = cvar.MustRegister("v_centerspeed", "500", cvar.NONE)
	ViewGunKick = cvar.MustRegister("v_gunkick", "1", cvar.NONE)
	ViewIPitchCycle = cvar.MustRegister("v_ipitch_cycle", "1", cvar.NONE)
	ViewIPitchLevel = cvar.MustRegister("v_ipitch_level", "0.3", cvar.NONE)
	ViewIRollCycle = cvar.MustRegister("v_iroll_cycle", "0.5", cvar.NONE)
	ViewIRollLevel = cvar.MustRegister("v_iroll_level", "0.1", cvar.NONE)
	ViewIYawCycle = cvar.MustRegister("v_iyaw_cycle", "2", cvar.NONE)
	ViewIYawLevel = cvar.MustRegister("v_iyaw_level", "0.3", cvar.NONE)
	ViewIdleScale = cvar.MustRegister("v_idlescale", "0", cvar.NONE)
	ViewKickPitch = cvar.MustRegister("v_kickpitch", "0.6", cvar.NONE)
	ViewKickRoll = cvar.MustRegister("v_kickroll", "0.6", cvar.NONE)
	ViewKickTime = cvar.MustRegister("v_kicktime", "0.5", cvar.NONE)
	ViewSize = cvar.MustRegister("viewsize", "100", cvar.ARCHIVE)
	Volume = cvar.MustRegister("volume", "0.7", cvar.ARCHIVE)

	// this cvar gets read from within the vm
	cvar.MustRegister("registered", "1", cvar.ROM)

	RNoLerpList = cvar.MustRegister("r_nolerp_list", strings.Join([]string{
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
	RNoShadowList = cvar.MustRegister("r_noshadow_list", strings.Join([]string{
		"progs/flame2.mdl",
		"progs/flame.mdl",
		"progs/bolt1.mdl",
		"progs/bolt2.mdl",
		"progs/bolt3.mdl",
		"progs/laser.mdl",
	}, ","), cvar.NONE)

	RFullBrightList = cvar.MustRegister("r_fullbright_list", strings.Join([]string{
		"progs/flame2.mdl",
		"progs/flame.mdl",
		"progs/boss.mdl",
	}, ","), cvar.NONE)
}
