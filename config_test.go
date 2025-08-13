package main

import "testing"

func TestGetSocksAddrport(t *testing.T) {
	cfg := &Config{
		Proxies: map[string]sshProxy{
			"example": {Port: 1080, TargetAddrs: []string{"example.com", "example.org", "example*"}},
		},
	}

	addr, port, err := GetSocksAddrport(cfg, "example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if addr != "127.0.0.1" {
		t.Fatalf("Expected address 127.0.0.1, got %v", addr)
	}
	if port != 1080 {
		t.Fatalf("Expected port 1080, got %v", port)
	}

	addr, port, err = GetSocksAddrport(cfg, "example.org")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if addr != "127.0.0.1" {
		t.Fatalf("Expected address 127.0.0.1, got %v", addr)
	}
	if port != 1080 {
		t.Fatalf("Expected port 1080, got %v", port)
	}

	addr, port, err = GetSocksAddrport(cfg, "example.net")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if addr != "127.0.0.1" {
		t.Fatalf("Expected address 127.0.0.1, got %v", addr)
	}
	if port != 1080 {
		t.Fatalf("Expected port 1080, got %v", port)
	}
}
