package redis

import (
	"github.com/gomodule/redigo/redis"
	"go-ws/utils/zaplog"
	"sync"
	"time"
)

var (
	RedisPool sync.Map
)

type RedisOptions struct {
	Host           string `toml:"host"`
	Port           string `toml:"port"`
	Password       string `toml:"password"`
	MaxIdle        int    `toml:"maxIdle"`
	MaxOpen        int    `toml:"maxOpen"`
	ConnectTimeout int    `toml:"connect_timeout"`
	ReadTimeout    int    `toml:"read_timeout"`
	WriteTimeout   int    `toml:"write_timeout"`
	IdleTimeout    int    `toml:"idle_timeout"`
}

func NewRedisClient(index string, re RedisOptions) error {
	pool := &redis.Pool{
		MaxIdle:   re.MaxIdle,
		MaxActive: re.MaxOpen,
		Dial: func() (redis.Conn, error) {

			c, err := redis.Dial("tcp", re.Host+":"+re.Port,
				redis.DialPassword(re.Password),
				redis.DialConnectTimeout(time.Duration(re.ConnectTimeout)*time.Second),
				redis.DialReadTimeout(time.Duration(re.ReadTimeout)*time.Second),
				redis.DialWriteTimeout(time.Duration(re.WriteTimeout)*time.Second))

			if err != nil {
				zaplog.Info("redis connect fail")
				return nil, err
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		IdleTimeout: 300 * time.Second,
		Wait:        true,
	}

	conn := pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PING"); err != nil {
		zaplog.Info("redis ping fail")
		return err
	} else {
		RedisPool.Store(index, pool)
		return nil
	}
}


