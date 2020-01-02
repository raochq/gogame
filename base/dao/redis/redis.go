package redis

import (
	"github.com/gomodule/redigo/redis"
	"gogame/base/conf"
	"gogame/base/logger"
	"time"
)

type redisCache struct {
	pool      *redis.Pool
	crossPool *redis.Pool
}

var cache redisCache

func InitRedis() {
	config := conf.GameGateConf()
	redisPool := &redis.Pool{
		MaxIdle:     config.RedisMaxIdle,
		IdleTimeout: time.Duration(config.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", config.ZoneRedisAddr)
			if err != nil {
				return nil, err
			}
			if config.ZoneRedisAuth != "" {
				if _, err := c.Do("AUTH", config.ZoneRedisAuth); err != nil {
					c.Close()
					return nil, err
				}
			}
			if config.ZoneRedisIndex > 0 && config.ZoneRedisIndex < 16 {
				if _, err = c.Do("SELECT", config.ZoneRedisIndex); err != nil {
					logger.Error("c.Do('SELECT', %v) failed(%v)", config.ZoneRedisIndex, err)
					c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	cache.pool = redisPool

	crossRedisPool := &redis.Pool{
		MaxIdle:     config.RedisMaxIdle,
		IdleTimeout: time.Duration(config.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", config.CrossRedisAddr)
			if err != nil {
				return nil, err
			}
			if config.CrossRedisAuth != "" {
				if _, err := c.Do("AUTH", config.CrossRedisAuth); err != nil {
					c.Close()
					return nil, err
				}
			}
			if config.CrossRedisIndex > 0 && config.CrossRedisIndex < 16 {
				if _, err = c.Do("SELECT", config.CrossRedisIndex); err != nil {
					logger.Error("c.Do('SELECT', %v) failed(%v)", config.CrossRedisIndex, err)
					c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	cache.crossPool = crossRedisPool
}

func (rc *redisCache) Pool(redisType RedisType) *redis.Pool {
	switch redisType {
	case RedisServerTypeZone:
		return cache.pool
	case RedisServerTypeCross:
		return cache.crossPool
	default:
		return cache.pool
	}
}
func (rc *redisCache) userLoc(con redis.Conn, uid int64) (uint16, error) {
	loc, err := redis.Int(con.Do("GET", GenKey(KeyTypeUser, KeySpecLoc, uint(uid))))
	if err != nil {
		return 0, err
	}
	return uint16(loc), nil
}
func (rc *redisCache) removeUserLoc(con redis.Conn, uid int64, gsid uint16) error {
	lock := NewRedisMutex(con, uint(uid), KeySpecLoc)
	lock.Lock()
	reply, err := rc.userLoc(con, uid)
	if reply == gsid {
		_, err = con.Do("DEL", GenKey(KeyTypeUser, KeySpecLoc, uint(uid)))
		if err != nil {
			lock.UnLock()
			return err
		}
		logger.Debug("remove user loc %d %d", uid, gsid)
	}
	lock.UnLock()
	return nil
}

func (rc *redisCache) getAccountToken(accountID int64) (string, error) {
	conn := rc.Pool(RedisServerTypeZone).Get()
	defer conn.Close()
	rKey := GenKey(KeyTypeUser, KeyToken, uint(accountID))
	return redis.String(conn.Do("Get", rKey))
}

func (rc *redisCache) refreshServerList(redisType RedisType, redisKey string) map[string]string {
	conn := rc.Pool(redisType).Get()
	defer conn.Close()
	reply, err := redis.StringMap(conn.Do("HGETALL", redisKey))
	if err != nil {
		logger.Error("redis %v hgetall %v failed, %s", redisType, redisKey, err.Error())
		return nil
	}
	return reply
}

func GetAccountToken(accountID int64) (string, error) {
	return cache.getAccountToken(accountID)
}
func Pool(redisType RedisType) *redis.Pool {
	return cache.Pool(redisType)
}

func UserLoc(con redis.Conn, uid int64) (uint16, error) {
	return cache.userLoc(con, uid)
}

func RefreshGameSvr() map[string]string {
	return cache.refreshServerList(RedisServerTypeCross, GameServerListInRedis)
}

func RefreshIMSvr() map[string]string {
	return cache.refreshServerList(RedisServerTypeCross, IMListInRedis)
}

func RefreshZoneRouters() map[string]string {
	return cache.refreshServerList(RedisServerTypeZone, RouterServerListInRedis)
}
func RefreshCrossRouters() map[string]string {
	return cache.refreshServerList(RedisServerTypeCross, RouterServerListInRedis)
}
