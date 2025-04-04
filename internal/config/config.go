package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Telegram struct {
	Token             string `yaml:"token"`
	AstrologerChannel string `yaml:"astrologer_channel"`
}

type Logger struct {
	Level string `yaml:"level"`
	Sink  string `yaml:"sink"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type AppConfig struct {
	Logger   Logger   `yaml:"log"`
	Telegram Telegram `yaml:"telegram"`
	Database Database `yaml:"database"`
}

func NewConfig(path string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var appConfig AppConfig
	if err := yaml.Unmarshal(data, &appConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &appConfig, nil
}
