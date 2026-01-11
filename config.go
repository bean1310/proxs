package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/kevinburke/ssh_config"
)

type Config struct {
	ListenPort int                 `toml:"port"`
	Proxies    map[string]sshProxy `toml:"proxy"`
}

func makeNestedSshConnection(host string) (*sshConnection, error) {

	result := &sshConnection{}
	var err error

	// If `SSH_CONFIG_FILE` is set, use it; otherwise, use the default location.
	configPath := os.Getenv("SSH_CONFIG_FILE")
	if configPath == "" {
		configPath = filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, err
	}

	// Check for ProxyJump
	jumpHost, _ := cfg.Get(host, "ProxyJump")
	if jumpHost != "" {
		// If ProxyJump is specified, create nested sshConnection
		result.JumpHost, err = makeNestedSshConnection(jumpHost)
		if err != nil {
			return nil, err
		}
	}

	result.HostName, err = cfg.Get(host, "HostName")
	if err != nil {
		slog.Error("Failed to get HostName from ssh config", "host", host, "error", err)
		return nil, err
	}

	portStr, err := cfg.Get(host, "Port")
	if err != nil {
		slog.Error("Failed to get Port from ssh config", "host", host, "error", err)
		return nil, err
	}
	if portStr == "" {
		portStr = "22" // Default SSH port
	}
	result.Port, err = strconv.Atoi(portStr)
	if err != nil {
		slog.Error("Failed to convert Port to integer", "host", host, "port", portStr, "error", err)
		return nil, err
	}

	result.User, err = cfg.Get(host, "User")
	if err != nil {
		slog.Error("Failed to get User from ssh config", "host", host, "error", err)
		return nil, err
	}

	return result, nil
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
	slog.Debug("Configuration loaded", "config", config)

	for key := range config.Proxies {
		proxy := config.Proxies[key]
		proxy.Connection, err = makeNestedSshConnection(proxy.Host)
		if err != nil {
			slog.Error("Failed to create sshConnection from ssh config", "host", proxy.Host, "error", err)
			return nil, err
		}
		config.Proxies[key] = proxy
	}
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
