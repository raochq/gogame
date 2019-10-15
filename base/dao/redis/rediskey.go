package redis

import (
	"errors"
	"strconv"
)

type RedisType = int

const (
	RedisServerTypeZone  RedisType = 1
	RedisServerTypeCross RedisType = 2
)

//key
const (
	KeyTypeUser       = "user"
	KeyTypeRedisMutex = "rmutex"
)

// For Server
const (
	ServerList = "server:list"
)

// For TypeUser.
const (
	KeySpecLoc = "loc"
	KeyToken   = "token"
)

var (
	errUserUncached    = errors.New("user has not been cached")
	errNoSuchWorker    = errors.New("no such worker server")
	errNoWorker        = errors.New("no available worker")
	errWorkerExhausted = errors.New("all worker fully cached")
	errImpossible      = errors.New("why you get here, awesome")
	errUnknownMsg      = errors.New("unknown client msg")
	errSessionNoFound  = errors.New("session not found")
	errRedisMutex      = errors.New("Redis mutex wait timeout")
)

func GenKey(kt, ks string, id uint) string {
	return kt + ":" + strconv.FormatUint(uint64(id), 10) + ":" + ks
}
