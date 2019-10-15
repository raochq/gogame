package redis

import (
	"github.com/garyburd/redigo/redis"
	"gogame/base/logger"
	"math/rand"
	"time"
)

type RedisMutex struct {
	con       redis.Conn
	LockerKey uint
	Spec      string
}

const (
	RedisMutexTTL        = 1 * 1800 // keep track of latest half hours
	RedisMutexRetryTimes = 5
)

func NewRedisMutex(con redis.Conn, lockerKey uint, spec string) *RedisMutex {
	return &RedisMutex{con, lockerKey, spec}
}

func (r *RedisMutex) wrapRedisInt64Clear(key string) (err error) {
	err = nil
	for i := 0; i < RedisMutexRetryTimes; i++ {
		_, err = r.con.Do("DEL", key)
		if err != nil {
			logger.Error("can't clear redis mutex for key %s with error %v with retry Cnt %v \n", key, err, i)
			time.Sleep(5 * time.Millisecond)
		} else {
			return
		}
	}
	return
}

func (r *RedisMutex) wrapRedisInt64Get(key string) (redisM int64, err error) {

	err = nil
	redisM = 0
	for i := 0; i < RedisMutexRetryTimes; i++ {
		redisM, err = redis.Int64(r.con.Do("GET", key))
		if err != nil {
			logger.Error("can't get redis mutex for key %s with error %v with retry Cnt %v \n", key, err, i)
			time.Sleep(5 * time.Millisecond)
		} else {
			return
		}
	}
	return
}

func (r *RedisMutex) wrapRedisInt64Decr(key string) (redisM int64, err error) {

	err = nil
	redisM = 0
	for i := 0; i < RedisMutexRetryTimes; i++ {
		redisM, err = redis.Int64(r.con.Do("DECR", key))
		if err != nil {
			logger.Error("can't decr redis mutex for key %s with error %v with retry Cnt %v \n", key, err, i)
			time.Sleep(5 * time.Millisecond)
		} else {
			return
		}
	}
	return
}

func (r *RedisMutex) wrapRedisInt64Incr(key string) (redisM int64, err error) {

	err = nil
	redisM = 0
	for i := 0; i < RedisMutexRetryTimes; i++ {
		redisM, err = redis.Int64(r.con.Do("INCR", key))
		if err != nil {
			logger.Error("can't incr redis mutex for key %s with error %v with retry Cnt %v \n", key, err, i)
			time.Sleep(1 * time.Millisecond)
		} else {
			r.con.Do("EXPIRE", key, RedisMutexTTL) // safety guard
			return
		}
	}
	return
}

func (r *RedisMutex) Lock() error {

	redisMutexKey := GenKey(KeyTypeRedisMutex, r.Spec, r.LockerKey)
	var err error = nil
	var redisM int64 = 0
	for i := 0; i < 150; i++ { // lock at most 15-30s
		redisM, err = r.wrapRedisInt64Incr(redisMutexKey)
		if err != nil {
			logger.Error("WrapRedisInt64Incr(%v) for %#v failed(%v) \n", redisMutexKey, r, err)
			return err
		} else {
			if redisM >= 2 { // has competition
				//log.Println("redisM %v >= 2, wait", redisM)
				_, err := r.wrapRedisInt64Decr(redisMutexKey)
				if err != nil {
					logger.Error("WrapRedisInt64Decr(%v) for %#v failed(%v) \n", redisMutexKey, r, err)
					return err
				}
			} else {
				//log.Printf("[Debug] Redis Mutext %#v got the lock, %v\n", r, redisM)
				return nil
			}
			time.Sleep(time.Duration(100+rand.Intn(100)) * time.Millisecond)
		}
	}

	logger.Error("[Debug] Redis Mutex%#v Lock, %v, timed out, clear\n", r, redisM)
	r.wrapRedisInt64Clear(redisMutexKey)
	err = errRedisMutex
	return err
}

func (r *RedisMutex) UnLock() error {

	redisMutexKey := GenKey(KeyTypeRedisMutex, r.Spec, r.LockerKey)
	var err error = nil
	var redisM int64 = 0

	redisM, err = r.wrapRedisInt64Decr(redisMutexKey)
	if err != nil {
		logger.Error("WrapRedisInt64Decr(%v) for %#v failed(%v) \n", redisMutexKey, r, err)
		return err
	}
	if redisM < 0 {
		r.wrapRedisInt64Clear(redisMutexKey)
	}
	// err = r.wrapRedisInt64Clear(redisMutexKey)
	// if err != nil {
	// 	log.Printf("wrapRedisInt64Clear(%v) for %#v failed(%v) \n", redisMutexKey, r, err)
	// 	return err
	// }

	//log.Printf("[Debug] Redis Mutex%#v Unluck, %v\n", r, redisM)
	return nil

}

func (r *RedisMutex) Validate() bool {

	redisMutexKey := GenKey(KeyTypeRedisMutex, r.Spec, r.LockerKey)

	val, err := r.wrapRedisInt64Get(redisMutexKey)
	if val == 0 && err == nil {
		return true
	}
	return false
}
