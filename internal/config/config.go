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

type API struct {
	ButtonServerAddr string `yaml:"button_server_addr"`
	PublicURL        string `yaml:"public_url"`
	PDFStoragePath   string `yaml:"pdf_storage_path"`
}

type GRPC struct {
	ServerAddr string `yaml:"server_addr"`
	Timeout    int    `yaml:"timeout"`
}

type Reminder struct {
	CheckPeriod  int `yaml:"check_period"`
	ReminderTime int `yaml:"reminder_time"`
}

type AppConfig struct {
	Logger   Logger   `yaml:"log"`
	Telegram Telegram `yaml:"telegram"`
	Database Database `yaml:"database"`
	API      API      `yaml:"api"`
	GRPC     GRPC     `yaml:"grpc"`
	Reminder Reminder `yaml:"reminder"`
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
