package progs

const (
	ProgVersion   = 6
	MaxParms      = 8 // matches OffsetParm0-7
	ProgHeaderCRC = 5927
)

// etype_t
const (
	EV_Bad = iota - 1
	EV_Void
	EV_String
	EV_Float
	EV_Vector
	EV_Entity
	EV_Field
	EV_Function
	EV_Pointer
)

const (
	OffSetNull     = iota           // 0
	OffsetReturn                    // 1
	OffsetParm0    = 1 + (iota-1)*3 // 4, leave 3 for each parm to hold vectors
	OffsetParm1                     // 7
	OffsetParm2                     // 10
	OffsetParm3                     // 13
	OffsetParm4                     // 16
	OffsetParm5                     // 19
	OffsetParm6                     // 22
	OffsetParm7                     // 25
	ReservedOffset                  // 28
)

const (
	FlagFly         = 1 << iota
	FlagSwim        = 1 << iota
	FlagConveyor    = 1 << iota
	FlagClient      = 1 << iota
	FlagInWater     = 1 << iota
	FlagMonstar     = 1 << iota
	FlagGodMode     = 1 << iota
	FlagNoTarget    = 1 << iota
	FlagItem        = 1 << iota
	FlagOnGround    = 1 << iota
	FlagPartialJump = 1 << iota
	FlagWaterJump   = 1 << iota
	FlagJumpRelease = 1 << iota
)

const (
	MoveTypeNone = iota
	MoveTypeAngleNoClip
	MoveTypeAngleClip
	MoveTypeWalk
	MoveTypeStep
	MoveTypeFly
	MoveTypeToss
	MoveTypePush
	MoveTypeNoClip
	MoveTypeFlyMissle
	MoveTypeBounce
)

const (
	ItemShotgun         = 1 << iota
	ItemSuperShotgun    = 1 << iota
	ItemNailgun         = 1 << iota
	ItemSuperNailgun    = 1 << iota
	ItemGrenadeLauncher = 1 << iota
	ItemRocketLauncher  = 1 << iota
	ItemLightning       = 1 << iota
	ItemSuperLightning  = 1 << iota
	ItemShells          = 1 << iota
	ItemNails           = 1 << iota
	ItemRockets         = 1 << iota
	ItemCells           = 1 << iota
	ItemAxe             = 1 << iota
	ItemArmor1          = 1 << iota
	ItemArmor2          = 1 << iota
	ItemArmor3          = 1 << iota
	ItemSuperHealth     = 1 << iota
	ItemKey1            = 1 << iota
	ItemKey2            = 1 << iota
	ItemInvisibility    = 1 << iota
	ItemInvulnerability = 1 << iota
	ItemSuit            = 1 << iota
	ItemQuad            = 1 << iota
	_                   = 1 << iota // 23
	_                   = 1 << iota
	_                   = 1 << iota
	_                   = 1 << iota
	_                   = 1 << iota
	ItemSigil1          = 1 << iota
	ItemSigil2          = 1 << iota
	ItemSigil3          = 1 << iota
	ItemSigil4          = 1 << iota
)

//===========================================
// rogue changed and added defines
const (
/*
 RIT_SHELLS 128
 RIT_NAILS 256
 RIT_ROCKETS 512
 RIT_CELLS 1024
 RIT_AXE 2048
 RIT_LAVA_NAILGUN 4096
 RIT_LAVA_SUPER_NAILGUN 8192
 RIT_MULTI_GRENADE 16384
 RIT_MULTI_ROCKET 32768
 RIT_PLASMA_GUN 65536
 RIT_ARMOR1 8388608
 RIT_ARMOR2 16777216
 RIT_ARMOR3 33554432
 RIT_LAVA_NAILS 67108864
 RIT_PLASMA_AMMO 134217728
 RIT_MULTI_ROCKETS 268435456
 RIT_SHIELD 536870912
 RIT_ANTIGRAV 1073741824
 RIT_SUPERHEALTH 2147483648
*/
)

