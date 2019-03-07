package server

const (
	//
	// server to client
	//
	Bad        = 0
	Nop        = 1
	Disconnect = 2
	// [byte] [long]
	UpdateStat = 3
	// [long] server version
	Version = 4
	// [short] entity number
	SetView = 5
	// <see code>
	Sound = 6
	// [float] server time
	Time = 7
	// [string] null terminated string
	Print = 8
	// [string] stuffed into client's console buffer
	// the string should be \n terminated
	StuffText = 9
	// [angle3] set the view angle to this absolute value
	SetAngle = 10
	// [long] version
	// [string] signon string
	// [string]..[0]model cache
	// [string]...[0]sounds cache
	ServerInfo = 11
	// [byte] [string]
	LightStyle = 12
	// [byte] [string]
	UpdateName = 13
	// [byte] [short]
	UpdateFrags = 14
	// <shortbits + data>
	ClientData = 15
	// <see code>
	StopSound = 16
	// [byte] [byte]
	UpdateColors = 17
	// [vec3] <variable>
	Particle    = 18
	Damage      = 19
	SpawnStatic = 20
	// svc_spawnbinary		=21
	SpawnBaseline = 22
	TempEntity    = 23
	// [byte] on / off
	SetPause = 24
	// [byte]  used for the signon sequence
	SignonNum = 25
	// [string] to put in center of the screen
	Centerprint   = 26
	KilledMonster = 27
	FoundSecret   = 28
	// [coord3] [byte] samp [byte] vol [byte] aten
	SpawnStaticSound = 29
	// [string] music
	Intermission = 30
	// [string] music [string] text
	Finale = 31
	// [byte] track [byte] looptrack
	CDTrack    = 32
	SellScreen = 33
	Cutscene   = 34

	// johnfitz -- PROTOCOL_FITZQUAKE -- new server messages

	// [string] name
	Skybox = 37
	BF     = 40
	// [byte] density [byte] red [byte] green [byte] blue [float] time
	Fog = 41
	// support for large modelindex, large framenum, alpha, using flags
	SpawnBaseline2 = 42
	// support for large modelindex, large framenum, alpha, using flags
	SpawnStatic2 = 43
	// [coord3] [short] samp [byte] vol [byte] aten
	SpawnStaticSound2 = 44

	// johnfitz
)

const (
	GameCoop       = iota
	GameDeathmatch = iota
)

const (
	SoundVolume      = 1 << iota
	SoundAttenuation = 1 << iota
	SoundLooping     = 1 << iota
	SoundLargeEntity = 1 << iota // fitzquake
	SoundLargeSound  = 1 << iota // fitzquake
)

const (
	EffectBrightField = 1 << iota
	EffectMuzzleFlash = 1 << iota
	EffectBrightLight = 1 << iota
	EffectDimLight    = 1 << iota
)

const (
	SpawnFlagNotEasy      = 1 << (8 + iota)
	SpawnFlagNotMedium    = 1 << (8 + iota)
	SpawnFlagNotHard      = 1 << (8 + iota)
	SpawnFlagNotDeathmath = 1 << (8 + iota)
)

const (
	DEFAULT_VIEWHEIGHT = 22
)

const (
	SU_VIEWHEIGHT   = (1 << iota)
	SU_IDEALPITCH   = (1 << iota)
	SU_PUNCH1       = (1 << iota)
	SU_PUNCH2       = (1 << iota)
	SU_PUNCH3       = (1 << iota)
	SU_VELOCITY1    = (1 << iota)
	SU_VELOCITY2    = (1 << iota)
	SU_VELOCITY3    = (1 << iota)
	SU_UNUSED8      = (1 << iota) // AVAILABLE BIT
	SU_ITEMS        = (1 << iota)
	SU_ONGROUND     = (1 << iota) // no data follows, the bit is it
	SU_INWATER      = (1 << iota) // no data follows, the bit is it
	SU_WEAPONFRAME  = (1 << iota)
	SU_ARMOR        = (1 << iota)
	SU_WEAPON       = (1 << iota)
	SU_EXTEND1      = (1 << iota) // another byte to follow
	SU_WEAPON2      = (1 << iota) // 1 byte, this is .weaponmodel & 0xFF00 (second byte)
	SU_ARMOR2       = (1 << iota) // 1 byte, this is .armorvalue & 0xFF00 (second byte)
	SU_AMMO2        = (1 << iota) // 1 byte, this is .currentammo & 0xFF00 (second byte)
	SU_SHELLS2      = (1 << iota) // 1 byte, this is .ammo_shells & 0xFF00 (second byte)
	SU_NAILS2       = (1 << iota) // 1 byte, this is .ammo_nails & 0xFF00 (second byte)
	SU_ROCKETS2     = (1 << iota) // 1 byte, this is .ammo_rockets & 0xFF00 (second byte)
	SU_CELLS2       = (1 << iota) // 1 byte, this is .ammo_cells & 0xFF00 (second byte)
	SU_EXTEND2      = (1 << iota) // another byte to follow
	SU_WEAPONFRAME2 = (1 << iota) // 1 byte, this is .weaponframe & 0xFF00 (second byte)
	SU_WEAPONALPHA  = (1 << iota) // 1 byte, this is alpha for weaponmodel, uses ENTALPHA_ENCODE, not sent if ENTALPHA_DEFAULT
	SU_UNUSED26     = (1 << iota)
	SU_UNUSED27     = (1 << iota)
	SU_UNUSED28     = (1 << iota)
	SU_UNUSED29     = (1 << iota)
	SU_UNUSED30     = (1 << iota)
	SU_EXTEND3      = (1 << iota) // another byte to follow, future expansion
)
