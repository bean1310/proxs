package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type sshProxy struct {
	HostName    string   `toml:"hostname"`
	User        string   `toml:"user"`
	Port        int      `toml:"port"`
	TargetAddrs []string `toml:"target_addrs"`
	sshClient   *ssh.Client
	cleanupFunc func()
}

func sshProxySelectFrom(addr string, proxies []sshProxy) (*sshProxy, error) {
	for _, proxy := range proxies {
		for _, targetAddr := range proxy.TargetAddrs {
			match, err := filepath.Match(targetAddr, addr)
			if err != nil {
				slog.Error("Error matching domain with target address", "domain", addr, "targetAddr", targetAddr, "error", err)
				return nil, err
			}
			if match {
				slog.Info("Matched proxy for domain", "domain", addr, "address", proxy, "port", proxy.Port)
				return &proxy, nil
			}
		}
	}
	slog.Warn("No proxy found for domain", "domain", addr)
	return nil, fmt.Errorf("no matching proxy found for address: %s", addr)
}

func (p *sshProxy) Dial(network, addr string) (net.Conn, error) {
	client, err := p.Activate()
	if err != nil {
		return nil, err
	}
	return client.Dial(network, addr)
}

func authFromAgent() (ssh.AuthMethod, func(), error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil, fmt.Errorf("SSH_AUTH_SOCK is empty")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil, err
	}
	ag := agent.NewClient(conn)
	return ssh.PublicKeysCallback(ag.Signers), func() { conn.Close() }, nil
}

func (p *sshProxy) CreateSshConfig() (*ssh.ClientConfig, func(), error) {
	auth, cleanup, err := authFromAgent()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create auth from agent: %w", err)
	}

	return &ssh.ClientConfig{
		User:            p.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
	}, cleanup, nil
}

func (p *sshProxy) Activate() (*ssh.Client, error) {
	if p.sshClient != nil {
		return p.sshClient, nil
	}
	config, cleanup, err := p.CreateSshConfig()
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		p.cleanupFunc = cleanup
	} else {
		p.cleanupFunc = func() {}
	}
	slog.Info("Activating SSH proxy", "hostname", p.HostName, "port", p.Port)
	p.sshClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", p.HostName, p.Port), config)
	if err != nil {
		slog.Error("Failed to connect to SSH proxy", "hostname", p.HostName, "port", p.Port, "error", err)
		return nil, err
	}
	return p.sshClient, nil
}

func (p *sshProxy) Deactivate() error {
	if p.cleanupFunc != nil {
		p.cleanupFunc()
		p.cleanupFunc = nil
	}
	if p.sshClient != nil {
		err := p.sshClient.Close()
		if err != nil {
			return err
		}
		p.sshClient = nil
	}
	return nil
}
