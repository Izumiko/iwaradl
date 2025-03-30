package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

var Cfg Config

func init() {
	err := LoadConfig(&Cfg)
	if err != nil {
		panic(err)
	}
}

type Config struct {
	RootDir       string `yaml:"rootDir"`
	UseSubDir     bool   `yaml:"useSubDir"`
	Authorization string `yaml:"authorization"`
	ProxyUrl      string `yaml:"proxyUrl"`
	ThreadNum     int    `yaml:"threadNum"`
	MaxRetry      int    `yaml:"maxRetry"`
}

func LoadConfig(cfg *Config, cfgfile ...string) error {
	if cfg == nil {
		return errors.New("config pointer cannot be nil")
	}

	var file string
	if len(cfgfile) == 0 {
		file = "config.yaml"
	} else {
		file = cfgfile[0]
	}
	f, err := os.Open(file)
	if err != nil {
		return errors.New("failed to open config file: " + file + ". Error: " + err.Error())
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return errors.New("failed to decode config file: " + file + ". Error: " + err.Error())
	}

	return nil
}
