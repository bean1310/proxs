package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"maps"
	"net"
	"slices"
)

func handleConnection(src net.Conn, proxies []sshProxy, cfg *Config) {
	defer src.Close()

	destAddr, destPort, err := socksConnection(src, cfg)
	if err != nil {
		log.Printf("Failed to establish SOCKS connection: %v", err)
		return
	}

	sp, err := sshProxySelectFrom(destAddr, proxies)
	if err != nil {
		log.Printf("Failed to select SSH proxy: %v", err)
		return
	}

	dst, err := sp.Dial("tcp", net.JoinHostPort(destAddr, fmt.Sprintf("%d", destPort)))
	if err != nil {
		slog.Error("Failed to create destination connection", "address", destAddr, "port", destPort, "error", err)
		return
	}
	defer dst.Close()
	defer sp.Deactivate()

	go func() {
		_, err := io.Copy(dst, src)
		if err != nil {
			log.Printf("Error copying from src to dst: %v", err)
		}
		dst.Close()
	}()
	_, err = io.Copy(src, dst)
	if err != nil {
		log.Printf("Error copying from dst to src: %v", err)
	}
}

func main() {
	var listenAddr string

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		return
	}
	listenAddr = net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", cfg.ListenPort))

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		slog.Error("Failed to listen on address", "address", listenAddr, "error", err)
		return
	}

	proxies := slices.Collect(maps.Values(cfg.Proxies))

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Warn("Failed to accept connection", "error", err)
			continue
		}
		go handleConnection(conn, proxies, cfg)
	}
}
