package shared

import (
	"encoding/json"
	"github.com/tailscale/hujson"
	"log"
	"os"
)

const (
	configVarName = "CONFIG"           // If set, will load config.json from this path and not from devConfigPath
	devConfigPath = "config.dev.jsonc" // Path to config.json in development environment
)

type Config struct {
	LogFile     string `json:"log_file"`
	LogLevel    string `json:"log_level"`
	ServicePort uint   `json:"service_port"`
	Host        string `json:"host"`
	BirbName    string `json:"birb_name"`
	BirbPubkey  string `json:"birb_pubkey"`
	BirbPrivkey string `json:"birb_privkey"`
}

func LoadConfig() *Config {

	// Where's our config file?
	cfgPath := os.Getenv(configVarName)
	if len(cfgPath) == 0 {
		cfgPath = devConfigPath
	}

	// Read file
	var err error
	var cfgJson []byte
	cfgJson, err = os.ReadFile(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	// JSONC => JSON
	cfgJson, err = standardizeJSON(cfgJson)
	if err != nil {
		log.Fatal(err)
	}

	// Parse
	var config Config
	if err := json.Unmarshal(cfgJson, &config); err != nil {
		log.Fatal(err)
	}
	return &config
}

func standardizeJSON(b []byte) ([]byte, error) {
	ast, err := hujson.Parse(b)
	if err != nil {
		return b, err
	}
	ast.Standardize()
	return ast.Pack(), nil
}
