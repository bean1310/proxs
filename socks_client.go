package main

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/net/proxy"
)

func createSocksConnection(socksServerAddr string, socksServerPort uint16, destAddr string, destPort uint16) (net.Conn, error) {
	pd, err := proxy.SOCKS5("tcp", net.JoinHostPort(socksServerAddr, fmt.Sprintf("%d", socksServerPort)), nil, proxy.Direct)
	if err != nil {
		log.Printf("Failed to create SOCKS5 proxy: %v", err)
		return nil, err
	}

	dst, err := pd.Dial("tcp", net.JoinHostPort(destAddr, fmt.Sprintf("%d", destPort)))
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", destAddr, err)
		return nil, err
	}

	return dst, nil
}
