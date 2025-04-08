package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

var Cfg Config

func init() {
	// 设置默认值
	Cfg = Config{
		RootDir:       ".",
		UseSubDir:     false,
		Authorization: "",
		ProxyUrl:      "",
		ThreadNum:     3,
		MaxRetry:      3,
	}

	// 尝试加载配置文件，如果文件不存在则使用默认值
	_ = LoadConfig(&Cfg)
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
		// 任何错误都使用默认配置
		return nil
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
