package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Config struct {
	ListenPort int                 `toml:"port"`
	Proxies    map[string]sshProxy `toml:"proxy"`
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	configDir, err := configDir("proxs")
	if err != nil {
		slog.Error("Failed to get configuration directory", "error", err)
		return nil, err
	}

	if _, err := toml.DecodeFile(filepath.Join(configDir, "config.toml"), config); err != nil {
		slog.Error("Failed to load configuration file", "file", "config.toml", "error", err)
		return nil, err
	}
	slog.Info("Configuration loaded", "config", config)
	return config, nil
}

func configDir(app string) (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, app), nil
	}

	// Fallback to user config directory
	base, err := os.UserConfigDir() // macOS: ~/Library/Application Support
	if err != nil {
		return "", err
	}
	return filepath.Join(base, app), nil
}

func GetSocksAddrport(cfg *Config, domain string) (string, uint16, error) {
	for _, proxy := range cfg.Proxies {
		for _, targetAddr := range proxy.TargetAddrs {
			match, err := filepath.Match(targetAddr, domain)
			if err != nil {
				slog.Error("Error matching domain with target address", "domain", domain, "targetAddr", targetAddr, "error", err)
				return "", 0, err
			}
			if match {
				slog.Info("Matched proxy for domain", "domain", domain, "address", "127.0.0.1", "port", proxy.Port)
				return "127.0.0.1", uint16(proxy.Port), nil
			}
		}
	}
	slog.Warn("No proxy found for domain", "domain", domain)
	return "0.0.0.0", 0, errors.New("no proxy found")
}
