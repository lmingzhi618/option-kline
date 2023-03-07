package common

import (
	"github.com/gomodule/redigo/redis"
	"time"
)

type RedisConfig struct {
	Host         string
	Port         string
	Auth         string
	DbName       string
	MaxIdle      int64
	MaxActive    int64
	IdleTimeout  int64
	Wait         bool
	ConnTimeout  int64
	WriteTimeout int64
	ReadTimeout  int64
}

var (
	RedisPool *redis.Pool
)

func initRedisPool() {
	RedisPool = &redis.Pool{
		MaxIdle:     int(REDISCONF.MaxIdle),
		MaxActive:   int(REDISCONF.MaxActive),
		IdleTimeout: time.Duration(REDISCONF.IdleTimeout) * time.Second,
		Wait:        REDISCONF.Wait,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp",
				REDISCONF.Host+":"+REDISCONF.Port,
				redis.DialConnectTimeout(time.Duration(REDISCONF.ConnTimeout)*time.Millisecond),
				redis.DialReadTimeout(time.Duration(REDISCONF.ReadTimeout)*time.Millisecond),
				redis.DialWriteTimeout(time.Duration(REDISCONF.WriteTimeout)*time.Millisecond),
			)
			if err != nil {
				return nil, err
			}
			//认证
			if len(REDISCONF.Auth) > 0 {
				conn.Do("AUTH", REDISCONF.Auth)
			}
			// 选择db
			if len(REDISCONF.DbName) > 0 {
				conn.Do("SELECT", REDISCONF.DbName)
			} else {
				conn.Do("SELECT", 0)
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func RedisGetData(args string) (ret string, err error) {
	c := RedisPool.Get()
	defer c.Close()
	ret, err = redis.String(c.Do("GET", args))
	return
}

func RedisMGetData(args redis.Args) (ret []string, err error) {
	c := RedisPool.Get()
	defer c.Close()
	ret, err = redis.Strings(c.Do("MGET", args...))
	return
}

func RedisHGetData(hkey, key string) (ret string, err error) {
	c := RedisPool.Get()
	defer c.Close()
	ret, err = redis.String(c.Do("HGET", hkey, key))
	return
}
