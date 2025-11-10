package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/net/proxy"

	"github.com/kevinburke/ssh_config"
)

type deepClient interface {
	Dial(network, addr string) (net.Conn, error)
	Close() error
}

type Socks5Client struct {
	d proxy.Dialer
}

func (c *Socks5Client) Close() error {
	return nil
}

func (c *Socks5Client) Dial(network, addr string) (net.Conn, error) {
	return c.d.Dial(network, addr)
}

func NewSocks5Client(network, address string, auth *proxy.Auth, forward proxy.Dialer) (deepClient, error) {
	dialer, err := proxy.SOCKS5(network, address, auth, forward)
	if err != nil {
		return nil, err
	}
	return &Socks5Client{d: dialer}, nil
}

type sshProxy struct {
	HostName     string `toml:"hostname"`
	User         string
	Port         int
	TargetAddrs  []string `toml:"target_addrs"`
	UseSshClient bool     `toml:"use_ssh_client"`
	sshClient    deepClient
	cleanupFunc  func()
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

	user, err := ssh_config.GetStrict(p.HostName, "User")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user from ssh config: %w", err)
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
	}, cleanup, nil
}

func (p *sshProxy) Activate() (deepClient, error) {
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

	jumpConn := net.Conn(nil)
	// If defined 'ProxyJump', create base ssh connection first.
	proxyJump, err := ssh_config.GetStrict(p.HostName, "ProxyJump")
	if err == nil && proxyJump != "" {
		slog.Info("Using ProxyJump for SSH proxy", "proxyJump", proxyJump)
		jumpHostName, err := ssh_config.GetStrict(proxyJump, "HostName")
		if err != nil {
			return nil, fmt.Errorf("failed to get jump host name: %w", err)
		}
		jumpUser, err := ssh_config.GetStrict(proxyJump, "User")
		if err != nil {
			return nil, fmt.Errorf("failed to get jump user: %w", err)
		}
		jumpPortStr, err := ssh_config.GetStrict(proxyJump, "Port")
		if err != nil {
			return nil, fmt.Errorf("failed to get jump port: %w", err)
		}
		jumpPort, err := strconv.Atoi(jumpPortStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert jump port to int: %w", err)
		}

		jumpConfig := &ssh.ClientConfig{
			User:            jumpUser,
			Auth:            config.Auth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		slog.Info("Activating jump SSH proxy", "hostname", jumpHostName, "port", jumpPort)
		jumpClient, err := ssh.Dial("tcp", proxyJump, jumpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to jump proxy: %w", err)
		}

		hostname, err := ssh_config.GetStrict(p.HostName, "HostName")
		if err != nil {
			return nil, fmt.Errorf("failed to get host name: %w", err)
		}
		port, err := ssh_config.GetStrict(p.HostName, "Port")
		if err != nil {
			return nil, fmt.Errorf("failed to get port: %w", err)
		}
		jumpConn, err = jumpClient.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to jump host: %w", err)
		}
	}

	if p.UseSshClient {
		socksClient, err := NewSocks5Client("tcp", fmt.Sprintf("%s:%d", p.HostName, p.Port), nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return socksClient, nil
	}

	slog.Info("Activating SSH proxy", "hostname", p.HostName, "port", p.Port)
	if jumpConn != nil {
		sshClient, _, _, err := ssh.NewClientConn(jumpConn, fmt.Sprintf("%s:%d", p.HostName, p.Port), config)
		if err != nil {
			slog.Error("Failed to connect to SSH proxy via jump host", "hostname", p.HostName, "port", p.Port, "error", err)
			return nil, err
		}
		p.sshClient = ssh.NewClient(sshClient, nil, nil)
	} else {
		p.sshClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", p.HostName, p.Port), config)
	}
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
