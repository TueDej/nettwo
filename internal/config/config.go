package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"filippo.io/age"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

type yamlParser struct{}

func (yamlParser) Unmarshal(b []byte) (map[string]any, error) {
	var m map[string]any
	err := yaml.Unmarshal(b, &m)
	return m, err
}

func (yamlParser) Marshal(m map[string]any) ([]byte, error) {
	return yaml.Marshal(m)
}

var yamlParserInstance = yamlParser{}

var (
	configDirName       = "nettwo"
	configFileName      = "config.yaml"
	credentialsFileName = "credentials.age"
	identityFileName    = "identity.age"
)

// testConfigDir is used to override the config directory for testing.
// When set, GetConfigDir will return this path instead of the default.
var testConfigDir string
var testConfigDirMu sync.Mutex
var credentialsMu sync.Mutex
var identityMu sync.Mutex

type Config struct {
	BaseURL            string `yaml:"base_url" koanf:"base_url"`
	ConnectTimeout     int    `yaml:"connect_timeout" koanf:"connect_timeout"`
	RequestTimeout     int    `yaml:"request_timeout" koanf:"request_timeout"`
	MaxRetries         int    `yaml:"max_retries" koanf:"max_retries"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" koanf:"insecure_skip_verify"`
}

type Credentials struct {
	Accounts []Account `json:"accounts"`
}

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var defaultConfig = Config{
	BaseURL:            "https://net2.sharif.edu",
	ConnectTimeout:     10,
	RequestTimeout:     30,
	MaxRetries:         3,
	InsecureSkipVerify: false,
}

func GetConfigDir() (string, error) {
	testConfigDirMu.Lock()
	if testConfigDir != "" {
		dir := testConfigDir
		testConfigDirMu.Unlock()
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", fmt.Errorf("failed to create config dir: %w", err)
		}
		return dir, nil
	}
	testConfigDirMu.Unlock()

	baseConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	appConfigDir := filepath.Join(baseConfigDir, configDirName)
	if err := os.MkdirAll(appConfigDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	return appConfigDir, nil
}

func configFilePath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func credentialsFilePath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, credentialsFileName), nil
}

func identityFilePath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, identityFileName), nil
}

func Load() (*Config, error) {
	k := koanf.New(".")

	path, err := configFilePath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), yamlParserInstance); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect config: %w", err)
	}

	if err := k.Load(env.Provider("NETTWO_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "NETTWO_"))
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}

	cfg := defaultConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validate(cfg *Config) error {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return fmt.Errorf("base_url must be a valid http(s) URL")
	}
	if cfg.ConnectTimeout <= 0 {
		return fmt.Errorf("connect_timeout must be greater than zero")
	}
	if cfg.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be greater than zero")
	}
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	return nil
}

func Save(cfg *Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return writeAtomic(path, data, 0600)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".nettwo-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func loadIdentity() (*age.X25519Identity, error) {
	identityMu.Lock()
	defer identityMu.Unlock()
	return loadIdentityUnlocked()
}

func loadIdentityUnlocked() (*age.X25519Identity, error) {
	path, err := identityFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return generateIdentity(path)
		}
		return nil, err
	}

	identity, err := age.ParseX25519Identity(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse identity: %w", err)
	}
	return identity, nil
}

func generateIdentity(path string) (*age.X25519Identity, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("failed to generate identity: %w", err)
	}

	// The identity.String() returns the armored key
	if err := writeAtomic(path, []byte(identity.String()), 0600); err != nil {
		return nil, fmt.Errorf("failed to write identity: %w", err)
	}
	return identity, nil
}

func LoadCredentials() (*Credentials, error) {
	credsPath, err := credentialsFilePath()
	if err != nil {
		return nil, err
	}

	identity, err := loadIdentity()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Credentials{Accounts: []Account{}}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return &Credentials{Accounts: []Account{}}, nil
	}

	decryptor, err := age.Decrypt(bytes.NewReader(data), identity)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	decrypted, err := io.ReadAll(decryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}
	return &creds, nil
}

func SaveCredentials(creds *Credentials) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	return saveCredentials(creds)
}

func saveCredentials(creds *Credentials) error {
	credsPath, err := credentialsFilePath()
	if err != nil {
		return err
	}

	identity, err := loadIdentity()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	recipient := identity.Recipient()
	var buf bytes.Buffer
	encryptor, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}
	if _, err := encryptor.Write(data); err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}
	if err := encryptor.Close(); err != nil {
		return fmt.Errorf("failed to close encryptor: %w", err)
	}

	return writeAtomic(credsPath, buf.Bytes(), 0600)
}

func AddAccount(username, password string) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	for i, acc := range creds.Accounts {
		if acc.Username == username {
			creds.Accounts[i].Password = password
			return saveCredentials(creds)
		}
	}

	creds.Accounts = append(creds.Accounts, Account{Username: username, Password: password})
	return saveCredentials(creds)
}

func DeleteAccount(username string) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	found := false
	newAccounts := make([]Account, 0, len(creds.Accounts))
	for _, acc := range creds.Accounts {
		if acc.Username != username {
			newAccounts = append(newAccounts, acc)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("account %q not found", username)
	}

	creds.Accounts = newAccounts
	return saveCredentials(creds)
}

func GetAccount(username string) (*Account, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	for _, acc := range creds.Accounts {
		if acc.Username == username {
			return &acc, nil
		}
	}
	return nil, fmt.Errorf("account %q not found", username)
}

func ListAccounts() ([]string, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	usernames := make([]string, len(creds.Accounts))
	for i, acc := range creds.Accounts {
		usernames[i] = acc.Username
	}
	return usernames, nil
}

func ValidateCredentials(username, password string) (*Account, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	for _, acc := range creds.Accounts {
		if acc.Username == username {
			if acc.Password != password {
				return nil, fmt.Errorf("invalid password")
			}
			return &acc, nil
		}
	}
	return nil, fmt.Errorf("account %q not found", username)
}
