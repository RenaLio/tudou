package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigKeepsConfiguredJWTSecret(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	configFile := writeTestConfig(t, filepath.Join(root, "config"), "configured-secret", "")

	cfg, err := getConfig(configFile)
	if err != nil {
		t.Fatalf("getConfig returned error: %v", err)
	}

	if cfg.Security.JWT.Secret != "configured-secret" {
		t.Fatalf("expected configured secret to be kept, got %q", cfg.Security.JWT.Secret)
	}

	secretFile, err := runtimeJWTSecretPath(cfg)
	if err != nil {
		t.Fatalf("runtimeJWTSecretPath returned error: %v", err)
	}
	if _, err := os.Stat(secretFile); !os.IsNotExist(err) {
		t.Fatalf("expected no runtime secret file, got err=%v", err)
	}
}

func TestGetConfigGeneratesAndPersistsJWTSecretWhenEmpty(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	configFile := writeTestConfig(t, filepath.Join(root, "config"), "", "")

	cfg, err := getConfig(configFile)
	if err != nil {
		t.Fatalf("getConfig returned error: %v", err)
	}

	if strings.TrimSpace(cfg.Security.JWT.Secret) == "" {
		t.Fatal("expected generated jwt secret, got empty value")
	}

	secretFile, err := runtimeJWTSecretPath(cfg)
	if err != nil {
		t.Fatalf("runtimeJWTSecretPath returned error: %v", err)
	}
	data, err := os.ReadFile(secretFile)
	if err != nil {
		t.Fatalf("read runtime secret file: %v", err)
	}
	if strings.TrimSpace(string(data)) != cfg.Security.JWT.Secret {
		t.Fatalf("expected persisted secret %q, got %q", cfg.Security.JWT.Secret, strings.TrimSpace(string(data)))
	}
}

func TestGetConfigReusesPersistedJWTSecretWhenEmpty(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	configFile := writeTestConfig(t, filepath.Join(root, "config"), "", "")
	seedCfg := &Config{}
	secretFile, err := runtimeJWTSecretPath(seedCfg)
	if err != nil {
		t.Fatalf("runtimeJWTSecretPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(secretFile), 0o755); err != nil {
		t.Fatalf("mkdir runtime secret dir: %v", err)
	}
	if err := os.WriteFile(secretFile, []byte("persisted-secret\n"), 0o600); err != nil {
		t.Fatalf("write runtime secret file: %v", err)
	}

	cfg, err := getConfig(configFile)
	if err != nil {
		t.Fatalf("getConfig returned error: %v", err)
	}

	if cfg.Security.JWT.Secret != "persisted-secret" {
		t.Fatalf("expected persisted secret to be reused, got %q", cfg.Security.JWT.Secret)
	}
}

func TestGetConfigUsesDefaultRuntimeSecretPathEvenWithCustomConfigLocation(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	configFile := writeTestConfig(t, filepath.Join(root, "etc", "tudou"), "", "")

	cfg, err := getConfig(configFile)
	if err != nil {
		t.Fatalf("getConfig returned error: %v", err)
	}

	secretFile := filepath.Join(root, "storage", "runtime", "jwt_secret")
	data, err := os.ReadFile(secretFile)
	if err != nil {
		t.Fatalf("read runtime secret file: %v", err)
	}
	if strings.TrimSpace(string(data)) != cfg.Security.JWT.Secret {
		t.Fatalf("expected secret persisted at default runtime path, got %q", strings.TrimSpace(string(data)))
	}
}

func TestGetConfigUsesConfiguredJWTSecretFile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	secretFile := filepath.Join(root, "secrets", "jwt.secret")
	configFile := writeTestConfig(t, filepath.Join(root, "config"), "", secretFile)

	cfg, err := getConfig(configFile)
	if err != nil {
		t.Fatalf("getConfig returned error: %v", err)
	}

	data, err := os.ReadFile(secretFile)
	if err != nil {
		t.Fatalf("read configured runtime secret file: %v", err)
	}
	if strings.TrimSpace(string(data)) != cfg.Security.JWT.Secret {
		t.Fatalf("expected secret persisted at configured path, got %q", strings.TrimSpace(string(data)))
	}
}

func writeTestConfig(t *testing.T, configDir string, secret string, secretFile string) string {
	t.Helper()

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	content := "security:\n  jwt:\n    secret: " + secret + "\n"
	if secret == "" {
		content = "security:\n  jwt:\n    secret: \"\"\n"
	}
	if secretFile != "" {
		content += "    secret_file: \"" + filepath.ToSlash(secretFile) + "\"\n"
	}
	if err := os.WriteFile(configFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return configFile
}
