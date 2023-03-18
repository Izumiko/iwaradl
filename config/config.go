package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

var Cfg Config

func init() {
	LoadConfig(&Cfg)
}

type Config struct {
	RootDir       string `yaml:"rootDir"`
	UseSubDir     bool   `yaml:"useSubDir"`
	Authorization string `yaml:"authorization"`
	ProxyUrl      string `yaml:"proxyUrl"`
	ThreadNum     int    `yaml:"threadNum"`
	MaxRetry      int    `yaml:"maxRetry"`
}

func LoadConfig(cfg *Config, cfgfile ...string) {
	var file string
	if len(cfgfile) == 0 {
		file = "config.yaml"
	} else {
		file = cfgfile[0]
	}
	f, err := os.Open(file)
	if err != nil {
		errors.New(file + " not found")
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		errors.New(file + " format error")
	}
}
