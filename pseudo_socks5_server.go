package main

import (
	"bufio"
	"io"
	"log"
	"net"
)

// https://datatracker.ietf.org/doc/html/rfc1928#autoid-3
//
//	+----+----------+----------+
//	|VER | NMETHODS | METHODS  |
//	+----+----------+----------+
//	| 1  |    1     | 1 to 255 |
//	+----+----------+----------+
type AuthMethods struct {
	Ver      byte
	NMethods byte
	Methods  []byte
}

func ParseAuthMethod(r io.Reader) (am AuthMethods, err error) {
	var buf [2]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return am, err
	}
	am.Ver = buf[0]
	am.NMethods = buf[1]

	if am.NMethods > 255 {
		return am, io.ErrShortBuffer
	}

	am.Methods = make([]byte, am.NMethods)
	if _, err := io.ReadFull(r, am.Methods); err != nil {
		return am, err
	}
	return am, nil
}

type Request struct {
	Ver      byte
	Command  byte
	AddrType byte
	DestAddr string
	DestPort uint16
}

type Reply struct {
	Ver      byte
	Rep      byte
	AddrType byte
	BndAddr  [4]byte
	BndPort  uint16
}

func ParseRequest(r io.Reader) (Request, error) {
	var req Request
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return req, err
	}
	req.Ver = buf[0]
	req.Command = buf[1]
	req.AddrType = buf[3]

	if req.Ver != 5 {
		return Request{}, io.ErrUnexpectedEOF
	}

	// Under prototype, we handle only CONNECT command.
	if req.Command != 1 {
		return Request{}, io.ErrUnexpectedEOF
	}

	// under prototype, we handle only domain address types.
	switch req.AddrType {
	case byte(0x01), byte(0x03):
		tmp := make([]byte, 1)
		if _, err := io.ReadFull(r, tmp); err != nil {
			return Request{}, err
		}
		domainLen := tmp[0]
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(r, domain); err != nil {
			return Request{}, err
		}
		req.DestAddr = string(domain)
		port := make([]byte, 2)
		if _, err := io.ReadFull(r, port); err != nil {
			return Request{}, err
		}
		req.DestPort = uint16(port[0])<<8 | uint16(port[1])
	default:
		return Request{}, io.EOF
	}

	return req, nil
}

func socksConnection(src net.Conn, cfg *Config) (destAddr string, destPort uint16, err error) {
	buffer := bufio.NewReader(src)

	am, err := ParseAuthMethod(buffer)
	if err != nil {
		log.Printf("Failed to parse authentication method: %v", err)
		return
	}

	if am.Ver != 5 {
		log.Printf("Unsupported SOCKS version: %d", am.Ver)
		return
	}

	if am.NMethods == 0 || len(am.Methods) == 0 {
		log.Println("No authentication methods provided")
		return
	}

	src.Write([]byte{5, 0}) // Reply with no authentication required

	request, err := ParseRequest(buffer)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		return
	}

	log.Printf("Received request: %+v", request)

	// Create reply request
	rep := Reply{
		Ver:      5,
		Rep:      0, // Success
		AddrType: byte(0x01),
		BndAddr:  [4]byte{0, 0, 0, 0},
		BndPort:  0,
	}

	if _, err = src.Write([]byte{rep.Ver, rep.Rep, byte(0x00), rep.AddrType, rep.BndAddr[0], rep.BndAddr[1], rep.BndAddr[2], rep.BndAddr[3], byte(rep.BndPort >> 8), byte(rep.BndPort & 0xff)}); err != nil {
		log.Printf("Failed to send reply: %v", err)
		return
	}
	return request.DestAddr, request.DestPort, nil
}
