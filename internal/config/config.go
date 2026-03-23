package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvConfigDir = "MYTASK_CONFIG_DIR"
	DefaultDir   = ".mytask"
)

type Config struct {
	ConfigDir string
	HooksDir  string
	DBPath    string
}

func Load() (*Config, error) {
	configDir, err := findConfigDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg := &Config{
		ConfigDir: configDir,
		HooksDir:  filepath.Join(configDir, "hooks"),
		DBPath:    filepath.Join(configDir, "tasks.db"),
	}

	if err := os.MkdirAll(cfg.HooksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create hooks directory: %w", err)
	}

	return cfg, nil
}

func findConfigDir() (string, error) {
	if envDir := os.Getenv(EnvConfigDir); envDir != "" {
		return envDir, nil
	}

	// 默认使用当前目录下的 .mytask 隐藏目录
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return filepath.Join(cwd, DefaultDir), nil
}