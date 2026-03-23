package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultDir(t *testing.T) {
	// 清除环境变量
	os.Unsetenv(EnvConfigDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer os.RemoveAll(cfg.ConfigDir)

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, DefaultDir)
	if cfg.ConfigDir != expected {
		t.Errorf("Expected config dir '%s', got '%s'", expected, cfg.ConfigDir)
	}

	// 验证子路径
	if cfg.DBPath != filepath.Join(expected, "tasks.db") {
		t.Errorf("Unexpected DBPath: %s", cfg.DBPath)
	}
	if cfg.HooksDir != filepath.Join(expected, "hooks") {
		t.Errorf("Unexpected HooksDir: %s", cfg.HooksDir)
	}
}

func TestLoad_EnvVar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv(EnvConfigDir, tmpDir)
	defer os.Unsetenv(EnvConfigDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.ConfigDir != tmpDir {
		t.Errorf("Expected config dir '%s', got '%s'", tmpDir, cfg.ConfigDir)
	}

	// 验证目录被创建
	if _, err := os.Stat(cfg.HooksDir); os.IsNotExist(err) {
		t.Error("Hooks directory was not created")
	}
}

func TestLoad_CreatesDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 使用不存在的子目录
	configPath := filepath.Join(tmpDir, "mytask")
	os.Setenv(EnvConfigDir, configPath)
	defer os.Unsetenv(EnvConfigDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 验证目录被创建
	if _, err := os.Stat(cfg.ConfigDir); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}
	if _, err := os.Stat(cfg.HooksDir); os.IsNotExist(err) {
		t.Error("Hooks directory was not created")
	}
}