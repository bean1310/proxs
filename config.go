package main

import (
	"log"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Proxy struct {
	Port        int      `toml:"port"`
	TargetAddrs []string `toml:"target_addrs"`
}

type Config struct {
	ListenPort int              `toml:"port"`
	Proxies    map[string]Proxy `toml:"proxy"`
}

func LoadConfig() *Config {
	config := &Config{}
	if _, err := toml.DecodeFile("./config.toml", config); err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return nil
	}
	log.Printf("Loaded configuration: %+v", config)
	return config
}

func GetSocksAddrport(cfg *Config, domain string) (string, uint16, error) {
	for _, proxy := range cfg.Proxies {
		if proxy.TargetAddrs[0] == domain {
			return "127.0.0.1", uint16(proxy.Port), nil
		}
	}
	log.Printf("No proxy found for domain %s", domain)
	return "0.0.0.0", 0, errors.New("no proxy found")
}
