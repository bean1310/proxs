package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
)

func handleConnection(src net.Conn, cfg *Config) {
	defer src.Close()

	destAddr, destPort, err := socksConnection(src, cfg)
	if err != nil {
		log.Printf("Failed to establish SOCKS connection: %v", err)
		return
	}

	socksAddr, socksPort, err := GetSocksAddrport(cfg, destAddr)
	if err != nil {
		log.Printf("Failed to get SOCKS address and port: %v", err)
		return
	}
	dst, err := createSocksConnection(socksAddr, socksPort, destAddr, destPort)
	if err != nil {
		log.Printf("Failed to create SOCKS connection: %v", err)
		return
	}
	defer dst.Close()

	// 双方向コピー
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

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Warn("Failed to accept connection", "error", err)
			continue
		}
		go handleConnection(conn, cfg)
	}
}
