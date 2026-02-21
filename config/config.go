package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

var Cfg Config

var configFile = "config.yaml"

func init() {
	// 设置默认值
	Cfg = Config{
		RootDir:          ".",                      // 文件下载根目录
		UseSubDir:        false,                    // 是否使用子目录(按作者分类)
		Email:            "",                       // 用户名
		Password:         "",                       // 密码，用于刷新API授权令牌
		Authorization:    "",                       // API授权令牌
		ProxyUrl:         "",                       // 代理服务器地址
		ApiToken:         "",                       // daemon HTTP API token
		FilenameTemplate: "{{title}}-{{video_id}}", // output filename template
		ThreadNum:        3,                        // 下载线程数
		MaxRetry:         3,                        // 最大重试次数
	}

	// 尝试加载配置文件，如果文件不存在则使用默认值
	_ = LoadConfig(&Cfg)
}

type Config struct {
	RootDir          string `yaml:"rootDir"`
	UseSubDir        bool   `yaml:"useSubDir"`
	Email            string `yaml:"email"`
	Password         string `yaml:"password"`
	Authorization    string `yaml:"authorization"`
	ProxyUrl         string `yaml:"proxyUrl"`
	ApiToken         string `yaml:"apiToken"`
	FilenameTemplate string `yaml:"filenameTemplate"`
	ThreadNum        int    `yaml:"threadNum"`
	MaxRetry         int    `yaml:"maxRetry"`
}

func LoadConfig(cfg *Config, cfgfile ...string) error {
	if cfg == nil {
		return errors.New("config pointer cannot be nil")
	}

	file := "config.yaml"
	if len(cfgfile) > 0 {
		file = cfgfile[0]
		configFile = file
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

func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config pointer cannot be nil")
	}
	f, err := os.Create(configFile)
	if err != nil {
		return errors.New("failed to create config file: " + configFile + ". Error: " + err.Error())
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	encoder := yaml.NewEncoder(f)
	defer func(encoder *yaml.Encoder) {
		_ = encoder.Close()
	}(encoder)
	err = encoder.Encode(cfg)
	if err != nil {
		return errors.New("failed to encode config file: " + configFile + ". Error: " + err.Error())
	}
	return nil
}
