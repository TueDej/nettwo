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

	"nettwo/internal/ui"
	"nettwo/internal/util"
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
	configDirName        = "nettwo"
	configFileName       = "config.yaml"
	credentialsFileName  = "credentials.age"
	identityFileName     = "identity.age"
	lastUsernameFileName = "last_username"
)

// testConfigDir is used to override the config directory for testing.
// When set, GetConfigDir will return this path instead of the default.
var testConfigDir string
var testConfigDirMu sync.Mutex
var credentialsMu sync.Mutex

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

func GetCacheDir() (string, error) {
	testConfigDirMu.Lock()
	if testConfigDir != "" {
		dir := testConfigDir
		testConfigDirMu.Unlock()
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", fmt.Errorf("failed to create cache dir: %w", err)
		}
		return dir, nil
	}
	testConfigDirMu.Unlock()

	baseCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache dir: %w", err)
	}
	appCacheDir := filepath.Join(baseCacheDir, configDirName)
	if err := os.MkdirAll(appCacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}
	return appCacheDir, nil
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
	if parsed.Path != "" && parsed.Path != "/" {
		return fmt.Errorf("base_url must not contain a path component")
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

	return util.WriteAtomic(path, data, 0600)
}

func loadIdentity() (*age.X25519Identity, error) {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
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
		// Backup corrupt identity file and generate new one
		backupPath := path + ".corrupt"
		_ = os.Rename(path, backupPath)
		ui.Warn(fmt.Sprintf("Corrupt identity file backed up to %s, generating new identity", backupPath))
		return generateIdentity(path)
	}
	return identity, nil
}

func generateIdentity(path string) (*age.X25519Identity, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("failed to generate identity: %w", err)
	}

	// The identity.String() returns the armored key
	if err := util.WriteAtomic(path, []byte(identity.String()), 0600); err != nil {
		return nil, fmt.Errorf("failed to write identity: %w", err)
	}
	return identity, nil
}

func LoadCredentials() (*Credentials, error) {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	return loadCredentialsUnlocked()
}

func loadCredentialsUnlocked() (*Credentials, error) {
	credsPath, err := credentialsFilePath()
	if err != nil {
		return nil, err
	}

	identity, err := loadIdentityUnlocked()
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
	return saveCredentialsUnlocked(creds)
}

func saveCredentials(creds *Credentials) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	return saveCredentialsUnlocked(creds)
}

func saveCredentialsUnlocked(creds *Credentials) error {
	credsPath, err := credentialsFilePath()
	if err != nil {
		return err
	}

	identity, err := loadIdentityUnlocked()
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

	return util.WriteAtomic(credsPath, buf.Bytes(), 0600)
}

func AccountExists(username string) (bool, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return false, err
	}
	for _, acc := range creds.Accounts {
		if acc.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func AddAccount(username, password string) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	creds, err := loadCredentialsUnlocked()
	if err != nil {
		return err
	}

	for i, acc := range creds.Accounts {
		if acc.Username == username {
			creds.Accounts[i].Password = password
			return saveCredentialsUnlocked(creds)
		}
	}

	creds.Accounts = append(creds.Accounts, Account{Username: username, Password: password})
	return saveCredentialsUnlocked(creds)
}

func DeleteAccount(username string) error {
	credentialsMu.Lock()
	defer credentialsMu.Unlock()
	creds, err := loadCredentialsUnlocked()
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
	return saveCredentialsUnlocked(creds)
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

func lastUsernameFilePath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, lastUsernameFileName), nil
}

func legacyLastUsernameFilePath() (string, error) {
	dir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, lastUsernameFileName), nil
}

func SetLastUsername(username string) error {
	path, err := lastUsernameFilePath()
	if err != nil {
		return err
	}
	return util.WriteAtomic(path, []byte(username), 0600)
}

func GetLastUsername() (string, error) {
	path, err := lastUsernameFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to the legacy cache-dir location.
			legacyPath, legacyErr := legacyLastUsernameFilePath()
			if legacyErr == nil && legacyPath != path {
				if legacyData, readErr := os.ReadFile(legacyPath); readErr == nil {
					return strings.TrimSpace(string(legacyData)), nil
				}
			}
			return "", fmt.Errorf("no last connected user found")
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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
