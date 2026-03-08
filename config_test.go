package main

import (
	"os"
	"testing"
)

func TestMakeNestedSshConnection(t *testing.T) {
	// Create a temporary ssh config file
	tmpfile, err := os.CreateTemp("", "ssh_config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test data to the temporary ssh config file
	content := `
Host testhost
    HostName example.com
    User testuser

Host testhost_with_jump
	HostName jump.example.com
	User jumpuser
	Port 2200
	ProxyJump testhost
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Define SSH_CONFIG_FILE environment variable to point to the temporary file
	os.Setenv("SSH_CONFIG_FILE", tmpfile.Name())
	defer os.Unsetenv("SSH_CONFIG_FILE")

	// Execute the test
	conn, err := makeNestedSshConnection("testhost")
	if err != nil {
		t.Fatal(err)
	}

	// Simple host test
	if conn.HostName != "example.com" {
		t.Errorf("expected example.com, got %s", conn.HostName)
	}

	if conn.User != "testuser" {
		t.Errorf("expected testuser, got %s", conn.User)
	}

	if conn.Port != 22 {
		t.Errorf("expected port 22, got %d", conn.Port)
	}

	// Test with ProxyJump
	connWithJump, err := makeNestedSshConnection("testhost_with_jump")
	if err != nil {
		t.Fatal(err)
	}

	if connWithJump.HostName != "jump.example.com" {
		t.Errorf("expected jump.example.com, got %s", connWithJump.HostName)
	}

	if connWithJump.User != "jumpuser" {
		t.Errorf("expected jumpuser, got %s", connWithJump.User)
	}

	if connWithJump.Port != 2200 {
		t.Errorf("expected port 2200, got %d", connWithJump.Port)
	}

	if connWithJump.JumpHost == nil {
		t.Fatal("expected JumpHost to be set, but it is nil")
	}

	if connWithJump.JumpHost.HostName != "example.com" {
		t.Errorf("expected jump host example.com, got %s", connWithJump.JumpHost.HostName)
	}

	if connWithJump.JumpHost.User != "testuser" {
		t.Errorf("expected jump host user testuser, got %s", connWithJump.JumpHost.User)
	}

	if connWithJump.JumpHost.Port != 22 {
		t.Errorf("expected jump host port 22, got %d", connWithJump.JumpHost.Port)
	}
}
