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

type sshConnection struct {
	HostName string
	User     string
	Port     int
	JumpHost *sshConnection
}

type sshProxy struct {
	Host        string   `toml:"host"`
	TargetAddrs []string `toml:"target_addrs"`
	Connection  *sshConnection
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
				slog.Info("Matched proxy for domain", "domain", addr, "address", proxy)
				return &proxy, nil
			}
		}
	}
	slog.Warn("No proxy found for domain", "domain", addr)
	return nil, fmt.Errorf("no matching proxy found for address: %s", addr)
}

// This function dials an SSH connection recursively through jump hosts.
func (sc *sshConnection) Dial(network, addr string) (*ssh.Client, error) {
	if sc.JumpHost == nil {
		config, cleanup, err := authFromAgent()
		if err != nil {
			return nil, fmt.Errorf("failed to create auth from agent: %w", err)
		}
		defer cleanup()

		sshConfig := &ssh.ClientConfig{
			User:            sc.User,
			Auth:            []ssh.AuthMethod{config},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
		}
		slog.Info("Dialing SSH connection", "hostname", sc.HostName, "port", sc.Port)
		return ssh.Dial(network, fmt.Sprintf("%s:%d", sc.HostName, sc.Port), sshConfig)
	} else {
		jumpClient, err := sc.JumpHost.Dial(network, "")
		if err != nil {
			return nil, fmt.Errorf("failed to dial jump host: %w", err)
		}
		ncc, err := jumpClient.Dial(network, fmt.Sprintf("%s:%d", sc.HostName, sc.Port))
		if err != nil {
			return nil, fmt.Errorf("failed to dial target host through jump host: %w", err)
		}

		config, cleanup, err := authFromAgent()
		if err != nil {
			return nil, fmt.Errorf("failed to create auth from agent: %w", err)
		}
		defer cleanup()

		conn, chans, reqs, err := ssh.NewClientConn(ncc, fmt.Sprintf("%s:%d", sc.HostName, sc.Port), &ssh.ClientConfig{
			User:            sc.User,
			Auth:            []ssh.AuthMethod{config},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create new SSH client connection: %w", err)
		}
		return ssh.NewClient(conn, chans, reqs), nil
	}
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

// func (p *sshProxy) CreateSshConfig() (*ssh.ClientConfig, func(), error) {
// 	auth, cleanup, err := authFromAgent()
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to create auth from agent: %w", err)
// 	}

// 	return &ssh.ClientConfig{
// 		User:            p.User,
// 		Auth:            []ssh.AuthMethod{auth},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
// 	}, cleanup, nil
// }

// func (p *sshProxy) Activate() (deepClient, error) {
// 	if p.sshClient != nil {
// 		return p.sshClient, nil
// 	}
// 	config, cleanup, err := p.CreateSshConfig()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if cleanup != nil {
// 		p.cleanupFunc = cleanup
// 	} else {
// 		p.cleanupFunc = func() {}
// 	}

// 	slog.Info("Activating SSH proxy", "hostname", p.HostName, "port", p.Port)
// 	p.sshClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", p.HostName, p.Port), config)
// 	if err != nil {
// 		slog.Error("Failed to connect to SSH proxy", "hostname", p.HostName, "port", p.Port, "error", err)
// 		return nil, err
// 	}
// 	return p.sshClient, nil
// }

// func (p *sshProxy) Deactivate() error {
// 	if p.cleanupFunc != nil {
// 		p.cleanupFunc()
// 		p.cleanupFunc = nil
// 	}
// 	if p.sshClient != nil {
// 		err := p.sshClient.Close()
// 		if err != nil {
// 			return err
// 		}
// 		p.sshClient = nil
// 	}
// 	return nil
// }
