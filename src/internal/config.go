package internal

import (
	"encoding/json"
	"log"
	"os"
)

const (
	configVarName = "CONFIG"          // If set, will load config.json from this path and not from devConfigPath
	devConfigPath = "config.dev.json" // Path to config.json in development environment
)

type Config struct {
	LogFile     string `json:"log_file"`
	ServicePort uint   `json:"service_port"`
	BaseUrl     string `json:"base_url"`
}

func (cfg *Config) GetServicePort() uint {
	return cfg.ServicePort
}

func (cfg *Config) GetBaseUrl() string {
	return cfg.BaseUrl
}

func ProvideConfig() *Config {
	cfgPath := os.Getenv(configVarName)
	if len(cfgPath) == 0 {
		cfgPath = devConfigPath
	}
	cfgJson, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	var config Config
	if err := json.Unmarshal(cfgJson, &config); err != nil {
		log.Fatal(err)
	}
	return &config
}
