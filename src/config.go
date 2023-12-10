package main

import (
	"encoding/json"
	"log"
	"os"
)

const (
	configVarName = "CONFIG"          // If set, will load config.json from this path and not from devConfigPath
	devConfigPath = "config.dev.json" // Path to config.json in development environment
)

type config struct {
	LogFile      string `json:"log_file"`
	ServicePort  uint   `json:"service_port"`
	InstanceName string `json:"instance_name"`
	BirbName     string `json:"birb_name"`
	BirbPubkey   string `json:"birb_pubkey"`
	BirbPrivkey  string `json:"birb_privkey"`
}

func (cfg *config) GetServicePort() uint {
	return cfg.ServicePort
}

func (cfg *config) GetInstanceName() string {
	return cfg.InstanceName
}

func (cfg *config) GetBirbName() string {
	return cfg.BirbName
}

func (cfg *config) GetBirbPubkey() string {
	return cfg.BirbPubkey
}

func provideConfig() *config {
	cfgPath := os.Getenv(configVarName)
	if len(cfgPath) == 0 {
		cfgPath = devConfigPath
	}
	cfgJson, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	var config config
	if err := json.Unmarshal(cfgJson, &config); err != nil {
		log.Fatal(err)
	}
	return &config
}
