package common

// TransType
const (
	TransType_None      = -1 //none type
	TransType_ByKey     = 1  //protocol to someone by key
	TransType_Broadcast = 2  //broadcast protocol
	TransType_Group     = 3  //spec group protocol
)

// SSMsg CMD
const (
	SSCMD_READY = iota + 1
	SSCMD_SYNC
	SSCMD_MSG
	SSCMD_HEARTBEAT
	SSCMD_DISCONNECT
)

// dest/src entity type
const (
	EntityType_Client       = 1  // Client
	EntityType_Portal       = 2  // Portal server
	EntityType_GameSvr      = 3  // Game server
	EntityType_SnsServer    = 4  // Sns server
	EntityType_BattleServer = 5  // Battle server
	EntityType_TeamSvr      = 6  // Team server
	EntityType_LoginSvr     = 7  // Login server
	EntityType_Router       = 8  // Router server
	EntityType_IM           = 9  // IM server
	ENtityType_Recharge     = 10 // Recharge server
	EntityType_GMClient     = 30 // GM client
	EntityType_VersionSvr   = 40 // Login server
)

const (
	INVALID_GAMESVR_ID = 0 // invalid gameserver id
	INVALID_PORTAL_ID  = 0 // invalid portal id
	INVALID_SESS_ID    = 0 // invalid session id
	INVALID_PK_POS_ID  = 0 // invalid pk pos id
	INVALID_ID         = 0 // invalid id
)
