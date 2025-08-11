package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/net/proxy"
)

func handleConnection(src net.Conn, cfg *Config) {
	defer src.Close()

	destAddr, destPort, err := socksConnection(src, cfg)
	if err != nil {
		log.Printf("Failed to establish SOCKS connection: %v", err)
		return
	}

	var targetAddr string
	targetAddr = GetSocksAddrport(cfg, destAddr)

	pd, err := proxy.SOCKS5("tcp", targetAddr, nil, proxy.Direct)
	if err != nil {
		log.Printf("Failed to create SOCKS5 proxy: %v", err)
		return
	}

	dst, err := pd.Dial("tcp", net.JoinHostPort(destAddr, fmt.Sprintf("%d", destPort)))
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", targetAddr, err)
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

	cfg := LoadConfig()
	if cfg != nil {
		listenAddr = fmt.Sprintf("127.0.0.1:%d", cfg.ListenPort)
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}
	// log.Printf("Listening on %s, forwarding to %s", listenAddr, targetAddrs[])

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn, cfg)
	}
}