// MED 01/04/97 added hipnotic defines
//===========================================
// hipnotic added defines
const (
/*
 HIT_PROXIMITY_GUN_BIT 16
 HIT_MJOLNIR_BIT 7
 HIT_LASER_CANNON_BIT 23
 HIT_PROXIMITY_GUN (1 << HIT_PROXIMITY_GUN_BIT)
 HIT_MJOLNIR (1 << HIT_MJOLNIR_BIT)
 HIT_LASER_CANNON (1 << HIT_LASER_CANNON_BIT)
 HIT_WETSUIT (1 << (23 + 2))
 HIT_EMPATHY_SHIELDS (1 << (23 + 3))
*/
)

type Header struct {
	Version          int32
	CRC              int32
	OffsetStatements int32
	NumStatements    int32
	OffsetGlobalDefs int32
	NumGlobalDefs    int32
	OffsetFieldDefs  int32
	NumFieldDefs     int32
	OffsetFunctions  int32
	NumFunctions     int32
	OffsetStrings    int32
	NumStrings       int32
	OffsetGlobals    int32
	NumGlobals       int32
	EntityFields     int32
}

type Function struct {
	FirstStatement int32
	ParmStart      int32
	Locals         int32
	Profile        int32
	SName          int32
	SFile          int32
	NumParms       int32
	ParmSize       [MaxParms]byte // matches OffsetParm0-7
}

type Def struct {
	Type   uint16
	Offset uint16
	SName  int32
}

type Statement struct {
	Operator uint16
	A        int16
	B        int16
	C        int16
}

type GlobalVars struct {
	Pad               [28]int32
	Self              int32
	Other             int32
	World             int32
	Time              float32
	FrameTime         float32
	ForceRetouch      float32
	MapName           int32
	DeathMatch        float32
	Coop              float32
	TeamPlay          float32
	ServerFlags       float32
	TotalSecrets      float32
	TotalMonsters     float32
	FoundSecrets      float32
	KilledMonsters    float32
	Parm              [16]float32
	VForward          [3]float32
	VUp               [3]float32
	VRight            [3]float32
	TraceAllSolid     float32
	TraceStartSolid   float32
	TraceFraction     float32
	TraceEndPos       [3]float32
	TracePlaneNormal  [3]float32
	TracePlaneDist    float32
	TraceEnt          int32
	TraceInOpen       float32
	TraceInWater      float32
	MsgEntity         int32
	Main              int32
	StartFrame        int32
	PlayerPreThink    int32
	PlayerPostThink   int32
	ClientKill        int32
	ClientConnect     int32
	PutClientInServer int32
	ClientDisconnect  int32
	SetNewParms       int32
	SetChangeParms    int32
}

type EntVars struct {
	ModelIndex   float32
	AbsMin       [3]float32
	AbsMax       [3]float32
	LTime        float32
	MoveType     float32
	Solid        float32
	Origin       [3]float32
	OldOrigin    [3]float32
	Velocity     [3]float32
	Angles       [3]float32
	AVelocity    [3]float32
	PunchAngle   [3]float32
	ClassName    int32
	Model        int32
	Frame        float32
	Skin         float32
	Effects      float32
	Mins         [3]float32
	Maxs         [3]float32
	Size         [3]float32
	Touch        int32
	Use          int32
	Think        int32
	Blocked      int32
	NextThink    float32
	GroundEntity int32
	Health       float32
	Frags        float32
	Weapon       float32
	WeaponModel  int32
	WeaponFrame  float32
	CurrentAmmo  float32
	AmmoShells   float32
	AmmoNails    float32
	AmmoRockets  float32
	AmmoCells    float32
	Items        float32
	TakeDamage   float32
	Chain        int32
	DeadFlag     float32
	ViewOfs      [3]float32
	Button0      float32
	Button1      float32
	Button2      float32
	Impulse      float32
	FixAngle     float32
	VAngle       [3]float32
	IdealPitch   float32
	NetName      int32
	Enemy        int32
	Flags        float32
	ColorMap     float32
	Team         float32
	MaxHealth    float32
	TeleportTime float32
	ArmorType    float32
	ArmorValue   float32
	WaterLevel   float32
	WaterType    float32
	IdealYaw     float32
	YawSpeed     float32
	Aiment       int32
	GoalEntity   int32
	SpawnFlags   float32
	Target       int32
	TargetName   int32
	DmgTake      float32
	DmgSave      float32
	DmgInflictor int32
	Owner        int32
	Movedir      [3]float32
	Message      int32
	Sounds       float32
	Noise        int32
	Noise1       int32
	Noise2       int32
	Noise3       int32
}
