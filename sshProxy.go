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
}

func sshProxySelectFrom(addr string, proxies []sshProxy) (sshProxy, error) {
	for _, proxy := range proxies {
		for _, targetAddr := range proxy.TargetAddrs {
			match, err := filepath.Match(targetAddr, addr)
			if err != nil {
				slog.Error("Error matching domain with target address", "domain", addr, "targetAddr", targetAddr, "error", err)
				return sshProxy{}, err
			}
			if match {
				slog.Info("Matched proxy for domain", "domain", addr, "address", proxy)
				return proxy, nil
			}
		}
	}
	slog.Warn("No proxy found for domain", "domain", addr)
	return sshProxy{}, fmt.Errorf("no matching proxy found for address: %s", addr)
}

// This function dials an SSH connection recursively through jump hosts.
// Returns the SSH client and a cleanup function that closes all connections.
func (sc *sshConnection) Dial(network, addr string) (*ssh.Client, func(), error) {
	if sc.JumpHost == nil {
		config, cleanup, err := authFromAgent()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create auth from agent: %w", err)
		}
		defer cleanup()

		sshConfig := &ssh.ClientConfig{
			User:            sc.User,
			Auth:            []ssh.AuthMethod{config},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
		}
		slog.Info("Dialing SSH connection", "hostname", sc.HostName, "port", sc.Port)
		conn, err := ssh.Dial(network, fmt.Sprintf("%s:%d", sc.HostName, sc.Port), sshConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to dial SSH connection: %w", err)
		}

		return conn, func() { conn.Close() }, nil
	} else {
		jumpClient, jumpCleanup, err := sc.JumpHost.Dial(network, "")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to dial jump host: %w", err)
		}
		ncc, err := jumpClient.Dial(network, fmt.Sprintf("%s:%d", sc.HostName, sc.Port))
		if err != nil {
			jumpCleanup()
			return nil, nil, fmt.Errorf("failed to dial target host through jump host: %w", err)
		}

		config, cleanup, err := authFromAgent()
		if err != nil {
			jumpCleanup()
			return nil, nil, fmt.Errorf("failed to create auth from agent: %w", err)
		}
		defer cleanup()

		conn, chans, reqs, err := ssh.NewClientConn(ncc, fmt.Sprintf("%s:%d", sc.HostName, sc.Port), &ssh.ClientConfig{
			User:            sc.User,
			Auth:            []ssh.AuthMethod{config},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // This code is insecure; use a proper host key callback in production
		})
		if err != nil {
			jumpCleanup()
			return nil, nil, fmt.Errorf("failed to create new SSH client connection: %w", err)
		}

		client := ssh.NewClient(conn, chans, reqs)
		// cleanupAll closes this connection and recursively closes all jump host connections
		cleanupAll := func() {
			client.Close()
			jumpCleanup() // This recursively closes all jump host connections in the chain
		}
		return client, cleanupAll, nil
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
