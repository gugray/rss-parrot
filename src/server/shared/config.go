package shared

import (
	"encoding/json"
	"github.com/tailscale/hujson"
	"log"
	"os"
	"time"
)

const (
	configVarName  = "CONFIG"                      // If set, will load config.json from this path and not from devConfigPath
	secretsVarName = "SECRETS"                     // If set, will load secrets.json from this path and not from devSecretsPath
	devConfigPath  = "../../dev/config.dev.jsonc"  // Path to config.json in development environment
	devSecretsPath = "../../dev/secrets.dev.jsonc" // Path to config.json in development environment
)

type Config struct {
	Secrets            Secrets        `json:"-"`
	LogFile            string         `json:"log_file"`
	LogLevel           string         `json:"log_level"`
	ServicePort        uint           `json:"service_port"`
	Host               string         `json:"host"`
	DbFile             string         `json:"db_file"`
	CachePageTemplates bool           `json:"cache_page_templates"`
	UpdateSchedule     UpdateSchedule `json:"update_schedule"`
	FallbackProfilePic string         `json:"fallback_profile_pic"`
	Birb               *UserInfo      `json:"birb"`
}

type UpdateSchedule struct {
	Day    int `json:"day"`
	Week   int `json:"week"`
	Weeks4 int `json:"weeks4"`
	Older  int `json:"older"`
}

type UserInfo struct {
	User                    string    `json:"user"`
	Published               time.Time `json:"published"`
	ManuallyApprovesFollows bool      `json:"manually_approves_follows"`
	ProfilePic              string    `json:"profile_pic"`
	HeaderPic               string    `json:"header_pic"`
	PubKey                  string    `json:"pub_key"`
	PrivKey                 string    `json:"priv_key"`
}

type Secrets struct {
	BirdPrivKeyPass string   `json:"birb_privkey_passphrase"`
	ApiKeys         []string `json:"api_keys"`
}

func LoadConfig() *Config {

	// Where are our config and secrets files?
	cfgPath := os.Getenv(configVarName)
	if len(cfgPath) == 0 {
		cfgPath = devConfigPath
	}
	secretsPath := os.Getenv(secretsVarName)
	if len(secretsPath) == 0 {
		secretsPath = devSecretsPath
	}

	// Read config file
	var config Config
	mustDeserializeFile(cfgPath, &config)
	// Read secrets member from secrets file
	mustDeserializeFile(secretsPath, &config.Secrets)
	return &config
}

func mustDeserializeFile[T any](fileName string, obj *T) {
	var err error
	var cfgJson []byte
	cfgJson, err = os.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}
	// JSONC => JSON
	cfgJson, err = standardizeJSON(cfgJson)
	if err != nil {
		log.Fatal(err)
	}
	// Parse
	if err := json.Unmarshal(cfgJson, obj); err != nil {
		log.Fatal(err)
	}
}

func standardizeJSON(b []byte) ([]byte, error) {
	ast, err := hujson.Parse(b)
	if err != nil {
		return b, err
	}
	ast.Standardize()
	return ast.Pack(), nil
}
