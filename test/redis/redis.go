package redis

import (
	"errors"
	"strings"
	"sync"


	"github.com/BurntSushi/toml"
	"github.com/gomodule/redigo/redis"
)

const (
	defaultRedisTomlConfigPath = "/root/go-ws/config/redis.toml"
)

var mutex sync.Mutex

func getRedisOptions(index string) (options RedisOptions, err error) {
	var config map[string]interface{}
	_, err = toml.DecodeFile(defaultRedisTomlConfigPath, &config)
	if err != nil {
		return
	}
	var env string
	instanceSlice := strings.Split(index, ".")
	if len(instanceSlice) == 2 {
		index = instanceSlice[0]
		env = instanceSlice[1]
	} else {
		env = config["env"].(string)
	}
	if instanceConf, ok := config[index].(map[string]interface{}); ok {
		if envConf, ok := instanceConf[env].(map[string]interface{}); ok {
			options = RedisOptions{
				Host:           envConf["host"].(string),
				Port:           envConf["port"].(string),
				Password:       envConf["password"].(string),
				MaxIdle:        int(envConf["maxIdle"].(int64)),
				MaxOpen:        int(envConf["maxOpen"].(int64)),
				ConnectTimeout: int(envConf["connect_timeout"].(int64)),
				ReadTimeout:    int(envConf["read_timeout"].(int64)),
				WriteTimeout:   int(envConf["write_timeout"].(int64)),
				IdleTimeout:    int(envConf["idle_timeout"].(int64)),
			}
			return
		}
	}
	err = errors.New("redis options read failed, " + index)
	return
}

func NewRedis(instance string) *redis.Pool {
	if redisLoad, ok := RedisPool.Load(instance); ok {
		redisPool := redisLoad.(*redis.Pool)
		return redisPool
	}

	//加锁防并发初始化
	mutex.Lock()
	defer mutex.Unlock()

	options, err := getRedisOptions(instance)
	if err != nil {
		panic(err)
	}

	if err := NewRedisClient(instance, options); err != nil {
		panic(err)
	}

	poolLoad, _ := RedisPool.Load(instance)
	redisPool := poolLoad.(*redis.Pool)
	return redisPool
}