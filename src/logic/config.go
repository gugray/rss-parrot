package logic

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
	LogFile      string `json:"log_file"`
	ServicePort  uint   `json:"service_port"`
	InstanceName string `json:"instance_name"`
	BirbName     string `json:"birb_name"`
	BirbPubkey   string `json:"birb_pubkey"`
	BirbPrivkey  string `json:"birb_privkey"`
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
