package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
)

var (
	Pools sync.Map
	mutex sync.Mutex
)

// default database toml config path
const (
	DefaultDatabaseTomlConfigPath = "./config/database.toml"
	DefaultDriver                 = "mysql"
	DefaultTimeout                = 1
	DefaultMaxIdle                = 10
	DefaultMaxOpen                = 1000
	DefaultMaxLifetime            = 300
)

// database config
type DBConfig struct {
	Driver      string
	Host        string
	Port        string
	Database    string
	Username    string
	Password    string
	Timeout     int64
	MaxIdle     int64
	MaxOpen     int64
	MaxLifetime int64
}

func getDatabaseConfig(instance string) (*DBConfig, error) {
	// get real instance
	var config map[string]interface{}
	_, err := toml.DecodeFile(DefaultDatabaseTomlConfigPath, &config)
	if err != nil {
		logger.Logger.Info("decode database toml file failed", zap.String("error", err.Error()))
		return nil, err
	}

	var env string
	instanceSlice := strings.Split(instance, ".")
	if len(instanceSlice) == 2 {
		instance = instanceSlice[0]
		env = instanceSlice[1]
	} else {
		env = config["env"].(string)
	}
	if instanceConf, ok := config[instance].(map[string]interface{}); ok {
		if envConf, ok := instanceConf[env].(map[string]interface{}); ok {
			c := &DBConfig{
				Driver:      DefaultDriver,
				Host:        envConf["host"].(string),
				Port:        envConf["port"].(string),
				Database:    envConf["database"].(string),
				Username:    envConf["username"].(string),
				Password:    envConf["password"].(string),
				Timeout:     DefaultTimeout,
				MaxIdle:     DefaultMaxIdle,
				MaxOpen:     DefaultMaxOpen,
				MaxLifetime: DefaultMaxLifetime,
			}
			if driver, ok := envConf["driver"].(string); ok {
				c.Driver = driver
			}
			if timeout, ok := envConf["timeout"].(int64); ok {
				c.Timeout = timeout
			}
			if maxIdle, ok := envConf["maxIdle"].(int64); ok {
				c.MaxIdle = maxIdle
			}
			if maxOpen, ok := envConf["maxOpen"].(int64); ok {
				c.MaxOpen = maxOpen
			}
			if maxLifetime, ok := envConf["maxLifetime"].(int64); ok {
				c.MaxLifetime = maxLifetime
			}
			return c, nil
		} else {
			return nil, errors.New("invalid database instance " + instance)
		}
	} else {
		logger.Logger.Warn("invalid database instance " + instance)
		return nil, errors.New("invalid database instance " + instance)
	}
}

func (c *DBConfig) GetDSN() string {
	temp := "%s:%s@tcp(%s:%s)/%s?charset=utf8&loc=%s&parseTime=true&timeout=5s&readTimeout=%ds"
	dsn := fmt.Sprintf(temp, c.Username, c.Password,
		c.Host, c.Port, c.Database, url.QueryEscape("Asia/Shanghai"), c.Timeout)
	return dsn
}

// 创建db连接
func newDb(index string, conf *DBConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", conf.GetDSN())
	if err != nil {
		logger.Logger.Warn("db open failed", zap.Error(err))
		return nil, err
	}

	db.SetMaxIdleConns(int(conf.MaxIdle))
	db.SetMaxOpenConns(int(conf.MaxOpen))
	db.SetConnMaxLifetime(time.Duration(conf.MaxLifetime) * time.Second)

	if err := db.Ping(); err != nil {
		err = fmt.Errorf("Failed to ping mysql: %s", err)
		return nil, err
	}

	return db, nil
}

// NewMySQL new mysql
func NewMySQL(index string) (*sql.DB, error) {
	if poolLoad, ok := Pools.Load(index); ok {
		db := poolLoad.(*sql.DB)
		return db, nil
	}

	//加锁防并发初始化
	mutex.Lock()
	defer mutex.Unlock()
	//锁后再判断一次，只有第一次获取锁的会做初始化。
	if poolLoad, ok := Pools.Load(index); ok {
		db := poolLoad.(*sql.DB)
		return db, nil
	}

	conf, err := getDatabaseConfig(index)
	if err != nil {
		logger.Logger.Warn("get config error", zap.String("error", err.Error()))
		return nil, err
	}

	db, err := newDb(index, conf)
	if err != nil {
		panic(err)
	}

	Pools.Store(index, db)
	return db, nil
}


