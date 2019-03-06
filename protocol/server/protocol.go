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
