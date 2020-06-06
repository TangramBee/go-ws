package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config define
type Config struct {
	Env string `toml:"env" `
	App appConfig
	Log logConfig
}

// AppConfig struct
type appConfig struct {
	Name           string
	Bind           string `toml:"bind" `
	Debug          int    `toml:"debug" `
}

type logConfig struct {
	Path      string
	AccessLog string `toml:"access_log"`
	RunLog    string `toml:"run_log"`
}

// Settings is app config
var Settings *Config

func Init(path string) (*Config, error) {
	if path == "" {
		path = "./config/config.toml"
	}
	Settings = &Config{}
	filePath, err := filepath.Abs(path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// try another  path
		filePath, err = filepath.Abs("../config/config.toml")
	}

	if err != nil {
		panic(err)
	}
	if _, err := toml.DecodeFile(filePath, &Settings); err != nil {
		panic(err)
	}
	return Settings, nil
}
