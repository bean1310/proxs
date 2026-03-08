package main

import (
	"bytes"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestParseAuthMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected AuthMethods
		wantErr  bool
	}{
		{
			name:  "Valid auth method with single method",
			input: []byte{5, 1, 0}, // Ver=5, NMethods=1, Methods=[0]
			expected: AuthMethods{
				Ver:      5,
				NMethods: 1,
				Methods:  []byte{0},
			},
			wantErr: false,
		},
		{
			name:  "Valid auth method with multiple methods",
			input: []byte{5, 3, 0, 1, 2}, // Ver=5, NMethods=3, Methods=[0,1,2]
			expected: AuthMethods{
				Ver:      5,
				NMethods: 3,
				Methods:  []byte{0, 1, 2},
			},
			wantErr: false,
		},
		{
			name:    "Invalid input - too short header",
			input:   []byte{5}, // Only version, missing NMethods
			wantErr: true,
		},
		{
			name:    "Invalid input - missing methods",
			input:   []byte{5, 2, 0}, // Ver=5, NMethods=2, but only 1 method
			wantErr: true,
		},
		{
			name:    "Empty input",
			input:   []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			result, err := ParseAuthMethod(reader)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAuthMethod() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseAuthMethod() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseAuthMethod() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Request
		wantErr  bool
	}{
		{
			name: "Valid CONNECT request with domain",
			// Ver=5, Cmd=1, Reserved=0, AddrType=3, DomainLen=11, Domain="example.com", Port=80
			input: []byte{5, 1, 0, 3, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80},
			expected: Request{
				Ver:      5,
				Command:  1,
				AddrType: 3,
				DestAddr: "example.com",
				DestPort: 80,
			},
			wantErr: false,
		},
		{
			name: "Valid CONNECT request with IPv4",
			// Ver=5, Cmd=1, Reserved=0, AddrType=1, DomainLen=11, Domain="example.com", Port=443
			input: []byte{5, 1, 0, 1, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 1, 187},
			expected: Request{
				Ver:      5,
				Command:  1,
				AddrType: 1,
				DestAddr: "example.com",
				DestPort: 443,
			},
			wantErr: false,
		},
		{
			name:    "Invalid version",
			input:   []byte{4, 1, 0, 3, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80},
			wantErr: true,
		},
		{
			name:    "Invalid command (not CONNECT)",
			input:   []byte{5, 2, 0, 3, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80},
			wantErr: true,
		},
		{
			name:    "Unsupported address type",
			input:   []byte{5, 1, 0, 4, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80},
			wantErr: true,
		},
		{
			name:    "Too short input",
			input:   []byte{5, 1, 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			result, err := ParseRequest(reader)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRequest() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRequest() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseRequest() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}

func TestSocksConnection(t *testing.T) {
	// Create a mock config
	cfg := &Config{
		ListenPort: 1080,
		Proxies:    make(map[string]sshProxy),
	}

	tests := []struct {
		name         string
		clientData   []byte
		expectedAddr string
		expectedPort uint16
		wantErr      bool
	}{
		{
			name:       "Invalid SOCKS version in auth",
			clientData: []byte{4, 1, 0}, // Wrong version
			wantErr:    true,
		},
		{
			name:       "No authentication methods",
			clientData: []byte{5, 0}, // NMethods=0
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pair of connected pipes to simulate network connection
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			// Start socksConnection in a goroutine
			resultChan := make(chan struct {
				addr string
				port uint16
				err  error
			}, 1)

			go func() {
				addr, port, err := socksConnection(server, cfg)
				resultChan <- struct {
					addr string
					port uint16
					err  error
				}{addr, port, err}
			}()

			// Send client data
			go func() {
				defer client.Close()
				client.Write(tt.clientData)
			}()

			// Wait for result with timeout
			select {
			case result := <-resultChan:
				if tt.wantErr {
					if result.err == nil {
						t.Errorf("socksConnection() expected error, but got none")
					}
					return
				}

				if result.err != nil {
					t.Errorf("socksConnection() unexpected error: %v", result.err)
					return
				}

				if result.addr != tt.expectedAddr {
					t.Errorf("socksConnection() addr = %s, expected %s", result.addr, tt.expectedAddr)
				}

				if result.port != tt.expectedPort {
					t.Errorf("socksConnection() port = %d, expected %d", result.port, tt.expectedPort)
				}

			case <-time.After(5 * time.Second):
				t.Error("socksConnection() timed out")
			}
		})
	}
}

// Helper function to create test data for SOCKS5 requests
func createSOCKS5Request(domain string, port uint16) []byte {
	domainBytes := []byte(domain)
	domainLen := byte(len(domainBytes))

	request := []byte{5, 1, 0, 3} // Ver=5, Cmd=1(CONNECT), Reserved=0, AddrType=3(Domain)
	request = append(request, domainLen)
	request = append(request, domainBytes...)
	request = append(request, byte(port>>8), byte(port&0xff)) // Port in network byte order

	return request
}

func TestCreateSOCKS5Request(t *testing.T) {
	// Test the helper function itself
	result := createSOCKS5Request("google.com", 443)
	expected := []byte{5, 1, 0, 3, 10, 'g', 'o', 'o', 'g', 'l', 'e', '.', 'c', 'o', 'm', 1, 187}

	if !bytes.Equal(result, expected) {
		t.Errorf("createSOCKS5Request() = %v, expected %v", result, expected)
	}
}

// Benchmark tests
func BenchmarkParseAuthMethod(b *testing.B) {
	data := []byte{5, 3, 0, 1, 2}

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		ParseAuthMethod(reader)
	}
}

func BenchmarkParseRequest(b *testing.B) {
	data := []byte{5, 1, 0, 3, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80}

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		ParseRequest(reader)
	}
}
