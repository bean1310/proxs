package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
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

func GetSocksAddrport(cfg *Config, domain string) string {
	for _, proxy := range cfg.Proxies {
		if proxy.TargetAddrs[0] == domain {
			return fmt.Sprintf("127.0.0.1:%d", proxy.Port)
		}
	}
	log.Printf("No proxy found for domain %s", domain)
	return "0.0.0.0:0"
}
