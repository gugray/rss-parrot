package shared

import (
	"encoding/json"
	"github.com/tailscale/hujson"
	"log"
	"os"
	"time"
)

const (
	configVarName = "CONFIG"                  // If set, will load config.json from this path and not from devConfigPath
	devConfigPath = "../dev/config.dev.jsonc" // Path to config.json in development environment
)

type Config struct {
	LogFile     string    `json:"log_file"`
	LogLevel    string    `json:"log_level"`
	ServicePort uint      `json:"service_port"`
	Host        string    `json:"host"`
	Birb        *UserInfo `json:"birb"`
}

type UserInfo struct {
	User       string    `json:"user"`
	Name       string    `json:"name"`
	Summary    string    `json:"summary"`
	Published  time.Time `json:"published"`
	ProfilePic string    `json:"profile_pic"`
	HeaderPic  string    `json:"header_pic"`
	PubKey     string    `json:"pub_key"`
	PrivKey    string    `json:"priv_key"`
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
