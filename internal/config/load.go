package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	Prefix = "TUDOU"
)

func NewConfig(path string) (*Config, error) {
	envConf := os.Getenv(fmt.Sprintf("%s_CONF", Prefix))
	if envConf == "" {
		envConf = path
	}
	slog.Info("load conf file", "file", envConf)
	return getConfig(envConf)
}

func getConfig(configFile string) (*Config, error) {
	if configFile == "" {
		return nil, errors.New("config file is empty")
	}

	v := viper.New()
	// set > env > config file > default
	v.SetConfigFile(configFile)

	// default values
	v.SetDefault("app.env", "dev")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)

	// env override: TUDOU_DATA_DB_USER_DSN -> data.db.user.dsn
	v.SetEnvPrefix(strings.ToUpper(Prefix))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	if err := ensureJWTSecret(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func ensureJWTSecret(cfg *Config) error {
	if strings.TrimSpace(cfg.Security.JWT.Secret) != "" {
		return nil
	}

	// When the config leaves JWT secret empty, reuse a persisted runtime secret
	// first so existing sessions survive restarts.
	secretPath, err := runtimeJWTSecretPath(cfg)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(secretPath)
	switch {
	case err == nil:
		cfg.Security.JWT.Secret = strings.TrimSpace(string(data))
		if cfg.Security.JWT.Secret == "" {
			return fmt.Errorf("jwt secret file is empty: %s", secretPath)
		}
		return nil
	case !os.IsNotExist(err):
		return fmt.Errorf("read jwt secret file failed: %w", err)
	}

	// Generate once and persist it; do not fall back to an in-memory-only secret.
	secret, err := generateJWTSecret()
	if err != nil {
		return fmt.Errorf("generate jwt secret failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		return fmt.Errorf("create jwt secret dir failed: %w", err)
	}
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		return fmt.Errorf("persist jwt secret failed: %w", err)
	}
	cfg.Security.JWT.Secret = secret
	slog.Warn("security.jwt.secret is empty, generated and persisted runtime secret", "path", secretPath)
	return nil
}

func runtimeJWTSecretPath(cfg *Config) (string, error) {
	if strings.TrimSpace(cfg.Security.JWT.SecretFile) != "" {
		return filepath.Abs(cfg.Security.JWT.SecretFile)
	}

	// Keep the generated secret under storage/ so compose volume mounts persist it.
	return filepath.Abs(filepath.Join("storage", "runtime", "jwt_secret"))
}

func generateJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
